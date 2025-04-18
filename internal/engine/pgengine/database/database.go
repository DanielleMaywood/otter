package database

import (
	"context"
	"database/sql"
	"github.com/jackc/pgx/v5"
)

type Store interface {
	GetColumnNullability(ctx context.Context, params GetColumnNullabilityParams) (bool, error)
	GetEnumVariantsByOID(ctx context.Context, oid sql.Null[uint32]) ([]string, error)
	GetTypeByOID(ctx context.Context, oid sql.Null[uint32]) (GetTypeByOIDRow, error)
}

type Querier struct {
	db *pgx.Conn
}

func New(db *pgx.Conn) *Querier {
	return &Querier{db: db}
}
