# testutils

<!-- archie:ai-start -->

> Shared, framework-light test helpers used across nearly every domain package: Postgres test DB provisioning, loggers, deterministic name generation, time helpers, and async polling. Constraint: keep it dependency-light (only pkg/framework) so it never creates import cycles with app/common wiring.

## Patterns

**pgtestdb-backed isolated DBs** — InitPostgresDB provisions a per-test template-cloned Postgres DB via pgtestdb.Custom, wrapping it in pgdriver + entdriver, returned as TestDB with a Close(t) cleanup. Skips the test when POSTGRES_HOST is unset. (`db := testutils.InitPostgresDB(t); defer db.Close(t)`)
**Functional options for DB setup** — InitPostgresDB takes Option values (WithPostgresConfig, WithMigrator, WithDriverOptions) implemented via the optionFunc/apply pattern. (`InitPostgresDB(t, WithMigrator(&NoopMigrator{}))`)
**t.Helper + t.Context discipline** — Helpers call t.Helper() and use t.Context() (not context.Background) so failures attribute to the caller and lifecycle ties to the test. (`func NewLogger(t testing.TB) *slog.Logger { t.Helper(); ... }`)
**Deterministic-ish name generation** — NameGenerator (package-level singleton) yields a GeneratedName{Key, Name} where Key is the lowercased, dash-joined slug of Name, via forscht/namegen adjective+animal dictionaries. (`n := testutils.NameGenerator.Generate(); n.Key, n.Name`)
**Async assertion wrapper** — EventuallyWithTf wraps require.EventuallyWithTf, capturing the last saved error via a sync.Map so the eventual failure message carries it. (`EventuallyWithTf(t, fn, wait, interval)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `pg_driver.go` | TestDB type plus InitPostgresDB; NoopMigrator (currently a no-op, see `TODO: fix migrations`). | Default migrator is NoopMigrator — schema is NOT migrated by default; callers needing a schema must pass WithMigrator. Test SKIPS (not fails) when POSTGRES_HOST is empty. |
| `logger.go` | NewLogger (slog.Default) and NewDiscardLogger for quiet tests. | Contains a local discardHandler copy marked TODO-removable once min Go has slog.DiscardHandler. |
| `namegen.go` | Random unique Key/Name generator seeded by time for test fixtures. | Seeded with time.Now().UnixNano() — names are not reproducible across runs; do not assert on exact values. |
| `time.go` | GetRFC3339Time parse helper and TimeEqualsApproximately tolerance assertion. | Both t.Fatalf on mismatch; GetRFC3339Time expects strict RFC3339 input. |
| `async.go` | EventuallyWithTf polling helper around testify. | saveErr stores into a sync.Map under a fixed key; only the last error is surfaced. |

## Anti-Patterns

- Importing app/common or domain wiring here — would create test-only import cycles; keep deps to pkg/framework.
- Assuming InitPostgresDB ran migrations (default NoopMigrator does nothing).
- Asserting exact generated names from NameGenerator (time-seeded, non-deterministic).
- Using context.Background() in helpers instead of t.Context().

## Decisions

- **Use pgtestdb template-cloning for per-test isolated databases.** — Gives each test a fresh, fast-cloned Postgres without cross-test contamination, while skipping cleanly when no Postgres host is configured.
- **Keep the package framework-only.** — It is imported by almost every domain package's tests; broad dependencies would risk import cycles and slow compilation.

## Example: Spin up an isolated Postgres-backed test DB

```
import "github.com/openmeterio/openmeter/openmeter/testutils"

func TestX(t *testing.T) {
	db := testutils.InitPostgresDB(t)
	defer db.Close(t)
	client := db.EntDriver.Client()
	_ = client
}
```

<!-- archie:ai-end -->
