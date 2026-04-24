# testutils

<!-- archie:ai-start -->

> Shared, app/common-independent test helpers for the entire openmeter/ tree: Postgres test database provisioning via pgtestdb, async assertion helpers, deterministic name generation, logger factories, and time utilities. Must never import app/common to avoid import cycles.

## Patterns

**InitPostgresDB skips when POSTGRES_HOST unset** — InitPostgresDB reads os.Getenv("POSTGRES_HOST") and calls t.Skip() if empty. Tests that call this will be silently skipped in environments without Postgres rather than failing. (`db := testutils.InitPostgresDB(t) // skips if POSTGRES_HOST not set`)
**TestDB bundles EntDriver + PGDriver + URL** — InitPostgresDB returns a *TestDB with both the Ent driver (EntPostgresDriver) and raw pgdriver.Driver for tests that need either. Call db.Close(t) in t.Cleanup to release both. (`db := testutils.InitPostgresDB(t)
t.Cleanup(func() { db.Close(t) })
client := db.EntDriver.Client()`)
**NoopMigrator for schema-less test databases** — The default migrator is NoopMigrator (no-op Hash/Migrate/Prepare/Verify). Pass WithMigrator(myMigrator) to apply real schema migrations; otherwise pgtestdb clones a blank template. (`db := testutils.InitPostgresDB(t, testutils.WithMigrator(&myMigrator{}))`)
**NameGenerator produces Key + Name pairs** — testutils.NameGenerator.Generate() returns a GeneratedName with a slug Key (lowercase, hyphen-separated) and a Title-case Name. Use Key for database identifiers, Name for display values. (`n := testutils.NameGenerator.Generate()
// n.Key = "happy-tiger", n.Name = "Happy Tiger"`)
**t.Context() throughout — no context.Background()** — All helpers use t.Context() (e.g. pgdriver.NewPostgresDriver(t.Context(), ...)) so test cancellation propagates correctly. Never substitute context.Background() in new test helpers. (`postgresDriver, err := pgdriver.NewPostgresDriver(t.Context(), dbConf.URL())`)
**EventuallyWithTf wraps require.EventuallyWithTf with error capture** — Use EventuallyWithTf when the polling function needs to report its last error on final failure. Pass a saveErr closure to record the error; it is surfaced as the EventuallyWithTf format argument. (`testutils.EventuallyWithTf(t, func(c *assert.CollectT, saveErr func(err any)) {
    saveErr(doCheck(c))
}, 5*time.Second, 100*time.Millisecond)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `pg_driver.go` | Core Postgres test provisioning: InitPostgresDB, TestDB struct, option types. The most-imported file in the package. | Importing app/common here — this would create an import cycle. Build Ent clients directly from entdriver.NewEntPostgresDriver(pgDriver.DB()). |
| `namegen.go` | Deterministic (per-run) human-readable name generator seeded at package init time. | Using NameGenerator across parallel tests without locking — the underlying generator is not goroutine-safe if multiple goroutines call Generate() concurrently; prefer one call per test. |
| `async.go` | EventuallyWithTf helper for async assertions that need last-error capture. | Using plain require.Eventually when the polling body produces a meaningful error — EventuallyWithTf surfaces that error on failure. |
| `logger.go` | NewLogger (slog.Default) and NewDiscardLogger (no-op) factories. NewDiscardLogger uses an inline discardHandler to work with Go versions before 1.24. | Using slog.New(slog.NewTextHandler(os.Stdout, nil)) inline in tests — use NewLogger(t) instead for consistency. |
| `time.go` | GetRFC3339Time (fatal on parse error) and TimeEqualsApproximately for time comparison with tolerance. | Using time.Parse directly in tests — GetRFC3339Time calls t.Fatalf on error, keeping test code clean. |

## Anti-Patterns

- Importing app/common or any Wire provider from this package — breaks the test-helper isolation rule
- Using context.Background() in helper functions — always use t.Context()
- Skipping db.Close(t) in test teardown — leaks Ent and PG connections
- Implementing domain-specific test logic here — domain test fixtures belong in openmeter/<domain>/testutils/
- Editing pg_driver.go to add real schema migration by default — use WithMigrator option to keep the base helper schema-agnostic

## Decisions

- **NoopMigrator as default migrator in InitPostgresDB** — Most test suites manage their own schema (via Ent auto-migrate or explicit DDL); a no-op default avoids coupling this shared helper to the Atlas migration chain.
- **TestDB bundles both EntDriver and PGDriver** — Some tests need the Ent typed client; others need the raw pgx pool (e.g. for direct SQL assertions). Providing both avoids callers re-constructing drivers from the URL.

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
