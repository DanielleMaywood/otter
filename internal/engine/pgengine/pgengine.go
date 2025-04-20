package pgengine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/DanielleMaywood/otter/internal/engine"
	"github.com/DanielleMaywood/otter/internal/engine/pgengine/database"
	"github.com/jackc/pgx/v5"
)

type Engine struct {
	conn  *pgx.Conn
	store database.Store
}

var _ engine.Engine = Engine{}

func New(conn *pgx.Conn) Engine {
	return Engine{conn: conn, store: database.New(conn)}
}

func (e Engine) ResolveQueries(ctx context.Context, queries map[string]string) (engine.Result, error) {
	typeMap := make(map[string]engine.Type)

	result := engine.Result{
		Queries: make(map[string]engine.Query),
	}

	for queryName, query := range queries {
		var queryType engine.Query
		queryType.Name = queryName
		queryType.SQL = strings.TrimSpace(query)

		preparedQuery, err := e.conn.Prepare(ctx, queryName, query)
		if err != nil {
			return result, fmt.Errorf("prepare query '%s': %w", queryName, err)
		}

		queryPlan, err := e.explainQuery(ctx, query)
		if err != nil {
			return result, fmt.Errorf("explain query '%s': %w", queryName, err)
		}

		switch queryPlan.Rows {
		case 0:
			queryType.Type = engine.QueryTypeExec
		default:
			queryType.Type = engine.ParseQueryType(query)
		}

		inputNames := engine.ParseQueryInputNames(query)

		inputNullabilityMap, outputNullabilityMap, err := e.computeNullability(ctx, queryPlan)
		if err != nil {
			return result, fmt.Errorf("compute nullable inputs: %w", err)
		}

		inputNullability := make([]bool, len(preparedQuery.ParamOIDs))
		for idx := range preparedQuery.ParamOIDs {
			inputNullability[idx] = inputNullabilityMap[fmt.Sprintf("$%d", idx+1)]
		}

		outputNullability := make([]bool, len(queryPlan.Output))
		for idx, output := range queryPlan.Output {
			outputNullability[idx] = outputNullabilityMap[output]
		}

		queryType.Inputs = make([]engine.Input, len(preparedQuery.ParamOIDs))
		for idx, oid := range preparedQuery.ParamOIDs {
			nullable := inputNullability[idx]

			inputType, err := e.resolveType(ctx, oid, nil)
			if err != nil {
				return result, fmt.Errorf("resolve type '%d': %w", oid, err)
			}

			inputType.Nullable = nullable
			typeMap[inputType.Name] = inputType

			queryType.Inputs[idx] = engine.Input{
				Name: inputNames[fmt.Sprint(idx+1)],
				Type: inputType,
			}
		}

		queryType.Outputs = make([]engine.Output, len(preparedQuery.Fields))
		for idx, field := range preparedQuery.Fields {
			nullable := outputNullability[idx]

			outputType, err := e.resolveType(ctx, field.DataTypeOID, &nullable)
			if err != nil {
				return result, fmt.Errorf("resolve type '%d': %w", field.DataTypeOID, err)
			}

			typeMap[outputType.Name] = outputType

			outputName := field.Name
			if outputName == "?column?" {
				outputName = ""
			}

			queryType.Outputs[idx] = engine.Output{
				Name: outputName,
				Type: outputType,
			}
		}

		result.Queries[queryName] = queryType
	}

	result.Types = make([]engine.Type, 0, len(typeMap))
	for _, typ := range typeMap {
		typ.Nullable = false
		result.Types = append(result.Types, typ)
	}

	return result, nil
}

func (e Engine) resolveType(ctx context.Context, oid uint32, nullable *bool) (engine.Type, error) {
	typeInfo, err := e.store.GetTypeByOID(ctx, oid)
	if err != nil {
		return engine.Type{}, fmt.Errorf("get type: %w", err)
	}

	if nullable != nil {
		typeInfo.NotNull = !*nullable
	}

	switch typeInfo.Type {
	// Base
	case 'b':
		return engine.Type{
			Kind:     engine.TypeKindBase,
			Name:     typeInfo.Name,
			Nullable: !typeInfo.NotNull,
		}, nil

	// Enum
	case 'e':
		variants, err := e.store.GetEnumVariantsByOID(ctx, oid)
		if err != nil {
			return engine.Type{}, fmt.Errorf("get variants: %w", err)
		}

		return engine.Type{
			Kind:     engine.TypeKindEnum,
			Name:     typeInfo.Name,
			Variants: variants,
			Nullable: !typeInfo.NotNull,
		}, nil

	default:
		return engine.Type{}, fmt.Errorf("unsupported type: %b", typeInfo.Type)
	}
}

type queryExplain struct {
	Plan queryPlan `json:"Plan"`
}

type queryPlan struct {
	NodeType  string      `json:"Node Type"`
	Operation string      `json:"Operation"`
	JoinType  string      `json:"Join Type"`
	Plans     []queryPlan `json:"Plans"`
	Output    []string    `json:"Output"`
	Alias     string      `json:"Alias"`
	Schema    string      `json:"Schema"`
	Relation  string      `json:"Relation Name"`
	Rows      int         `json:"Plan Rows"`
}

func (e Engine) explainQuery(ctx context.Context, query string) (queryPlan, error) {
	query = "explain (format json, verbose, generic_plan) " + query

	results, err := e.conn.PgConn().Exec(ctx, query).ReadAll()
	if err != nil {
		return queryPlan{}, fmt.Errorf("execute explain query: %w", err)
	} else if resultsLen := len(results); resultsLen != 1 {
		return queryPlan{}, fmt.Errorf("unexpected results len: %d", resultsLen)
	}

	result := results[0]
	if err := result.Err; err != nil {
		return queryPlan{}, fmt.Errorf("explain error: %w", err)
	} else if rowsLen := len(result.Rows); rowsLen != 1 {
		return queryPlan{}, fmt.Errorf("unexpected rows len: %d", rowsLen)
	}

	row := result.Rows[0]
	if rowLen := len(row); rowLen != 1 {
		return queryPlan{}, fmt.Errorf("unexpected row len: %d", rowLen)
	}

	var explains []queryExplain
	if err := json.Unmarshal(row[0], &explains); err != nil {
		return queryPlan{}, fmt.Errorf("unmarshal explains: %w", err)
	} else if explainsLen := len(explains); explainsLen != 1 {
		return queryPlan{}, fmt.Errorf("unexpected explains len: %d", explainsLen)
	}

	return explains[0].Plan, nil
}

func (e Engine) computeNullability(ctx context.Context, plan queryPlan) (map[string]bool, map[string]bool, error) {
	switch plan.NodeType {
	case "Result":
		return make(map[string]bool), make(map[string]bool), nil

	case "Hash", "Limit", "Sort", "Materialize":
		return e.computeNullability(ctx, plan.Plans[0])

	case "ModifyTable":
		switch plan.Operation {
		case "Insert":
			inputs, outputs, err := e.computeNullability(ctx, plan.Plans[0])
			if err != nil {
				return nil, nil, err
			}
			if plan.Plans[0].NodeType != "Result" {
				return inputs, outputs, nil
			}

			resultPlan := plan.Plans[0]

			nullability, err := e.store.GetRelationNullability(ctx, database.GetRelationNullabilityParams{
				Schema:   plan.Schema,
				Relation: plan.Relation,
			})

			for idx, name := range resultPlan.Output {
				if nullability[idx] {
					inputs[name] = true
				}
			}

			return inputs, outputs, nil

		default:
			return e.computeNullability(ctx, plan)
		}

	case "Seq Scan", "Index Scan", "Index Only Scan":
		outputs := make(map[string]bool)

		for _, output := range plan.Output {
			columnName, _ := strings.CutPrefix(output, plan.Alias+".")

			nullable, err := e.store.GetColumnNullability(ctx, database.GetColumnNullabilityParams{
				Schema:     plan.Schema,
				Relation:   plan.Relation,
				ColumnName: columnName,
			})
			if err != nil {
				return nil, nil, fmt.Errorf("compute output '%s' nullability: %w", output, err)
			}

			outputs[output] = nullable
		}

		return make(map[string]bool), outputs, nil

	case "Hash Join", "Merge Join", "Nested Loop":
		outputs := make(map[string]bool)
		for _, output := range plan.Output {
			outputs[output] = false
		}

		lhsInputs, lhsOutputs, err := e.computeNullability(ctx, plan.Plans[0])
		if err != nil {
			return nil, nil, err
		}

		rhsInputs, rhsOutputs, err := e.computeNullability(ctx, plan.Plans[1])
		if err != nil {
			return nil, nil, err
		}

		switch plan.JoinType {
		case "Left":
			// When performing a Left join, all columns in the right side
			// will become nullable
			for rhsOutput := range rhsOutputs {
				if _, found := outputs[rhsOutput]; found {
					outputs[rhsOutput] = true
				}
			}

			// We want to keep the nullability of the left side.
			for lhsOutput, nullable := range lhsOutputs {
				if _, found := outputs[lhsOutput]; found {
					outputs[lhsOutput] = nullable
				}
			}

		case "Inner":
			// We want to keep the nullability of the right side.
			for lhsOutput, nullable := range lhsOutputs {
				if _, found := outputs[lhsOutput]; found {
					outputs[lhsOutput] = nullable
				}
			}

			// We want to keep the nullability of the left side.
			for rhsOutput, nullable := range rhsOutputs {
				if _, found := outputs[rhsOutput]; found {
					outputs[rhsOutput] = nullable
				}
			}

		case "Right":
			// When performing a Right join, all columns in the left side
			// will become nullable
			for lhsOutput := range lhsOutputs {
				if _, found := outputs[lhsOutput]; found {
					outputs[lhsOutput] = true
				}
			}

			// We want to keep the nullability of the right side.
			for rhsOutput, nullable := range rhsOutputs {
				if _, found := outputs[rhsOutput]; found {
					outputs[rhsOutput] = nullable
				}
			}

		default:
			return nil, nil, fmt.Errorf("unsupported join type: %s", plan.JoinType)
		}

		inputs := rhsInputs
		for k, v := range lhsInputs {
			inputs[k] = v
		}

		return inputs, outputs, nil

	default:
		return nil, nil, fmt.Errorf("unsupported node type: %s", plan.NodeType)
	}
}
