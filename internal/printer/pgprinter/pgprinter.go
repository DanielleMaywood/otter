package pgprinter

import (
	"cmp"
	"fmt"
	"maps"
	"slices"

	"github.com/DanielleMaywood/otter/internal/engine"
	"github.com/DanielleMaywood/otter/internal/printer"
	"github.com/dave/jennifer/jen"
	"github.com/iancoleman/strcase"
)

type Printer struct {
	packageName string
	overrides   printer.TypeOverrides
}

func New(packageName string, overrides printer.TypeOverrides) Printer {
	return Printer{packageName: packageName, overrides: overrides}
}

func (p Printer) PrintQueries(queries engine.Result) printer.Result {
	databaseFile := jen.NewFile(p.packageName)
	queriesFile := jen.NewFile(p.packageName)
	modelsFile := jen.NewFile(p.packageName)

	databaseFile.ImportName("github.com/jackc/pgx/v5", "pgx")
	databaseFile.Type().Id("Store").Struct(
		jen.Id("db").Op("*").Qual("github.com/jackc/pgx/v5", "Conn"),
	)
	databaseFile.Func().
		Id("New").
		Params(jen.Id("db").Op("*").Qual("github.com/jackc/pgx/v5", "Conn")).
		Op("*").Id("Store").
		Block(
			jen.Return(jen.Op("&").Id("Store").Values(jen.Dict{
				jen.Id("db"): jen.Id("db"),
			})),
		)

	// Sort the types by name then type so that we have a stable
	// order.
	slices.SortStableFunc(queries.Types, func(a, b engine.Type) int {
		return cmp.Compare(a.Name, b.Name)
	})
	slices.SortStableFunc(queries.Types, func(a, b engine.Type) int {
		return cmp.Compare(a.Kind, b.Kind)
	})

	for _, typ := range queries.Types {
		p.printType(modelsFile, typ)
	}

	// Sort the queries alphabetically for a stable order.
	queryNames := slices.Collect(maps.Keys(queries.Queries))
	slices.SortStableFunc(queryNames, cmp.Compare)

	for _, queryName := range queryNames {
		query := queries.Queries[queryName]
		p.printQuery(queriesFile, query)
	}

	return printer.Result{
		Database: databaseFile.GoString(),
		Queries:  queriesFile.GoString(),
		Models:   modelsFile.GoString(),
	}
}

func (p Printer) printQuery(file *jen.File, query engine.Query) {
	switch query.Type {
	case engine.QueryTypeExec:
		p.printExecQuery(file, query)

	case engine.QueryTypeOne:
		p.printOneQuery(file, query)

	case engine.QueryTypeMany:
		p.printManyQuery(file, query)

	default:
		panic(fmt.Sprintf("unexpected query kind: %s", query.Type))
	}
}

func (p Printer) printExecQuery(file *jen.File, query engine.Query) {
	params := p.buildQueryParams(query)
	args := p.buildQueryArgs(query)

	file.Func().
		Params(jen.Id("s").Op("*").Id("Store")).
		Id(query.Name).
		Params(
			append([]jen.Code{jen.Id("ctx").Qual("context", "Context")}, params...)...,
		).
		Error().
		Block(
			jen.List(jen.Id("_"), jen.Err()).
				Op(":=").
				Id("s").Dot("db").Dot("Exec").Call(
				append([]jen.Code{jen.Id("ctx"), jen.Lit(query.SQL)}, args...)...,
			),
			jen.Return(jen.Err()),
		).
		Line()

}

func (p Printer) printOneQuery(file *jen.File, query engine.Query) {
	resultType, scanRefs := p.maybePrintQueryRowType(file, query)
	params := p.buildQueryParams(query)
	args := p.buildQueryArgs(query)

	file.Func().
		Params(jen.Id("s").Op("*").Id("Store")).
		Id(query.Name).
		Params(
			append([]jen.Code{jen.Id("ctx").Qual("context", "Context")}, params...)...,
		).
		Params(jen.Add(resultType), jen.Error()).
		Block(
			jen.Var().Id("item").Add(resultType),
			jen.If(
				jen.Err().Op(":=").Id("s").Dot("db").Dot("QueryRow").Call(
					append([]jen.Code{jen.Id("ctx"), jen.Lit(query.SQL)}, args...)...,
				).Dot("Scan").Call(scanRefs...),
				jen.Err().Op("!=").Nil(),
			).Block(
				jen.Return(jen.Id("item"), jen.Err()),
			),
			jen.Return(jen.Id("item"), jen.Nil()),
		).
		Line()
}

func (p Printer) printManyQuery(file *jen.File, query engine.Query) {
	resultType, scanRefs := p.maybePrintQueryRowType(file, query)
	params := p.buildQueryParams(query)
	args := p.buildQueryArgs(query)

	file.Func().
		Params(jen.Id("s").Op("*").Id("Store")).
		Id(query.Name).
		Params(
			append([]jen.Code{jen.Id("ctx").Qual("context", "Context")}, params...)...,
		).
		Params(jen.Index().Add(resultType), jen.Error()).
		Block(
			jen.List(jen.Id("rows"), jen.Err()).
				Op(":=").
				Id("s").Dot("db").Dot("Query").Call(
				append([]jen.Code{jen.Id("ctx"), jen.Lit(query.SQL)}, args...)...,
			),
			jen.If(jen.Err().Op("!=").Nil()).Block(
				jen.Return(jen.Nil(), jen.Err()),
			),
			jen.Defer().Id("rows").Dot("Close").Call(),
			jen.Line(),
			jen.Var().Id("items").Index().Add(resultType),
			jen.For(jen.Id("rows").Dot("Next").Call()).Block(
				jen.Var().Id("item").Add(resultType),
				jen.If(
					jen.Err().
						Op(":=").
						Id("rows").Dot("Scan").Call(scanRefs...),
					jen.Err().Op("!=").Nil(),
				).Block(
					jen.Return(jen.Nil(), jen.Err()),
				),
				jen.Id("items").Op("=").Append(jen.Id("items"), jen.Id("item")),
			),
			jen.If(jen.Err().Op(":=").Id("rows").Dot("Err").Call(), jen.Err().Op("!=").Nil()).Block(
				jen.Return(jen.Nil(), jen.Err()),
			),
			jen.Line(),
			jen.Return(jen.Id("items"), jen.Nil()),
		).
		Line()
}

func (p Printer) buildQueryParams(query engine.Query) []jen.Code {
	params := make([]jen.Code, len(query.Inputs))
	for idx, input := range query.Inputs {
		paramName := input.Name
		if paramName == "" {
			paramName = fmt.Sprintf("arg%d", idx)
		}

		typeName := p.typeID(input.Type)
		params[idx] = jen.Id(paramName).Add(typeName)
	}
	return params
}

func (p Printer) buildQueryArgs(query engine.Query) []jen.Code {
	params := make([]jen.Code, len(query.Inputs))
	for idx, input := range query.Inputs {
		paramName := input.Name
		if paramName == "" {
			paramName = fmt.Sprintf("arg%d", idx)
		}

		params[idx] = jen.Id(paramName)
	}
	return params
}

func (p Printer) buildQueryScanReferences(query engine.Query) []jen.Code {
	scanReferences := make([]jen.Code, len(query.Outputs))
	for idx, output := range query.Outputs {
		outputName := output.Name
		if outputName == "" {
			outputName = fmt.Sprintf("Field%d", idx)
		}

		scanReferences[idx] = jen.Op("&").Id("item").Dot(outputName)
	}
	return scanReferences
}

func (p Printer) maybePrintQueryRowType(file *jen.File, query engine.Query) (jen.Code, []jen.Code) {
	if len(query.Outputs) != 1 {
		p.printQueryRowType(file, query)
		return jen.Id(query.Name + "Row"), p.buildQueryScanReferences(query)
	}

	outputType := query.Outputs[0].Type
	resultType := p.typeID(outputType)
	return resultType, []jen.Code{jen.Op("&").Id("item")}
}

func (p Printer) printQueryRowType(file *jen.File, query engine.Query) {
	fields := make([]jen.Code, len(query.Outputs))
	for idx, output := range query.Outputs {
		outputName := output.Name
		if outputName == "" {
			outputName = fmt.Sprintf("Field%d", idx)
		}

		fieldType := p.typeID(output.Type)

		fields[idx] = jen.Id(outputName).Add(fieldType)
	}

	file.Type().
		Id(query.Name + "Row").
		Struct(fields...).
		Line()
}

func (p Printer) printType(file *jen.File, typ engine.Type) {
	override, found := p.overrides[strcase.ToSnake(typ.Name)]
	if found && (!typ.Nullable && override.GoType != "" || typ.Nullable && override.Null != nil) {
		return
	}

	switch typ.Kind {
	case engine.TypeKindBase:
		p.printBaseType(file, typ)
		p.printNullableType(file, typ)

	case engine.TypeKindEnum:
		p.printEnumType(file, typ)
		p.printNullableType(file, typ)

	default:
		panic(fmt.Sprintf("unexpected type kind: %s", typ.Kind))
	}
}

func (p Printer) printBaseType(file *jen.File, typ engine.Type) {
	switch typ.Name {
	case "Int4":
		file.Type().Id(typ.Name).Op("=").Int32().Line()

	case "Text":
		file.Type().Id(typ.Name).Op("=").String().Line()

	case "Name":
		file.Type().Id(typ.Name).Op("=").String().Line()

	case "Char":
		file.Type().Id(typ.Name).Op("=").Byte().Line()

	case "Bool":
		file.Type().Id(typ.Name).Op("=").Bool().Line()

	case "Oid":
		file.Type().Id(typ.Name).Op("=").Uint32().Line()

	default:
		panic(fmt.Sprintf("unexpected base type: %s", typ.Name))
	}
}

func (p Printer) printEnumType(file *jen.File, typ engine.Type) {
	file.Type().Id(typ.Name).String().Line()

	defs := make([]jen.Code, len(typ.Variants))
	for idx, variant := range typ.Variants {
		variantName := typ.Name + strcase.ToCamel(variant)
		defs[idx] = jen.Id(variantName).Op("=").Lit(variant)
	}

	file.Const().Defs(defs...)

	file.Func().
		Params(jen.Id("t").Op("*").Id(typ.Name)).
		Id("Scan").
		Params(jen.Id("src").Any()).
		Error().
		Block(
			jen.Switch(jen.Id("s").Op(":=").Id("src").Assert(jen.Type())).Block(
				jen.Case(jen.Index().Byte()).Block(
					jen.Op("*").Id("t").Op("=").Id(typ.Name).Call(jen.Id("s")),
				),
				jen.Case(jen.String()).Block(
					jen.Op("*").Id("t").Op("=").Id(typ.Name).Call(jen.Id("s")),
				),
				jen.Default().Block(
					jen.Return(jen.Qual("fmt", "Errorf").Call(
						jen.Lit("unsupported scan type for "+typ.Name+": %T"),
						jen.Id("src"),
					)),
				),
			),
			jen.Return(jen.Nil()),
		)
}

func (p Printer) printNullableType(file *jen.File, typ engine.Type) {
	file.Type().Id("Null"+typ.Name).Struct(
		jen.Id(typ.Name).Id(typ.Name),
		jen.Id("Valid").Bool(),
	)

	file.Func().
		Params(jen.Id("t").Op("*").Id("Null"+typ.Name)).
		Id("Scan").
		Params(jen.Id("src").Any()).
		Error().
		Block(
			jen.Var().Id("empty").Id(typ.Name),
			jen.If(jen.Id("src").Op("==").Nil()).Block(
				jen.Id("t").Dot(typ.Name).Op("=").Id("empty"),
				jen.Id("t").Dot("Valid").Op("=").False(),
				jen.Return(jen.Nil()),
			),
			jen.Id("t").Dot("Valid").Op("=").True(),
			jen.Return(jen.Id("t").Dot(typ.Name).Dot("Scan").Call(jen.Id("src"))),
		).
		Line()

	file.Func().
		Params(jen.Id("t").Id("Null"+typ.Name)).
		Id("Value").
		Params().
		Params(jen.Qual("database/sql/driver", "Value"), jen.Error()).
		Block(
			jen.If(jen.Op("!").Id("t").Dot("Valid")).Block(
				jen.Return(jen.Nil(), jen.Nil()),
			),
			jen.Return(jen.String().Call(jen.Id("t").Dot(typ.Name)), jen.Nil()),
		)
}

func (p Printer) typeID(typ engine.Type) jen.Code {
	typeID := jen.Id(typ.Name)

	override, found := p.overrides[strcase.ToSnake(typ.Name)]
	if found {
		if override.GoType != "" {
			typeID = jen.Qual(override.GoPackage, override.GoType)
		}

		if typ.Nullable && override.Null != nil {
			return jen.Qual(override.Null.GoPackage, override.Null.GoType)
		}
	}

	if typ.Nullable {
		return jen.Qual("database/sql", "Null").Index(typeID)
	}

	return typeID
}
