package migrate

import (
	"embed"

	_ "github.com/golang-migrate/migrate/v4/database/clickhouse"
)

//go:embed clickhouse_migrations
var clickHouseMigrations embed.FS

const (
	ClickHouseMigrationsTable = "schema_om_clickhouse"
)

var ClickHouseMigrationsConfig = MigrationsConfig{
	FS:             clickHouseMigrations,
	FSPath:         "clickhouse_migrations",
	StateTableName: ClickHouseMigrationsTable,
}
