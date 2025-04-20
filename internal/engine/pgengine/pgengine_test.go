package pgengine_test

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/DanielleMaywood/otter/internal/engine"
	"github.com/DanielleMaywood/otter/internal/engine/pgengine"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/peterldowns/pgtestdb"
	"github.com/peterldowns/pgtestdb/migrators/golangmigrator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		schema          string
		queries         map[string]string
		expectedTypes   []engine.Type
		expectedQueries map[string]engine.Query
	}{
		{
			name:   "SimpleUserTable",
			schema: `create table users ( id int not null, username text );`,
			queries: map[string]string{
				"GetUsers": `
					-- :many
					select * from users
				`,
				"GetUserByID": `
					-- :one
					-- $1: id
					select * from users where id = $1 limit 1
				`,
				"InsertUser": `
					-- :exec
					-- $1: id
					-- $2: username
					insert into users ( id, username ) values ($1, $2)
				`,
			},
			expectedTypes: []engine.Type{
				{
					Kind: engine.TypeKindBase,
					Name: "int4",
				},
				{
					Kind: engine.TypeKindBase,
					Name: "text",
				},
			},
			expectedQueries: map[string]engine.Query{
				"GetUsers": {
					Type:   engine.QueryTypeMany,
					Inputs: []engine.Input{},
					Outputs: []engine.Output{
						{
							Name: "id",
							Type: engine.Type{
								Kind: engine.TypeKindBase,
								Name: "int4",
							},
						},
						{
							Name: "username",
							Type: engine.Type{
								Kind:     engine.TypeKindBase,
								Name:     "text",
								Nullable: true,
							},
						},
					},
				},
				"GetUserByID": {
					Type: engine.QueryTypeOne,
					Inputs: []engine.Input{
						{
							Name: "id",
							Type: engine.Type{
								Kind: engine.TypeKindBase,
								Name: "int4",
							},
						},
					},
					Outputs: []engine.Output{
						{
							Name: "id",
							Type: engine.Type{
								Kind: engine.TypeKindBase,
								Name: "int4",
							},
						},
						{
							Name: "username",
							Type: engine.Type{
								Kind:     engine.TypeKindBase,
								Name:     "text",
								Nullable: true,
							},
						},
					},
				},
				"InsertUser": {
					Type: engine.QueryTypeExec,
					Inputs: []engine.Input{
						{
							Name: "id",
							Type: engine.Type{
								Kind: engine.TypeKindBase,
								Name: "int4",
							},
						},
						{
							Name: "username",
							Type: engine.Type{
								Kind:     engine.TypeKindBase,
								Name:     "text",
								Nullable: true,
							},
						},
					},
					Outputs: []engine.Output{},
				},
			},
		},
		{
			name: "SimpleEmployeeTable",
			schema: `
				create table employees ( id int not null, name text not null, department_id int );
				create table departments ( id int not null, name text not null );
			`,
			queries: map[string]string{
				"GetEmployeesWithDepartments": `
					-- :many
					select
						e.id   as employee_id,
						e.name as employee_name,
						d.id   as department_id,
						d.name as department_name
					from employees e
					left join departments d
					on e.department_id = d.id
				`,
				"GetDepartmentsWithEmployees": `
					-- :many
					select
						e.id   as employee_id,
						e.name as employee_name,
						d.id   as department_id,
						d.name as department_name
					from employees e
					right join departments d
					on e.department_id = d.id
				`,
				"GetEmployeesWithValidDepartments": `
					-- :many
					select
						e.id   as employee_id,
						e.name as employee_name,
						d.id   as department_id,
						d.name as department_name
					from employees e
					inner join departments d
					on e.department_id = d.id
				`,
			},
			expectedTypes: []engine.Type{
				{
					Kind: engine.TypeKindBase,
					Name: "int4",
				},
				{
					Kind: engine.TypeKindBase,
					Name: "text",
				},
			},
			expectedQueries: map[string]engine.Query{
				"GetEmployeesWithDepartments": {
					Type:   engine.QueryTypeMany,
					Inputs: []engine.Input{},
					Outputs: []engine.Output{
						{
							Name: "employee_id",
							Type: engine.Type{
								Kind: engine.TypeKindBase,
								Name: "int4",
							},
						},
						{
							Name: "employee_name",
							Type: engine.Type{
								Kind: engine.TypeKindBase,
								Name: "text",
							},
						},
						{
							Name: "department_id",
							Type: engine.Type{
								Kind:     engine.TypeKindBase,
								Name:     "int4",
								Nullable: true,
							},
						},
						{
							Name: "department_name",
							Type: engine.Type{
								Kind:     engine.TypeKindBase,
								Name:     "text",
								Nullable: true,
							},
						},
					},
				},
				"GetDepartmentsWithEmployees": {
					Type:   engine.QueryTypeMany,
					Inputs: []engine.Input{},
					Outputs: []engine.Output{
						{
							Name: "employee_id",
							Type: engine.Type{
								Kind:     engine.TypeKindBase,
								Name:     "int4",
								Nullable: true,
							},
						},
						{
							Name: "employee_name",
							Type: engine.Type{
								Kind:     engine.TypeKindBase,
								Name:     "text",
								Nullable: true,
							},
						},
						{
							Name: "department_id",
							Type: engine.Type{
								Kind: engine.TypeKindBase,
								Name: "int4",
							},
						},
						{
							Name: "department_name",
							Type: engine.Type{
								Kind: engine.TypeKindBase,
								Name: "text",
							},
						},
					},
				},
				"GetEmployeesWithValidDepartments": {
					Type:   engine.QueryTypeMany,
					Inputs: []engine.Input{},
					Outputs: []engine.Output{
						{
							Name: "employee_id",
							Type: engine.Type{
								Kind: engine.TypeKindBase,
								Name: "int4",
							},
						},
						{
							Name: "employee_name",
							Type: engine.Type{
								Kind: engine.TypeKindBase,
								Name: "text",
							},
						},
						{
							Name: "department_id",
							Type: engine.Type{
								Kind: engine.TypeKindBase,
								Name: "int4",
							},
						},
						{
							Name: "department_name",
							Type: engine.Type{
								Kind: engine.TypeKindBase,
								Name: "text",
							},
						},
					},
				},
			},
		},
		{
			name: "NoSchema",
			queries: map[string]string{
				"GetOne": `
					-- :one
					select 1
				`,
			},
			expectedTypes: []engine.Type{
				{
					Kind: engine.TypeKindBase,
					Name: "int4",
				},
			},
			expectedQueries: map[string]engine.Query{
				"GetOne": {
					Type:   engine.QueryTypeOne,
					Inputs: []engine.Input{},
					Outputs: []engine.Output{
						{
							Type: engine.Type{
								Kind:     engine.TypeKindBase,
								Name:     "int4",
								Nullable: false,
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		// Rather than hand-write the expected SQL, we're going to generate
		// it here.
		for queryName, query := range tt.queries {
			expectedQuery := tt.expectedQueries[queryName]
			expectedQuery.Name = queryName
			expectedQuery.SQL = strings.TrimSpace(query)
			tt.expectedQueries[queryName] = expectedQuery
		}

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := mustCreateDB(t, tt.schema)
			e := pgengine.New(db)

			resolved, err := e.ResolveQueries(t.Context(), tt.queries)
			require.Nil(t, err)

			assert.ElementsMatch(t, resolved.Types, tt.expectedTypes)
			assert.Equal(t, resolved.Queries, tt.expectedQueries)
		})
	}

}

func mustCreateDB(t *testing.T, schema string) *pgx.Conn {
	t.Helper()

	fs := fstest.MapFS{
		"migrations/000001_create_database.up.sql": &fstest.MapFile{
			Data: []byte(schema),
		},
	}

	migrator := golangmigrator.New("migrations", golangmigrator.WithFS(fs))

	config := pgtestdb.Custom(t, pgtestdb.Config{
		DriverName: "pgx",
		Host:       "localhost",
		User:       "danielle",
		Port:       "5432",
		Options:    "sslmode=disable",
	}, migrator)

	conn, err := pgx.Connect(t.Context(), config.URL())
	if err != nil {
		t.Fatalf("pgx.Connect = %v, want nil", err)
	}

	t.Cleanup(func() {
		err := conn.Close(t.Context())
		if err != nil {
			t.Fatalf("conn.Close = %v, want nil", err)
		}
	})

	return conn
}
