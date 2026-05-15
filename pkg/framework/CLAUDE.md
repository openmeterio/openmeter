# framework

<!-- archie:ai-start -->

> Shared low-level infrastructure layer providing the generic HTTP transport pipeline (httptransport), Ent transaction management (entutils), distributed locks (lockr), RFC 7807 error encoding (commonhttp), OTel instrumentation helpers (clickhouseotel, tracex), and pgx pool wiring (pgdriver). All domain packages under openmeter/ depend on this layer; it must never import them.

## Patterns

**httptransport decode/operate/encode pipeline** — Every HTTP endpoint uses httptransport.NewHandler with a RequestDecoder, operation.Operation, ResponseEncoder, and ErrorEncoder. Never implement ServeHTTP directly in a domain handler struct. (`httptransport.NewHandler(op, DecodeRequest, EncodeResponse, EncodeError, httptransport.WithOperationName("op"))`)
**TransactingRepo on every adapter method body** — Every domain adapter method wraps its DB access in entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (*T, error){...}) so it rebinds to any Ent transaction already carried in ctx. TransactingRepoWithNoValue for error-only operations. (`return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (*Entity, error) { return toDomain(tx.db.Entity.Create().Save(ctx)) })`)
**TxCreator + TxUser[T] triad on every adapter** — Each adapter must implement Tx(ctx) via HijackTx+NewTxDriver, WithTx(ctx, tx) via NewTxClientFromRawConfig, and Self() — all three required for TransactingRepo to work correctly. (`func (a *adapter) Self() *adapter { return a }
func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter { return &adapter{db: entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()).Client()} }`)
**GenericErrorEncoder chain (bool-returning, first-match wins)** — Error encoders return bool; false passes to the next encoder. Domain errors must be models.Generic* sentinels matched by commonhttp.HandleErrorIfTypeMatches; never write HTTP status codes directly in handler logic. (`func EncodeError(ctx context.Context, err error, w http.ResponseWriter) bool { return commonhttp.HandleErrorIfTypeMatches[models.GenericNotFoundError](ctx, http.StatusNotFound, err, w) }`)
**LockForTX inside active Ent transaction** — lockr.Locker.LockForTX(ctx, key) must be called inside an entutils.TransactingRepo block; it calls pg_advisory_xact_lock inside the tx in ctx and errors if no transaction is present. (`return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (*T, error) { if err := locker.LockForTX(ctx, key); err != nil { return nil, err }; ... })`)
**tracex.Start + Wrap for OTel span lifecycle** — Use tracex.Start/Wrap instead of tracer.Start directly; Wrap records errors, sets span status, and recovers panics. Always pass s.Ctx() (not outer ctx) into operations inside Wrap. (`return tracex.Start(ctx, tracer, "svc.Op", func(s *tracex.Span[*Entity]) (*Entity, error) { return s.Wrap(adapter.Write(s.Ctx(), id)) }).Val, tracex.Start(...).Err()`)
**Standard mixin composition on all Ent entity schemas** — Every openmeter/ent/schema/*.go entity must compose entutils.IDMixin{}, entutils.NamespaceMixin{}, entutils.TimeMixin{} in Mixin(). Omitting NamespaceMixin breaks multi-tenancy queries; omitting TimeMixin breaks soft-delete. (`func (Entity) Mixin() []ent.Mixin { return []ent.Mixin{entutils.IDMixin{}, entutils.NamespaceMixin{}, entutils.TimeMixin{}} }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `pkg/framework/entutils/transaction.go` | TransactingRepo/TxDriver implementation; ctx-propagated transaction reuse with savepoints for nesting. Core of all adapter DB access. | Never call creator.Tx() directly; always use TransactingRepo/TransactingRepoWithNoValue. Never store *entdb.Tx on adapter struct. |
| `pkg/framework/entutils/mixins.go` | Ent mixin definitions — IDMixin (ULID char(26)), NamespaceMixin, TimeMixin, MetadataMixin, AnnotationsMixin. Used by all ~30 entity schemas. | Never add id/namespace/created_at/updated_at fields manually on a schema — always compose these mixins. |
| `pkg/framework/commonhttp/errors.go` | RFC 7807 problem-detail error chain: GenericErrorEncoder, HandleIssueIfHTTPStatusKnown, ValidationIssue HTTP status attribute helpers. | Always call WithHTTPStatusCodeAttribute on ValidationIssue before returning; without it HandleIssueIfHTTPStatusKnown returns false and falls through to 500. |
| `pkg/framework/lockr/locker.go` | Transaction-scoped advisory lock via pg_advisory_xact_lock; key hashed to uint64 via xxhash. Auto-released on tx commit/rollback. | Must be inside an active Ent transaction in ctx. Use pgdriver.WithLockTimeout instead of context.WithTimeout — ctx cancel kills the pgx connection. |
| `pkg/framework/lockr/session.go` | Connection-scoped advisory lock (pg_advisory_lock) using a dedicated *sql.Conn. For admin flows where lock must outlive individual transactions. | Not goroutine-safe under high contention. Always call Close() to release the dedicated connection; failure leaks from the pool. |
| `pkg/framework/transport/httptransport/handler.go` | Generic Handler[Request,Response] implementing http.Handler; the decode→operation→encode pipeline used by every v1 and v3 endpoint. | Never implement ServeHTTP directly in domain httpdriver packages — always construct via httptransport.NewHandler. |
| `pkg/framework/tracex/tracex.go` | Generic OTel Span[T] wrapper with automatic error recording, span status, and panic recovery. | Always use s.Ctx() inside Wrap callbacks — never introduce context.Background() there. Don't call span.End() manually after Wrap; it double-ends the span. |
| `pkg/framework/pgdriver/driver.go` | Constructs the project-wide *sql.DB from pgxpool with OTel tracing and optional lock_timeout. Produces the DB consumed by Ent and lockr. | Don't set MaxIdleConns > 0 on the returned *sql.DB — pgx pool manages idle connections; dual management causes leaks. |

## Anti-Patterns

- Importing openmeter/* domain packages from any pkg/framework sub-package — creates import cycles; this layer must be a leaf dependency with no domain imports.
- Using context.Background() or context.TODO() inside adapter TransactingRepo callbacks or tracex.Wrap functions — always propagate the caller ctx.
- Calling creator.Tx() directly or storing *entdb.Tx on adapter structs instead of using TransactingRepo — falls off the ctx-bound transaction and produces partial writes under concurrency.
- Adding ORDER BY clauses on Ent queries before calling .Cursor() — entcursor appends (created_at ASC, id ASC) and the combined ordering becomes undefined.
- Implementing business logic inside operation.Middleware — middleware is for cross-cutting concerns (auth, logging, tracing) only; business logic belongs in the Operation function.

## Decisions

- **TransactingRepo reads transaction from ctx rather than accepting *entdb.Tx as a parameter** — Ent transactions propagate implicitly via ctx; explicit *entdb.Tx parameters leak tx plumbing into every call site and cannot enforce nesting via savepoints.
- **httptransport.Handler[Request,Response] generic pipeline separates decode/operate/encode** — Enforces consistent request validation, error encoding, and OTel tracing across all ~60+ endpoints without duplicating boilerplate in each domain handler.
- **Two advisory lock types: Locker (tx-scoped) and SessionLocker (connection-scoped)** — Different lifetime requirements — billing operations need lock released on tx commit/rollback (Locker); some admin flows need locks that outlive individual transactions (SessionLocker).

## Example: New HTTP endpoint: full decode/operate/encode handler construction using httptransport pipeline

```
package httpdriver

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type ListFoosRequest struct{ Namespace string }
type ListFoosResponse struct{ Items []Foo }

func NewListFoosHandler(svc Service, errHandler httptransport.ErrorHandler) http.Handler {
// ...
```

<!-- archie:ai-end -->
