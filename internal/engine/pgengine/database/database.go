package database

import "github.com/jackc/pgx/v5"

type Store struct {
	db *pgx.Conn
}

func New(db *pgx.Conn) *Store {
	return &Store{db: db}
}
