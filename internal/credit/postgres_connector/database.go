package postgres_connector

import (
	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/XSAM/otelsql"
	_ "github.com/jackc/pgx/v5/stdlib" // pgx database driver
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"

	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db"
)

// Open new connection
func Open(databaseURL string) (*db.Client, error) {
	// TODO: inject trace and metrics provider
	database, err := otelsql.Open("pgx", databaseURL, otelsql.WithAttributes(
		semconv.DBSystemPostgreSQL,
	))
	if err != nil {
		return nil, err
	}

	drv := entsql.OpenDB(dialect.Postgres, database)

	return db.NewClient(db.Driver(drv)), nil
}
