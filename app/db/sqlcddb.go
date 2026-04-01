package db

import (
	"context"
	"database/sql"
	_ "embed"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed migrations/20251013154838_create_database.sql
var DDL string

func InitDB() (*sql.DB, *Queries, error) {
	conn, err := sql.Open("pgx", os.Getenv("DATABASE_SESSION_DSN"))

	if err != nil {
		return nil, nil, err
	}

	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		if _, err := conn.ExecContext(context.Background(), DDL); err != nil {
			return nil, nil, err
		}
	}

	var query = New(conn)

	return conn, query, nil
}