package db

import (
	"context"
	"database/sql"
	_ "embed"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed migrations/20251013154838_create_database.sql
var ddl string

var (
	Query *Queries
	DB 	  *sql.DB
)

func InitDB() error {
	conn, err := sql.Open("pgx", os.Getenv("DATABASE_SESSION_DSN"))

	if err != nil {
		return err
	}

	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		if _, err := conn.ExecContext(context.Background(), ddl); err != nil {
			return err
		}
	}

	Query = New(conn)
	DB = conn

	return nil
}