# testutils

<!-- archie:ai-start -->

> Shared, app/common-independent test helpers for the entire openmeter/ tree: Postgres test database provisioning via pgtestdb, async assertion helpers, deterministic name generation, logger factories, and time utilities. Must never import app/common to avoid import cycles.

## Patterns

**InitPostgresDB skips when POSTGRES_HOST unset** — InitPostgresDB reads os.Getenv("POSTGRES_HOST") and calls t.Skip() if empty. Tests are silently skipped in environments without Postgres rather than failing. (`db := testutils.InitPostgresDB(t); t.Cleanup(func() { db.Close(t) })`)
**TestDB bundles EntDriver + PGDriver + URL** — InitPostgresDB returns a *TestDB with both the Ent driver and raw pgdriver.Driver. Call db.Close(t) in t.Cleanup to release both. (`db := testutils.InitPostgresDB(t); t.Cleanup(func() { db.Close(t) }); client := db.EntDriver.Client()`)
**NoopMigrator default; WithMigrator for schema-applying tests** — Default migrator is NoopMigrator. Pass WithMigrator(myMigrator) to apply real schema migrations; otherwise pgtestdb clones a blank template. (`db := testutils.InitPostgresDB(t, testutils.WithMigrator(&myMigrator{}))`)
**t.Context() throughout — no context.Background()** — All helpers use t.Context() so test cancellation propagates. Never substitute context.Background() in new test helpers. (`postgresDriver, err := pgdriver.NewPostgresDriver(t.Context(), dbConf.URL(), o.driverOptions...)`)
**EventuallyWithTf wraps require.EventuallyWithTf with error capture** — Use EventuallyWithTf when the polling function must report its last error on final failure. Pass a saveErr closure to record the error. (`testutils.EventuallyWithTf(t, func(c *assert.CollectT, saveErr func(err any)) { saveErr(doCheck(c)) }, 5*time.Second, 100*time.Millisecond)`)
**NameGenerator produces Key + Name pairs** — NameGenerator.Generate() returns a GeneratedName with a slug Key (lowercase, hyphen-separated) and a Title-case Name. Use Key for DB identifiers, Name for display. (`n := testutils.NameGenerator.Generate() // n.Key = "happy-tiger", n.Name = "Happy Tiger"`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `pg_driver.go` | Core Postgres test provisioning: InitPostgresDB, TestDB, option types (WithPostgresConfig, WithMigrator, WithDriverOptions). Most-imported file. | Importing app/common here creates an import cycle. Build Ent clients directly from entdriver.NewEntPostgresDriver(postgresDriver.DB()). |
| `namegen.go` | Deterministic (per-run) human-readable name generator seeded at package init using forscht/namegen. | Calling NameGenerator.Generate() concurrently from multiple goroutines without locking — the underlying generator is not goroutine-safe. |
| `async.go` | EventuallyWithTf helper for async assertions needing last-error capture via sync.Map. | Using plain require.Eventually when the polling body produces a meaningful error — EventuallyWithTf surfaces it on failure. |
| `logger.go` | NewLogger (slog.Default) and NewDiscardLogger (inline discardHandler) factories for test logging. | Using slog.New(slog.NewTextHandler(os.Stdout, nil)) inline in tests — use NewLogger(t) for consistency. |
| `time.go` | GetRFC3339Time (t.Fatalf on parse error) and TimeEqualsApproximately for tolerant time comparison. | Using time.Parse directly in tests — GetRFC3339Time wraps the error into t.Fatalf, keeping test code clean. |

## Anti-Patterns

- Importing app/common or any Wire provider from this package — breaks test-helper isolation and creates import cycles
- Using context.Background() in helper functions — always use t.Context()
- Skipping db.Close(t) in test teardown — leaks both Ent and PG connections
- Implementing domain-specific test logic here — domain fixtures belong in openmeter/<domain>/testutils/
- Editing pg_driver.go to apply schema migrations by default — use the WithMigrator option

## Decisions

- **NoopMigrator as default migrator in InitPostgresDB** — Most test suites manage their own schema (Ent auto-migrate or explicit DDL); a no-op default avoids coupling this shared helper to the Atlas migration chain or any domain schema.
- **TestDB bundles both EntDriver and PGDriver** — Some tests need the Ent typed client; others need the raw pgx pool (e.g. direct SQL assertions). Providing both avoids callers reconstructing drivers from the URL string.

## Example: Standard Postgres-backed unit test setup

```
import (
	"testing"
	"github.com/openmeterio/openmeter/openmeter/testutils"
)

func TestSomething(t *testing.T) {
	db := testutils.InitPostgresDB(t) // skips if POSTGRES_HOST unset
	t.Cleanup(func() { db.Close(t) })

	entClient := db.EntDriver.Client()
	// use entClient for domain adapter construction
}
```

<!-- archie:ai-end -->
