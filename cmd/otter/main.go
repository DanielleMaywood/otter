package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/DanielleMaywood/otter/internal/engine/pgengine"
	"github.com/DanielleMaywood/otter/internal/printer"
	"github.com/DanielleMaywood/otter/internal/printer/pgprinter"
	"github.com/jackc/pgx/v5"
)

type Config struct {
	Database string
	Queries  string

	Overrides printer.TypeOverrides `toml:"overrides"`

	Package struct {
		Name string
		Path string
	}
}

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	configFile, err := os.ReadFile("otter.toml")
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	config := Config{
		Overrides: printer.TypeOverrides{
			"text": {
				GoType: "string",
				Null: &printer.NullOverride{
					GoPackage: "database/sql",
					GoType:    "NullString",
				},
			},
			"bool": {
				GoType: "bool",
				Null: &printer.NullOverride{
					GoPackage: "database/sql",
					GoType:    "NullBool",
				},
			},
			"char": {
				GoType: "byte",
			},
			"oid": {
				GoType: "uint32",
			},
			"name": {
				GoType: "string",
				Null: &printer.NullOverride{
					GoPackage: "database/sql",
					GoType:    "NullString",
				},
			},
		},
	}

	if _, err := toml.Decode(string(configFile), &config); err != nil {
		return fmt.Errorf("decode config: %w", err)
	}

	conn, err := pgx.Connect(ctx, config.Database)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}

	e := pgengine.New(conn)

	queryEntries, err := os.ReadDir(config.Queries)
	if err != nil {
		return fmt.Errorf("read queries dir: %w", err)
	}

	queryMap := make(map[string]string)
	for _, queryEntry := range queryEntries {
		if queryEntry.IsDir() {
			continue
		}

		queryPath := queryEntry.Name()
		queryName := filepath.Base(queryPath)
		queryName, isQuery := strings.CutSuffix(queryName, ".sql")
		if !isQuery {
			continue
		}

		query, err := os.ReadFile(filepath.Join(config.Queries, queryPath))
		if err != nil {
			return fmt.Errorf("read query: %w", err)
		}

		queryMap[queryName] = strings.TrimSpace(string(query))
	}

	queries, err := e.ResolveQueries(ctx, queryMap)
	if err != nil {
		return fmt.Errorf("resolve queries: %w", err)
	}

	printer := pgprinter.New(config.Package.Name, config.Overrides)
	printed := printer.PrintQueries(queries)

	if err := os.MkdirAll(config.Package.Path, 0644); err != nil {
		return fmt.Errorf("make package dir: %w", err)
	}

	databasePath := filepath.Join(config.Package.Path, "database.go")
	queriesPath := filepath.Join(config.Package.Path, "queries.go")
	modelsPath := filepath.Join(config.Package.Path, "models.go")

	os.WriteFile(databasePath, []byte(printed.Database), 0644)
	os.WriteFile(queriesPath, []byte(printed.Queries), 0644)
	os.WriteFile(modelsPath, []byte(printed.Models), 0644)

	return nil
}
