# pgdriver

<!-- archie:ai-start -->

> Constructs the project-wide *sql.DB from a pgxpool.Pool with OTel tracing (XSAM/otelsql), optional metric.Meter pool observation (pgxpoolobserver), and an optional lock_timeout runtime parameter for lockr; exposes Driver.DB() for Ent and lockr consumers.

## Patterns

**Option interface for functional configuration** — All optional settings (TracerProvider, MeterProvider, MetricMeter, SpanOptions, LockTimeout) are passed as Option values, each implementing apply(*options). Callers compose options without a config struct. (`pgdriver.NewPostgresDriver(ctx, url, pgdriver.WithLockTimeout(3*time.Second), pgdriver.WithTracerProvider(tp))`)
**MaxIdleConns must be 0 — pgx pool manages idle connections** — otelsql.OpenDB wraps pgxstdlib.GetPoolConnector(pool). db.SetMaxIdleConns(0) is mandatory because pgx manages idle connections; a non-zero Go idle pool duplicates connection management. (`db.SetMaxIdleConns(0) // set after otelsql.OpenDB`)
**WithLockTimeout sets lock_timeout via RuntimeParams** — WithLockTimeout sets lock_timeout in ConnConfig.RuntimeParams as a millisecond string, enabling PostgreSQL-side advisory lock timeouts for lockr without context cancellation destroying the connection. (`o.connConfig.ConnConfig.RuntimeParams["lock_timeout"] = fmt.Sprintf("%d", timeout.Milliseconds())`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `driver.go` | Single file with NewPostgresDriver, the Driver struct, and DB()/Close() methods. | Close() calls pool.Close() but NOT db.Close() — callers holding *sql.DB must not call db.Close() independently. Pool shutdown is immediate; ensure in-flight queries complete first. |

## Anti-Patterns

- Setting MaxIdleConns > 0 on the returned *sql.DB — pgx pool manages idle connections; a non-zero Go idle pool causes duplicate management and resource waste.
- Calling Driver.Close() before all in-flight queries complete — pool.Close() is immediate and does not drain.
- Using context.WithTimeout for lock acquisition on connections from this pool — use pgdriver.WithLockTimeout to avoid pgx connection cancellation on ctx deadline.

## Decisions

- **Wrap pgxpool behind database/sql via pgxstdlib.** — Ent ORM and lockr both require database/sql; pgxpool gives fine-grained pool control (RuntimeParams, metrics) while pgxstdlib bridges the two APIs without reimplementing the driver.
- **lock_timeout in RuntimeParams rather than per-query SET.** — Setting lock_timeout at connection-config level applies it to all advisory lock acquisitions on that connection without each caller issuing a SET, making lockr usage transparent.

## Example: Create a Driver with OTel tracing, pool metrics, and lock timeout for Ent and lockr

```
import "github.com/openmeterio/openmeter/pkg/framework/pgdriver"

drv, err := pgdriver.NewPostgresDriver(
    ctx, postgresURL,
    pgdriver.WithTracerProvider(otel.GetTracerProvider()),
    pgdriver.WithMeterProvider(otel.GetMeterProvider()),
    pgdriver.WithMetricMeter(meter),
    pgdriver.WithLockTimeout(5*time.Second),
)
if err != nil { return err }
defer drv.Close()
// drv.DB() is *sql.DB used by Ent and lockr
```

<!-- archie:ai-end -->
