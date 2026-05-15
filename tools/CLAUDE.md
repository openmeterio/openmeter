# tools

<!-- archie:ai-start -->

> Operational boundary between schema authorship and runtime startup: tools/migrate owns Atlas-generated SQL migrations, the golang-migrate runtime wrapper, and the viewgen sub-tool for ClickHouse view DDL. wait-for-compose.sh is a CI readiness gate. Nothing here contains business logic.

## Patterns

**SourceWrapper for migration:ignore filtering** — tools/migrate/fs.go wraps the embedded migration filesystem and filters out files tagged migration:ignore so Atlas lint files do not confuse golang-migrate. (`migrate.New("iofs://migrations", db) uses the SourceWrapper to strip non-SQL files before applying.`)
**OMMigrationsConfig as canonical migration config** — All migration runtime configuration (DB URL, migration table name) flows through OMMigrationsConfig rather than ad-hoc viper reads. (`cfg := tools/migrate.OMMigrationsConfig{DatabaseURL: conf.Postgres.URL()}`)
**Migration stop tests via runner/stops pattern** — tools/migrate/migrate_test.go uses a runner/stops table pattern: each stop specifies an up-to version and a validation action on raw *sql.DB, with purgeDB between down and up passes to ensure idempotence. (`stops := []migrationStop{{version: 20, action: func(db *sql.DB) error { /* assert schema shape */ }}}`)
**viewgen for ClickHouse view DDL** — tools/migrate/cmd/viewgen generates ClickHouse view SQL from Ent view schemas. Run it separately from `atlas migrate diff` because Atlas does not diff ent.View schemas. (`go run ./tools/migrate/cmd/viewgen  # writes tools/migrate/views.sql`)
**wait-for-compose polling pattern** — wait-for-compose.sh polls docker inspect health/state in a 60-attempt/2s loop before CI test steps proceed. Containers with no healthcheck fall through to state-based check. (`./tools/wait-for-compose.sh postgres svix redis`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `tools/migrate/migrate.go` | Runtime migration wrapper: embeds migrations FS, applies via golang-migrate, exposes Migrate(ctx, db) used at binary startup. | Do not call atlas migrate at runtime — only golang-migrate is used here. OMMigrationsConfig is the only config input. |
| `tools/migrate/fs.go` | SourceWrapper that strips migration:ignore files from the embedded FS before golang-migrate sees them. | Any new file added to migrations/ that is not a .up.sql/.down.sql must be tagged migration:ignore or it will confuse golang-migrate. |
| `tools/migrate/migrate_test.go` | Stop tests that verify schema shape at specific migration versions using raw *sql.DB. | Use raw *sql.DB, not Ent — Ent only understands current schema. Call purgeDB between up and down passes. |
| `tools/migrate/view_parity_test.go` | Asserts that views.sql matches what viewgen would generate from the current Ent schema. | If ent.View schema changes, run viewgen and commit the updated views.sql — this test will fail otherwise. |
| `tools/migrate/cmd/viewgen/main.go` | Standalone tool that reads Ent schema paths and writes ClickHouse view DDL to views.sql. | Do not import app/common or openmeter/ domain packages — this tool only needs Ent schema paths. |
| `tools/wait-for-compose.sh` | CI readiness gate: blocks until listed docker compose services are healthy or running. | Exit 1 on unhealthy state; timeout after 120s (60 attempts × 2s). Containers without healthcheck use state-based check. |

## Anti-Patterns

- Hand-editing any file in tools/migrate/migrations/ — Atlas owns the chain and atlas.sum will fail CI validation
- Calling `atlas migrate diff` expecting new ent.View schemas to appear — views require viewgen, not Atlas diff
- Using Ent or any ORM inside migration stop test actions — raw *sql.DB is required because ORM only understands current schema
- Adding a new migration stop test outside the runner/stops table pattern — it will not run purgeDB/down/up cycle
- Importing app/common or openmeter/ domain packages from tools/migrate/cmd/viewgen/main.go

## Decisions

- **golang-migrate wraps Atlas-generated SQL rather than using Atlas CLI at runtime.** — Atlas CLI is a dev/CI tool; golang-migrate provides a stable embedded Go runtime for applying migrations at binary startup without Atlas binary dependency.
- **viewgen is a separate tool from Atlas migration generation.** — Atlas does not diff ent.View schemas into SQL migrations; ClickHouse view DDL requires custom generation and must be applied as an explicit SQL migration.
- **wait-for-compose.sh is a standalone bash script rather than a Go helper.** — It must run before any Go binary is available; bash docker-inspect polling has no runtime dependencies.

<!-- archie:ai-end -->
