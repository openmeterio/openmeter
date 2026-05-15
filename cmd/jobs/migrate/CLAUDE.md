# migrate

<!-- archie:ai-start -->

> Cobra sub-command for running database migrations via the Wire-provided Migrator; validates the migration mode flag before executing and explicitly rejects the 'job' mode to prevent circular migration loops.

## Patterns

**Override App.Migrator.Config at runtime before calling Migrate** — The --mode flag value is parsed via config.AutoMigrate() and written to internal.App.Migrator.Config.AutoMigrate before calling Migrate. This is the only place in jobs that mutates App state after initialization. (`internal.App.Migrator.Config.AutoMigrate = migrationMode; err := internal.App.Migrator.Migrate(cmd.Context())`)
**Validate mode before execution with explicit rejection of 'job' mode** — Uses config.AutoMigrate() to parse and validate the mode string. Explicitly rejects AutoMigrateMigrationJob mode to prevent circular use of the jobs binary for job-mode migration. (`if !migrationMode.Enabled() { return fmt.Errorf("migration mode is disabled") }
if migrationMode == config.AutoMigrateMigrationJob { return fmt.Errorf("migration mode cannot be job") }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `migrate.go` | Implements the 'migrate' Cobra command; wraps common.Migrator with mode validation. Default mode is 'migration' (config.AutoMigrateMigration). | The --mode flag accepts only 'ent' and 'migration'; 'job' is explicitly rejected. Do not add migration logic (DDL, data transforms) here — those belong in tools/migrate/migrations/. |

## Anti-Patterns

- Calling tools/migrate directly instead of going through internal.App.Migrator
- Adding migration logic (DDL, data transforms) to this command — those belong in tools/migrate/migrations/
- Allowing the 'job' migration mode through without rejection

## Decisions

- **Migration mode is overridden on App.Migrator.Config rather than constructing a new Migrator.** — The Wire-provided Migrator already has the correct DB client and logger; overriding just the mode field reuses all that wiring without duplication.

<!-- archie:ai-end -->
