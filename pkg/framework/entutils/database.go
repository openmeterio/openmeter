package entutils

import (
	"entgo.io/ent/dialect"
	entDialectSQL "entgo.io/ent/dialect/sql"
	"github.com/XSAM/otelsql"
	_ "github.com/jackc/pgx/v5/stdlib" // pgx database driver
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
)

func GetPGDriver(databaseURL string) (*entDialectSQL.Driver, error) {
	// TODO: inject trace and metrics provider
	database, err := otelsql.Open("pgx", databaseURL, otelsql.WithAttributes(
		semconv.DBSystemPostgreSQL,
	))
	if err != nil {
		return nil, err
	}

	return entDialectSQL.OpenDB(dialect.Postgres, database), nil
}
