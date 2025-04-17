package database

import (
	"context"
	"database/sql"
)

func (s *Store) GetColumnNullability(ctx context.Context, schema sql.NullString, relation sql.NullString, columnName sql.NullString) (bool, error) {
	var item bool
	if err := s.db.QueryRow(ctx, "-- :one\n-- $1: schema\n-- $2: relation\n-- $3: columnName\nselect is_nullable = 'YES' from information_schema.columns where table_schema = $1 and table_name = $2 and column_name = $3 limit 1", schema, relation, columnName).Scan(&item); err != nil {
		return item, err
	}
	return item, nil
}

func (s *Store) GetEnumVariantsByOID(ctx context.Context, oid sql.Null[uint32]) ([]string, error) {
	rows, err := s.db.Query(ctx, "-- :many\n-- $1: oid\nselect enumlabel from pg_enum where enumtypid = $1 order by enumsortorder", oid)
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

func (s *Store) GetTypeByOID(ctx context.Context, oid sql.Null[uint32]) (GetTypeByOIDRow, error) {
	var item GetTypeByOIDRow
	if err := s.db.QueryRow(ctx, "-- :one\n-- $1: oid\nselect\n    typname as \"name\",\n    typtype as \"type\",\n    typnotnull as \"not_null\"\nfrom pg_type where oid = $1 limit 1", oid).Scan(&item.Name, &item.Type, &item.NotNull); err != nil {
		return item, err
	}
	return item, nil
}
