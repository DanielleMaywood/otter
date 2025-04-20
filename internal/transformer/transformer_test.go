package transformer_test

import (
	"testing"

	"github.com/DanielleMaywood/otter/internal/engine"
	"github.com/DanielleMaywood/otter/internal/transformer"
	"github.com/stretchr/testify/assert"
)

func TestTypeNameTransformer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		before engine.Result
		after  engine.Result
	}{
		{
			name: "SingleInputSingleOutput",
			before: engine.Result{
				Types: []engine.Type{{Name: "int4"}},
				Queries: map[string]engine.Query{
					"SomeQuery": {
						Inputs: []engine.Input{
							{Name: "id", Type: engine.Type{Name: "text"}},
						},
						Outputs: []engine.Output{
							{Name: "id", Type: engine.Type{Name: "bool"}},
						},
					},
				},
			},
			after: engine.Result{
				Types: []engine.Type{{Name: "Int4"}},
				Queries: map[string]engine.Query{
					"SomeQuery": {
						Inputs: []engine.Input{
							{Name: "id", Type: engine.Type{Name: "Text"}},
						},
						Outputs: []engine.Output{
							{Name: "ID", Type: engine.Type{Name: "Bool"}},
						},
					},
				},
			},
		},
		{
			name: "MultipleInputSingleOutput",
			before: engine.Result{
				Types: []engine.Type{{Name: "int4"}},
				Queries: map[string]engine.Query{
					"SomeQuery": {
						Inputs: []engine.Input{
							{Name: "id", Type: engine.Type{Name: "text"}},
							{Name: "name", Type: engine.Type{Name: "text"}},
						},
						Outputs: []engine.Output{
							{Name: "id", Type: engine.Type{Name: "bool"}},
						},
					},
				},
			},
			after: engine.Result{
				Types: []engine.Type{{Name: "Int4"}},
				Queries: map[string]engine.Query{
					"SomeQuery": {
						Inputs: []engine.Input{
							{Name: "ID", Type: engine.Type{Name: "Text"}},
							{Name: "Name", Type: engine.Type{Name: "Text"}},
						},
						Outputs: []engine.Output{
							{Name: "ID", Type: engine.Type{Name: "Bool"}},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			trans := transformer.NewTypeNameTransformer(transformer.NewStringCaser(map[string]string{
				"id": "ID",
			}))

			trans.Transform(&tt.before)
			assert.Equal(t, tt.after, tt.before)
		})
	}
}
