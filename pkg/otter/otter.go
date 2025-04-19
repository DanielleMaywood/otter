package otter

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/DanielleMaywood/otter/internal/engine"
	"github.com/DanielleMaywood/otter/internal/printer"
	"github.com/spf13/afero"
)

type Option func(*Otter)

type Otter struct {
	engine  engine.Engine
	printer printer.Printer
	fs      afero.Fs

	databasePath string
	queriesPath  string
	modelsPath   string
}

func WithFS(fs afero.Fs) Option {
	return func(o *Otter) {
		o.fs = fs
	}
}

func New(engine engine.Engine, printer printer.Printer, opts ...Option) Otter {
	otter := Otter{
		engine:       engine,
		printer:      printer,
		databasePath: "database.go",
		queriesPath:  "queries.go",
		modelsPath:   "models.go",
	}
	for _, opt := range opts {
		opt(&otter)
	}
	if otter.fs == nil {
		otter.fs = afero.NewOsFs()
	}
	return otter
}

func (o Otter) Run(ctx context.Context, queryPath, outPath string) error {
	queryMap, err := o.collectQueries(queryPath)
	if err != nil {
		return fmt.Errorf("collect queries: %w", err)
	}

	queries, err := o.engine.ResolveQueries(ctx, queryMap)
	if err != nil {
		return fmt.Errorf("resolve queries: %w", err)
	}

	printed := o.printer.PrintQueries(queries)
	if err := o.writePrintedQueries(outPath, printed); err != nil {
		return fmt.Errorf("write queries: %w", err)
	}

	return nil
}

func (o Otter) collectQueries(queryPath string) (map[string]string, error) {
	entries, err := afero.ReadDir(o.fs, queryPath)
	if err != nil {
		return nil, fmt.Errorf("read directory '%s': %w", queryPath, err)
	}

	queryMap := make(map[string]string)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		queryName := entry.Name()
		queryName, isQuery := strings.CutSuffix(queryName, ".sql")
		if !isQuery {
			continue
		}

		query, err := afero.ReadFile(o.fs, filepath.Join(queryPath, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read query '%s': %w", queryName, err)
		}

		queryMap[queryName] = strings.TrimSpace(string(query))
	}

	return queryMap, nil
}

func (o Otter) writePrintedQueries(outPath string, printed printer.Result) error {
	databasePath := filepath.Join(outPath, o.databasePath)
	queriesPath := filepath.Join(outPath, o.queriesPath)
	modelsPath := filepath.Join(outPath, o.modelsPath)

	if err := afero.WriteFile(o.fs, databasePath, []byte(printed.Database), 0644); err != nil {
		return fmt.Errorf("write %s: %w", databasePath, err)
	}

	if err := afero.WriteFile(o.fs, queriesPath, []byte(printed.Queries), 0644); err != nil {
		return fmt.Errorf("write %s: %w", queriesPath, err)
	}

	if err := afero.WriteFile(o.fs, modelsPath, []byte(printed.Models), 0644); err != nil {
		return fmt.Errorf("write %s: %w", modelsPath, err)
	}

	return nil
}
