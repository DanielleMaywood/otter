package main

import (
	"context"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/DanielleMaywood/otter/internal/engine/pgengine"
	"github.com/DanielleMaywood/otter/internal/printer"
	"github.com/DanielleMaywood/otter/internal/printer/pgprinter"
	"github.com/DanielleMaywood/otter/pkg/otter"
	"github.com/jackc/pgx/v5"
)

type Config struct {
	Overrides printer.TypeOverrides `toml:"overrides"`

	Stores []StoreConfig
}

type StoreConfig struct {
	Database string
	Queries  string
	Package  struct {
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

	for _, store := range config.Stores {
		conn, err := pgx.Connect(ctx, store.Database)
		if err != nil {
			return fmt.Errorf("connect to database: %w", err)
		}

		engine := pgengine.New(conn)
		printer := pgprinter.New(store.Package.Name, config.Overrides)

		if err := otter.New(engine, printer).Run(ctx,
			store.Queries,
			store.Package.Path,
		); err != nil {
			return err
		}
	}

	return nil
}
