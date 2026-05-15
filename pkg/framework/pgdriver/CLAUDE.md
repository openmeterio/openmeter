# pgdriver

<!-- archie:ai-start -->

> Constructs the project-wide *sql.DB from a pgxpool.Pool with OTel tracing (XSAM/otelsql), optional metric.Meter pool observation (pgxpoolobserver), and optional lock_timeout runtime parameter for lockr; exposes Driver.DB() for Ent and lockr consumers.

## Patterns

**Option interface for functional configuration** — All optional settings (TracerProvider, MeterProvider, MetricMeter, SpanOptions, LockTimeout) are passed as Option values; each implements apply(*options). Callers compose options without using a config struct. (`pgdriver.NewPostgresDriver(ctx, url, pgdriver.WithLockTimeout(3*time.Second), pgdriver.WithTracerProvider(tp))`)
**MaxIdleConns must be 0 — pgx pool manages idle connections** — otelsql.OpenDB wraps pgxstdlib.GetPoolConnector(pool). db.SetMaxIdleConns(0) is mandatory because pgx manages idle connections in its pool; a non-zero Go idle pool duplicates connection management and wastes resources. (`db.SetMaxIdleConns(0) // set after otelsql.OpenDB`)
**WithLockTimeout sets lock_timeout via RuntimeParams** — WithLockTimeout sets lock_timeout in pgxpool ConnConfig.ConnConfig.RuntimeParams as a millisecond string. This enables PostgreSQL-side advisory lock timeouts used by lockr without context cancellation destroying the connection. (`o.connConfig.ConnConfig.RuntimeParams["lock_timeout"] = fmt.Sprintf("%d", timeout.Milliseconds())`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `driver.go` | Single file containing NewPostgresDriver, Driver struct, DB() and Close() methods. | Close() calls pool.Close() but does NOT call db.Close() — callers that hold *sql.DB references must not call db.Close() independently. Pool shutdown is immediate; ensure in-flight queries complete first. |

## Anti-Patterns

- Setting MaxIdleConns > 0 on the returned *sql.DB — pgx pool manages idle connections; a non-zero Go idle pool causes duplicate connection management and resource waste
- Calling Driver.Close() before ensuring all in-flight queries have completed — pool.Close() is immediate and does not drain
- Using context.WithTimeout for lock acquisition on connections from this pool — use pgdriver.WithLockTimeout instead to avoid pgx connection cancellation on ctx deadline

## Decisions

- **Wrap pgxpool behind database/sql via pgxstdlib** — Ent ORM and lockr both require database/sql; pgxpool gives fine-grained connection pool control (RuntimeParams, pool metrics) while pgxstdlib bridges the two APIs without reimplementing the driver.
- **lock_timeout in RuntimeParams rather than per-query SET** — Setting lock_timeout at connection config level applies it to all advisory lock acquisitions on that connection without requiring each caller to issue a SET command, making lockr usage transparent.

## Example: Create a Driver with OTel tracing, pool metrics, and lock timeout for use with Ent and lockr

```
import "github.com/openmeterio/openmeter/pkg/framework/pgdriver"

drv, err := pgdriver.NewPostgresDriver(
    ctx,
    postgresURL,
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
