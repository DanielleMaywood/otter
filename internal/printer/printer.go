package printer

import "github.com/DanielleMaywood/otter/internal/engine"

type TypeOverride struct {
	GoPackage string `toml:"go_package"`
	GoType    string `toml:"go_type"`

	Null *NullOverride `toml:"null"`
}

type NullOverride struct {
	GoPackage string `toml:"go_package"`
	GoType    string `toml:"go_type"`
}

type TypeOverrides map[string]TypeOverride

type Result struct {
	Database string
	Models   string
	Queries  string
}

type Printer interface {
	PrintQueries(queries engine.Result) Result
}
