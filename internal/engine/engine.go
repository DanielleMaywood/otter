package engine

import (
	"context"
	"strings"
)

type TypeKind string

var (
	TypeKindBase TypeKind = "base"
	TypeKindEnum TypeKind = "enum"
)

type Type struct {
	Kind     TypeKind
	Name     string
	Nullable bool
	Variants []string
}

type QueryType string

var (
	QueryTypeExec QueryType = "exec"
	QueryTypeOne  QueryType = "one"
	QueryTypeMany QueryType = "many"
)

type Input struct {
	Name string
	Type Type
}

type Output struct {
	Name string
	Type Type
}

type Query struct {
	SQL     string
	Name    string
	Type    QueryType
	Inputs  []Input
	Outputs []Output
}

type Result struct {
	Types []Type

	Queries map[string]Query
}

type Engine interface {
	ResolveQueries(ctx context.Context, queries map[string]string) (Result, error)
}

func ParseQueryInputNames(query string) map[string]string {
	inputNames := make(map[string]string)

	for queryLine := range strings.SplitSeq(query, "\n") {
		queryLine = strings.TrimSpace(queryLine)
		queryLine, hasArgument := strings.CutPrefix(queryLine, "-- $")
		if !hasArgument {
			continue
		}

		arg, name, hasName := strings.Cut(queryLine, ":")
		if !hasName {
			continue
		}

		inputNames[strings.TrimSpace(arg)] = strings.TrimSpace(name)
	}

	return inputNames
}

func ParseQueryType(query string) QueryType {
	for queryLine := range strings.SplitSeq(query, "\n") {
		queryLine = strings.TrimSpace(queryLine)

		switch queryLine {
		case "-- :one":
			return QueryTypeOne
		case "-- :exec":
			return QueryTypeExec
		case "-- :many":
			return QueryTypeMany
		}
	}

	return QueryTypeMany
}
