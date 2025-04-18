package database

import (
	"context"
	"database/sql"
)

type GetColumnNullabilityParams struct {
	Schema     sql.NullString
	Relation   sql.NullString
	ColumnName sql.NullString
}

func (q *Querier) GetColumnNullability(ctx context.Context, params GetColumnNullabilityParams) (bool, error) {
	var item bool
	if err := q.db.QueryRow(ctx, "-- :one\n-- $1: schema\n-- $2: relation\n-- $3: columnName\nselect is_nullable = 'YES' from information_schema.columns where table_schema = $1 and table_name = $2 and column_name = $3 limit 1", params.Schema, params.Relation, params.ColumnName).Scan(&item); err != nil {
		return item, err
	}
	return item, nil
}

func (q *Querier) GetEnumVariantsByOID(ctx context.Context, oid sql.Null[uint32]) ([]string, error) {
	rows, err := q.db.Query(ctx, "-- :many\n-- $1: oid\nselect enumlabel from pg_enum where enumtypid = $1 order by enumsortorder", oid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []string
	for rows.Next() {
		var item string
		if err := rows.Scan(&item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

type GetTypeByOIDRow struct {
	Name    string
	Type    byte
	NotNull bool
}

func (q *Querier) GetTypeByOID(ctx context.Context, oid sql.Null[uint32]) (GetTypeByOIDRow, error) {
	var item GetTypeByOIDRow
	if err := q.db.QueryRow(ctx, "-- :one\n-- $1: oid\nselect\n    typname as \"name\",\n    typtype as \"type\",\n    typnotnull as \"not_null\"\nfrom pg_type where oid = $1 limit 1", oid).Scan(&item.Name, &item.Type, &item.NotNull); err != nil {
		return item, err
	}
	return item, nil
}
