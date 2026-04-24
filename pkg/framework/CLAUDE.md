# framework

<!-- archie:ai-start -->

> Shared low-level infrastructure layer providing the generic HTTP transport pipeline (httptransport), Ent transaction management (entutils), distributed locks (lockr), RFC 7807 error encoding (commonhttp), OTel instrumentation helpers (clickhouseotel, tracex), and pgx pool wiring (pgdriver). All domain packages under openmeter/ depend on this layer; it must not import them.

## Patterns

**httptransport decode/operate/encode pipeline** — Every HTTP endpoint uses httptransport.NewHandler(operation, decoder, responseEncoder, errorEncoder, opts...) — never implement ServeHTTP directly in a handler struct. (`httptransport.NewHandler(op, DecodeRequest, EncodeResponse, EncodeError, httptransport.WithOperationName("op"))`)
**TransactingRepo for all Ent adapter DB access** — Every adapter method wraps its body in entutils.TransactingRepo(ctx, client, func(tx *entdb.Tx) (*T, error){...}) so it rebinds to any transaction already carried in ctx. (`return entutils.TransactingRepo(ctx, a.client, func(tx *entdb.Tx) (*Entity, error) { return tx.Entity.Create()...; })`)
**operation.Operation function type with Middleware** — Business logic is expressed as operation.Operation[Req, Resp] (a plain func) and composed via operation.Chain/Middleware. Never put business logic in middleware. (`var op operation.Operation[Request, Response] = func(ctx context.Context, req Request) (Response, error) { ... }`)
**GenericErrorEncoder chain** — Error encoders return bool; false passes to the next encoder. Add domain errors as models.Generic* sentinel types mapped in commonhttp.GenericErrorEncoder; never write status codes directly inside handler code. (`func EncodeError(ctx context.Context, err error, w http.ResponseWriter) bool { return commonhttp.HandleErrorIfTypeMatches[*MyError](ctx, http.StatusBadRequest, err, w) }`)
**Locker requires active Postgres transaction in ctx** — lockr.Locker.LockForTX(ctx, key) calls pg_advisory_xact_lock inside the Ent tx already in ctx. Always call inside entutils.TransactingRepo; never call from outside a transaction. (`entutils.TransactingRepo(ctx, client, func(tx *entdb.Tx) (*T, error) { if err := locker.LockForTX(ctx, key); err != nil { return nil, err }; ... })`)
**tracex.Start + Wrap for span lifecycle** — Use tracex.Start/Wrap instead of tracer.Start directly; Wrap records errors, sets span status, and handles panics automatically. (`return tracex.Start(ctx, tracer, "svc.Op", func(s *tracex.Span[*Entity]) (*Entity, error) { return s.Wrap(doWork(s.Ctx())) })`)
**Functional options (Option interface) for configuration** — Packages like pgdriver and clickhouseotel expose Option interfaces for configuration; call Validate() before first use to surface nil Conn/Meter/Tracer early. (`driver, err := pgdriver.New(pool, pgdriver.WithLockTimeout(5*time.Second), pgdriver.WithMetrics(meter))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `pkg/framework/entutils/transaction.go` | TransactingRepo/TxDriver implementation; ctx-propagated transaction reuse with savepoints for nesting | Never call creator.Tx() directly; always use TransactingRepo. Never store *entdb.Tx on adapter struct. |
| `pkg/framework/entutils/mixins.go` | Ent mixin definitions (IDMixin ULID char(26), NamespaceMixin, TimeMixin, MetadataMixin, AnnotationsMixin) | All entities must compose these mixins; never add id/namespace/created_at fields manually on schema. |
| `pkg/framework/commonhttp/errors.go` | RFC 7807 problem-detail error chain: GenericErrorEncoder, HandleIssueIfHTTPStatusKnown, ValidationIssue HTTP status attribute | Always use WithHTTPStatusCodeAttribute on ValidationIssue; without it HandleIssueIfHTTPStatusKnown returns false and falls through. |
| `pkg/framework/lockr/locker.go` | Transaction-scoped advisory lock via pg_advisory_xact_lock; key hashed with xxhash | Must be called inside an active Ent transaction; calling outside returns error. Don't use context.WithTimeout — use pgdriver.WithLockTimeout instead. |
| `pkg/framework/lockr/session.go` | Connection-scoped advisory lock (pg_advisory_lock/unlock) needing a dedicated *sql.Conn | Not goroutine-safe under high contention; always call Close() to release the dedicated connection. |
| `pkg/framework/transport/httptransport/handler.go` | Generic Handler[Request,Response] implementing http.Handler; wraps operation with decode/encode | Never implement ServeHTTP directly in domain httpdriver — always construct via httptransport.NewHandler. |
| `pkg/framework/pgdriver/driver.go` | Constructs project-wide *sql.DB from pgxpool with OTel tracing, metrics, and optional lock_timeout | Don't set MaxIdleConns > 0 on the returned *sql.DB — pgx pool manages idle connections; dual management causes leaks. |
| `pkg/framework/tracex/tracex.go` | Generic OTel span wrapper with error recording, status, and panic recovery | Always pass s.Ctx() through inside Wrap callbacks; never introduce context.Background() there. |

## Anti-Patterns

- Importing openmeter/* domain packages from any pkg/framework sub-package — creates import cycles; this layer must be a leaf dependency
- Using context.Background() or context.TODO() inside adapter callbacks or Wrap functions — always propagate the caller ctx
- Calling creator.Tx() or storing *entdb.Tx on adapter structs instead of using TransactingRepo — falls off the ctx-bound transaction
- Adding ORDER BY clauses on Ent queries before calling .Cursor() — entcursor appends its own order producing undefined combined ordering
- Implementing business logic inside operation.Middleware — middleware is for cross-cutting concerns (auth, logging, tracing) only

## Decisions

- **TransactingRepo reads transaction from ctx rather than accepting *entdb.Tx as a parameter** — Ent transactions propagate implicitly via ctx; explicit *entdb.Tx parameters leak tx plumbing into every call site and cannot enforce nesting via savepoints.
- **httptransport.Handler[Request,Response] generic pipeline separates decode/operate/encode** — Enforces consistent request validation, error encoding, and OTel tracing across all ~60+ endpoints without duplicating boilerplate in each handler.
- **Two advisory lock types (Locker for tx-scoped, SessionLocker for connection-scoped)** — Different lifetime requirements: billing operations need lock released on tx commit/rollback (Locker), while some admin flows need lock outlive individual transactions (SessionLocker).

## Example: New HTTP endpoint: full decode/operate/encode handler construction

```
package httpdriver

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/pkg/framework/operation"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
)

type ListFoosRequest struct{ Namespace string }
type ListFoosResponse struct{ Items []Foo }

func NewListFoosHandler(svc Service, errHandler httptransport.ErrorHandler) http.Handler {
// ...
```

<!-- archie:ai-end -->
