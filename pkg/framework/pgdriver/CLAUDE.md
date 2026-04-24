# pgdriver

<!-- archie:ai-start -->

> Constructs the project-wide *sql.DB from a pgxpool.Pool with OTel tracing (XSAM/otelsql), optional metric.Meter pool observation (pgxpoolobserver), and optional lock_timeout runtime parameter for lockr; exposes Driver.DB() for Ent and lockr consumers.

## Patterns

**Option interface for functional configuration** — All optional settings (TracerProvider, MeterProvider, MetricMeter, SpanOptions, LockTimeout) are passed as Option values; each implements apply(*options). (`pgdriver.NewPostgresDriver(ctx, url, pgdriver.WithLockTimeout(3*time.Second), pgdriver.WithTracerProvider(tp))`)
**Pool-backed sql.DB with MaxIdleConns=0** — otelsql.OpenDB wraps pgxstdlib.GetPoolConnector(pool). db.SetMaxIdleConns(0) is mandatory because pgx manages idle connections in the pool itself. (`db.SetMaxIdleConns(0)`)
**lock_timeout via RuntimeParams** — WithLockTimeout sets lock_timeout in pgxpool ConnConfig.ConnConfig.RuntimeParams as a millisecond string, enabling PostgreSQL-side advisory lock timeouts used by lockr. (`o.connConfig.ConnConfig.RuntimeParams["lock_timeout"] = fmt.Sprintf("%d", timeout.Milliseconds())`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `driver.go` | Single file; NewPostgresDriver, Driver struct, DB() and Close() methods. | Close() calls pool.Close() but does NOT close db — callers that hold *sql.DB must not call db.Close() separately. |

## Anti-Patterns

- Setting MaxIdleConns > 0 on the returned *sql.DB — pgx pool manages idle connections; a non-zero Go idle pool causes duplicate connection management
- Calling Driver.Close() without ensuring all in-flight queries have completed — pool.Close() is immediate

## Decisions

- **Wrap pgxpool behind database/sql via pgxstdlib** — Ent ORM and lockr both require database/sql; pgxpool gives connection-pool control while pgxstdlib bridges the two APIs.

<!-- archie:ai-end -->
