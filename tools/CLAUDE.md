# tools

<!-- archie:ai-start -->

> Operational boundary between schema authorship and runtime startup. tools/migrate owns the Atlas-generated SQL migration chain, the golang-migrate runtime wrapper applied at binary startup, and the viewgen sub-tool that generates ClickHouse view DDL outside Atlas; wait-for-compose.sh is a CI readiness gate. Primary constraint: migrations/ SQL and atlas.sum are Atlas-owned and must never be hand-edited.

## Patterns

**golang-migrate over embedded Atlas SQL** — Migrations are embedded and applied at startup via golang-migrate (migrate.go), not the Atlas CLI; OMMigrationsConfig is the only config input (DB URL, migration table name). (`cfg := tools/migrate.OMMigrationsConfig{DatabaseURL: conf.Postgres.URL()}`)
**SourceWrapper for migration:ignore filtering** — fs.go wraps the embedded migration filesystem and strips files tagged migration:ignore so Atlas lint artefacts do not confuse golang-migrate. (`migrate.New("iofs://migrations", db) uses the SourceWrapper to drop non-SQL files; filterErrNoChange swallows migrate.ErrNoChange.`)
**Runner/stops migration tests on raw *sql.DB** — migrate_test.go drives a table of stops {version, action} validating schema shape on raw *sql.DB, with purgeDB between down and up passes to assert idempotence. (`stops := []migrationStop{{version: 20, action: func(db *sql.DB) error { /* assert schema */ }}}`)
**viewgen for ClickHouse view DDL with parity test** — tools/migrate/cmd/viewgen generates view SQL from Ent view schemas (Atlas does not diff ent.View); view_parity_test.go fails if views.sql diverges from viewgen output. (`go run ./tools/migrate/cmd/viewgen  # writes tools/migrate/views.sql`)
**wait-for-compose polling gate** — wait-for-compose.sh polls docker inspect health/state for listed services (60 attempts x 2s) before CI test steps run, falling back to state-based checks for containers without healthchecks. (`./tools/wait-for-compose.sh postgres svix redis`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `tools/migrate/migrate.go` | Runtime migration wrapper: embeds the migrations FS, applies via golang-migrate, exposes Migrate(ctx, db) used at binary startup. | Do not call Atlas at runtime; OMMigrationsConfig is the only config input. |
| `tools/migrate/fs.go` | SourceWrapper stripping migration:ignore files from the embedded FS before golang-migrate sees them. | Any non-.up.sql/.down.sql file added to migrations/ must be tagged migration:ignore or it will confuse golang-migrate. |
| `tools/migrate/migrate_test.go` | Stop tests verifying schema shape at specific migration versions using raw *sql.DB. | Use raw *sql.DB, not Ent (Ent only understands the current schema); call purgeDB between up and down passes. |
| `tools/migrate/view_parity_test.go` | Asserts views.sql matches what viewgen would generate from the current Ent schema. | When ent.View schema changes, run viewgen and commit the updated views.sql or this test fails. |
| `tools/migrate/cmd/viewgen/main.go` | Standalone tool reading Ent schema paths and writing ClickHouse view DDL to views.sql. | Do not import app/common or openmeter/ domain packages — it needs only Ent schema paths. |
| `tools/migrate/views.sql` | Generated ClickHouse view DDL applied as an explicit SQL migration. | Generated, not hand-edited; run `make generate-view-sql` to regenerate. |
| `tools/wait-for-compose.sh` | CI readiness gate blocking until listed docker compose services are healthy or running. | Exits 1 on unhealthy state and times out after ~120s (60 x 2s); standalone bash so it runs before any Go binary is available. |

## Anti-Patterns

- Hand-editing any file in tools/migrate/migrations/ — Atlas owns the chain and atlas.sum fails CI validation.
- Calling `atlas migrate diff` expecting it to pick up a new ent.View schema — Atlas does not diff views; use viewgen.
- Using Ent or any ORM inside migration stop-test actions — raw *sql.DB is required because the ORM only understands the current schema version.
- Adding a migration stop test outside the runner/stops pattern — it skips the purgeDB/down/up cycle and can leave the DB dirty for other tests.
- Hand-editing tools/migrate/views.sql — it is generated; run `make generate-view-sql`.

## Decisions

- **golang-migrate wraps Atlas-generated SQL rather than using the Atlas CLI at runtime.** — Atlas is a dev/CI tool; golang-migrate gives a stable embedded Go runtime for applying migrations at binary startup with no Atlas binary dependency.
- **viewgen is a separate tool from Atlas migration generation.** — Atlas does not diff ent.View schemas into SQL migrations; ClickHouse view DDL requires custom generation applied as an explicit SQL migration.
- **wait-for-compose.sh is a standalone bash script rather than a Go helper.** — It must run before any Go binary is available; bash docker-inspect polling has no runtime dependencies.

## Example: Applying embedded migrations at binary startup

```
import migrate "github.com/openmeterio/openmeter/tools/migrate"

cfg := migrate.OMMigrationsConfig{DatabaseURL: conf.Postgres.URL()}
if err := migrate.Migrate(ctx, cfg); err != nil {
	return fmt.Errorf("run migrations: %w", err)
}
```

<!-- archie:ai-end -->
