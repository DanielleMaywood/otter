package transformer_test

import (
	"testing"

	"github.com/DanielleMaywood/otter/internal/transformer"
	"github.com/stretchr/testify/assert"
)

func TestToPascalCase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		initialisms map[string]string
		from        string
		expected    string
	}{
		{
			from:     "snake_case",
			expected: "SnakeCase",
		},
		{
			initialisms: map[string]string{"id": "ID"},
			from:        "user_id",
			expected:    "UserID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.from, func(t *testing.T) {
			t.Parallel()

			caser := transformer.NewStringCaser(tt.initialisms)
			assert.Equal(t, tt.expected, caser.ToPascalCase(tt.from))
		})
	}
}

func TestToCamelCase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		initialisms map[string]string
		from        string
		expected    string
	}{
		{
			from:     "snake_case",
			expected: "snakeCase",
		},
		{
			initialisms: map[string]string{"id": "ID"},
			from:        "user_id",
			expected:    "userID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.from, func(t *testing.T) {
			t.Parallel()

			caser := transformer.NewStringCaser(tt.initialisms)
			assert.Equal(t, tt.expected, caser.ToCamelCase(tt.from))
		})
	}
}
