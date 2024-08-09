package entutils

import (
	"database/sql"

	"entgo.io/ent/dialect"
	entDialectSQL "entgo.io/ent/dialect/sql"
	"github.com/XSAM/otelsql"
	_ "github.com/jackc/pgx/v5/stdlib" // pgx database driver
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
)

func GetSQLDriver(databaseURL string) (*sql.DB, error) {
	return otelsql.Open("pgx", databaseURL, otelsql.WithAttributes(
		semconv.DBSystemPostgreSQL,
	))
}

func GetEntDriver(databaseURL string) (*entDialectSQL.Driver, error) {
	// TODO: inject trace and metrics provider
	database, err := GetSQLDriver(databaseURL)
	if err != nil {
		return nil, err
	}

	return entDialectSQL.OpenDB(dialect.Postgres, database), nil
}
