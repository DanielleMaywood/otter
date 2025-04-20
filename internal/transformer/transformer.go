package transformer

import (
	"github.com/DanielleMaywood/otter/internal/engine"
)

type Transformer interface {
	Transform(result *engine.Result)
}

func Transform(
	result *engine.Result,
	transformers ...Transformer,
) {
	for _, transformer := range transformers {
		transformer.Transform(result)
	}
}

type TypeNameTransformer struct {
	caser StringCaser
}

func NewTypeNameTransformer(caser StringCaser) TypeNameTransformer {
	return TypeNameTransformer{caser: caser}
}

func (t TypeNameTransformer) Transform(result *engine.Result) {
	for idx, typ := range result.Types {
		typ.Name = t.caser.ToPascalCase(typ.Name)

		result.Types[idx] = typ
	}

	for queryName, query := range result.Queries {
		for idx, input := range query.Inputs {
			if len(query.Inputs) == 1 {
				input.Name = t.caser.ToCamelCase(input.Name)
			} else {
				input.Name = t.caser.ToPascalCase(input.Name)
			}

			input.Type.Name = t.caser.ToPascalCase(input.Type.Name)

			query.Inputs[idx] = input
		}

		for idx, output := range query.Outputs {
			output.Name = t.caser.ToPascalCase(output.Name)
			output.Type.Name = t.caser.ToPascalCase(output.Type.Name)

			query.Outputs[idx] = output
		}

		result.Queries[queryName] = query
	}
}
