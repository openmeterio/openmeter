# migrate

<!-- archie:ai-start -->

> Cobra sub-command for running database migrations via the Wire-provided Migrator; validates the requested migration mode before executing, and refuses the 'job' mode to avoid self-referential migration loops.

## Patterns

**Mutate internal.App.Migrator.Config before calling Migrate** — The migration mode flag overrides App.Migrator.Config.AutoMigrate at runtime rather than wiring a separate migrator. This is the only place in jobs that mutates App state. (`internal.App.Migrator.Config.AutoMigrate = migrationMode; err := internal.App.Migrator.Migrate(cmd.Context())`)
**Validate mode before execution** — Uses config.AutoMigrate() to parse and validate the mode string; explicitly rejects AutoMigrateMigrationJob mode to prevent circular use. (`if !migrationMode.Enabled() { return fmt.Errorf("migration mode is disabled") }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `migrate.go` | Implements the 'migrate' Cobra command; wraps common.Migrator with mode validation. Default mode is 'migration' (config.AutoMigrateMigration). | The --mode flag accepts only 'ent' and 'migration'; 'job' mode is explicitly rejected. Do not add business logic beyond mode selection and Migrate() invocation. |

## Anti-Patterns

- Calling tools/migrate directly instead of going through internal.App.Migrator
- Adding migration logic (DDL, data transforms) to this command — those belong in tools/migrate/migrations/

## Decisions

- **Migration mode is overridden on App.Migrator.Config rather than constructing a new migrator.** — The Wire-provided Migrator already has the correct DB client and logger; overriding just the mode reuses all that wiring without duplication.

<!-- archie:ai-end -->
