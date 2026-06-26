# migrate

<!-- archie:ai-start -->

> Database migration engine and the authoritative store of golang-migrate-format SQL migrations plus their up/down/up data-migration tests. Wraps golang-migrate/v4 with an embedded migrations FS (state table schema_om), supports an ignore marker, and is the runtime entrypoint invoked on server startup when postgres.autoMigrate is ent or migration.

## Patterns

**Migrate wraps golang-migrate via type alias** — Migrate embeds *goMigrate (a type alias for migrate.Migrate) so Migrate.Migrate(version) can call goMigrate.Migrate without a name clash, and every wrapper (Up/Down/Migrate) routes through filterErrNoChange to swallow migrate.ErrNoChange (`func (m *Migrate) Up() error { return m.filterErrNoChange(m.goMigrate.Up()) }`)
**Config validated, no slog.Default fallback** — MigrateOptions and MigrationsConfig each Validate() by collecting into []error and returning errors.Join(...); Logger is a required *slog.Logger, and ConnectionString, FS, FSPath, StateTableName are all required (`if m.Logger == nil { errs = append(errs, errors.New("logger is required")) }`)
**Embedded migrations + SourceWrapper ignore marker** — //go:embed migrations supplies OMMigrationsConfig; New wraps the FS in NewSourceWrapper which recursively reads dirs and drops any file whose lines start with '-- migration:ignore' (IgnoreMarker) (`const IgnoreMarker = "-- migration:ignore"`)
**Migration table name injected via query param** — setMigrationTableName parses the connection URL and sets x-migrations-table to StateTableName (schema_om) so OpenMeter's migrations don't collide with Ent's default state table (`values.Set("x-migrations-table", tableName)`)
**Data-migration tests use runner/stops harness** — Tests in package migrate_test build runner{stops: stops{...}} where each stop has version, direction (directionUp/directionDown), and action(t, db); the runner migrates to each up version ascending, runs the action, then Up, descending downs, purge, Down, Up to prove reversibility (`runner{stops: stops{{version: 20260511120000, direction: directionUp, action: func(t *testing.T, db *sql.DB){...}}}}.Test(t)`)
**Raw *sql.DB for fixtures, never an ORM** — Test actions insert/assert with db.Exec/db.QueryRow on testutils.InitPostgresDB(t).PGDriver.DB() because any ORM would only know the latest schema; fixtures use ulid.Make().String() ids and explicit column lists valid at that version (`_, err := db.Exec(`INSERT INTO features (namespace, id, ...) VALUES ($1, $2, ...)`, ...)`)
**View parity guarded by viewgen** — view_parity_test.go regenerates views via viewgen.GenerateSQL, strips CREATE/DROP VIEW out of the real migrations, and asserts pg_get_viewdef + information_schema.columns match between a fully-migrated DB and a generated-schema DB (`sql, err := viewgen.GenerateSQL("../../openmeter/ent/schema")`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `migrate.go` | Core Migrate type, New() constructor, OMMigrationsConfig with embedded migrations FS, validation, WaitForMigrationJob, LatestVersion, table-name injection, Close helpers | Always go through filterErrNoChange; never call goMigrate.Up directly. WaitForMigrationJob fails on a dirty DB. MigrationsTable is fixed to schema_om |
| `fs.go` | SourceWrapper implementing fs.ReadDirFS/ReadFileFS that flattens nested dirs and skips files starting with the '-- migration:ignore' IgnoreMarker | ReadDir recurses and scans every file's content; a misplaced ignore marker silently drops a migration from the embedded source |
| `migrate_test.go` | Defines the runner/stops/stop harness, directionUp/directionDown constants, ups()/downs() ordering, and purgeDB. TestUpDownUp runs the full forward/back/forward cycle with no stops | purgeDB TRUNCATE CASCADE skips the schema_om state table; down-then-up must succeed. Stops sorted ascending for ups, descending for downs |
| `view_parity_test.go` | Asserts hand-written view DDL in migrations matches viewgen output by diffing column metadata and normalized view definitions across two fresh DBs | buildMigrationsWithoutViews strips any statement matching the VIEW regex; a view added to migrations but not the Ent schema (or vice versa) breaks this test |
| `*_test.go (data migrations)` | One file per non-trivial data migration (dedupe_tax_codes, feature_meter_id, flatfee_runs, llmcost_normalize_providers, ledger_tax_behavior, productcatalog, feature_advanced_meter_group_by_filters) asserting before/after row state | Fixtures must match the exact column set valid at the targeted version; pass full JSON strings as bound params ($n::jsonb) to dodge Postgres char(26)-vs-jsonb parameter type inference errors |
| `generate-sqlc-testdata.sh` | Spins up postgres, migrates a scratch DB to VERSION, pg_dumps the schema, and runs sqlc generate into testdata/sqlcgen/<VERSION> | Requires the nix dev shell (docker, migrate, pg_dump, sqlc) and uses x-migrations-table=schema_om to match the runtime migration table |
| `ledger_tax_behavior_test.go` | Documents a rollback guard: a down migration that fails loudly when V2 routing-key rows still exist rather than silently corrupting routes | Irreversible-with-data down migrations should RAISE an explicit error; the test asserts the exact message |

## Anti-Patterns

- Using an ORM or ent client inside migration tests; only raw *sql.DB sees the historical schema at a given version
- Hand-editing tools/migrate/views.sql — it carries a 'Code generated by viewgen, DO NOT EDIT.' header and is overwritten by make generate-view-sql
- Calling goMigrate.Up/Down/Migrate directly and re-leaking migrate.ErrNoChange instead of going through the Migrate wrapper methods
- Writing an irreversible down migration without a loud guard (see ledger_tax_behavior) — TestUpDownUp purges then runs Down then Up and will fail
- Adding a CREATE VIEW to a migration without keeping the Ent view schema in sync, breaking view_parity_test's column/definition diff

## Decisions

- **Replace ent.Schema.Create with a lightweight golang-migrate wrapper over an embedded SQL FS** — Gives explicit, reviewable, reversible SQL migrations with a dedicated schema_om state table instead of opaque auto-create, while still being generated from Ent schema diffs via Atlas
- **Test every non-trivial data migration with a forward/backward/forward stops harness against real Postgres** — Data transforms (backfills, dedup, normalization, JSONB rewrites) can't be verified by schema diff alone; the runner proves both the data result and full reversibility
- **Generate view DDL separately (viewgen) and validate parity rather than relying on Atlas diff** — Ent ent.View schemas don't appear in generated migrate metadata, so view SQL must be authored/checked independently to stay consistent with the Ent schema

## Example: Constructing a migrator and running a targeted data-migration verification in a test

```
import (
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/tools/migrate"
)

testDB := testutils.InitPostgresDB(t)
defer testDB.PGDriver.Close()

migrator, err := migrate.New(migrate.MigrateOptions{
	ConnectionString: testDB.URL,
	Migrations:       migrate.OMMigrationsConfig,
	Logger:           testutils.NewLogger(t),
})
require.NoError(t, err)
defer func() { err1, err2 := migrator.Close(); require.NoError(t, errors.Join(err1, err2)) }()
// ...
```

<!-- archie:ai-end -->
