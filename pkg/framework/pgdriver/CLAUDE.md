# pgdriver

<!-- archie:ai-start -->

> Constructs the application's PostgreSQL driver: NewPostgresDriver builds a pgxpool-backed *sql.DB wrapped with OpenTelemetry (otelsql) tracing/metrics and optional pgxpool observability, exposing Driver.DB() and Driver.Close().

## Patterns

**Functional options via Option interface** — Configuration is supplied through Option values (WithTracerProvider, WithMeterProvider, WithMetricMeter, WithSpanOptions, WithLockTimeout) implemented as optionFunc applied to an internal options struct. (`pgdriver.NewPostgresDriver(ctx, url, pgdriver.WithLockTimeout(3*time.Second))`)
**lock_timeout via pgx RuntimeParams** — WithLockTimeout sets connConfig.ConnConfig.RuntimeParams["lock_timeout"] in milliseconds — this is what lockr relies on for PG-side lock timeouts instead of context cancellation. (`o.connConfig.ConnConfig.RuntimeParams["lock_timeout"] = fmt.Sprintf("%d", timeout.Milliseconds())`)
**pgxpool-backed otelsql DB** — The pool is created via pgxpool.NewWithConfig, then otelsql.OpenDB(pgxstdlib.GetPoolConnector(pool)) wraps it; MaxIdleConns is set to 0 because idle connections are managed by the pgx pool, not database/sql. (`db := otelsql.OpenDB(pgxstdlib.GetPoolConnector(pool), o.otelOptions...); db.SetMaxIdleConns(0)`)
**Opt-in pool metrics** — When WithMetricMeter is provided, pgxpoolobserver.ObservePoolMetrics(meter, pool) registers pool gauges before returning the driver. (`if o.metricMeter != nil { pgxpoolobserver.ObservePoolMetrics(o.metricMeter, pool) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `driver.go` | Entire package: Option machinery, options/Driver structs, and NewPostgresDriver constructor. | db.SetMaxIdleConns(0) is intentional — do not change it; idle pooling is pgx's job. Close() closes the pool (and thus the *sql.DB) — there is no separate DB close. |

## Anti-Patterns

- Constructing the *sql.DB or pgxpool directly instead of via NewPostgresDriver (loses otelsql instrumentation and the MaxIdleConns=0 invariant).
- Setting MaxIdleConns to a non-zero value, which double-manages idle connections against the pgx pool.
- Relying on context timeouts for PG lock waits instead of WithLockTimeout (see lockr).

## Decisions

- **lock_timeout is configured as a pgx RuntimeParam** — Provides server-side advisory-lock acquisition timeouts (used by lockr) that keep the connection usable, unlike client-side context cancellation.
- **MaxIdleConns forced to 0 on the otelsql DB** — The pgxpool already owns connection lifecycle; letting database/sql also pool idle connections would conflict (per pgx stdlib guidance).

## Example: Create an instrumented Postgres driver

```
import "github.com/openmeterio/openmeter/pkg/framework/pgdriver"

drv, err := pgdriver.NewPostgresDriver(ctx, dbURL,
  pgdriver.WithMeterProvider(mp),
  pgdriver.WithMetricMeter(meter),
  pgdriver.WithLockTimeout(3*time.Second),
)
if err != nil { return err }
defer drv.Close()
sqlDB := drv.DB()
```

<!-- archie:ai-end -->
