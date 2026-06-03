# framework

<!-- archie:ai-start -->

> Shared low-level infrastructure layer providing the generic HTTP transport pipeline (transport/httptransport), Ent transaction management and schema mixins (entutils), DB-agnostic transaction propagation (transaction), distributed advisory locks (lockr), RFC 7807 error/encoding primitives (commonhttp), the Operation/Middleware contract (operation), pgx pool wiring (pgdriver), and OTel helpers (tracex, clickhouseotel). Every domain package under openmeter/ depends on this layer; its primary constraint is that it must remain a domain-free leaf and never import openmeter/* domain packages.

## Patterns

**httptransport decode/operate/encode pipeline** — Every HTTP endpoint is built via httptransport.NewHandler with a RequestDecoder, operation.Operation, ResponseEncoder, and ErrorEncoder. Domain handlers never implement ServeHTTP directly. (`httptransport.NewHandler(op, DecodeRequest, EncodeResponse, EncodeError, httptransport.WithOperationName("op"))`)
**TransactingRepo on every adapter method body** — Domain adapter methods wrap DB access in entutils.TransactingRepo(ctx, a, func(ctx, tx *adapter)(*T,error){...}) (or TransactingRepoWithNoValue) so they rebind to any Ent transaction already in ctx, falling back to Self() when none is present. (`return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (*Entity, error) { return toDomain(tx.db.Entity.Create().Save(ctx)) })`)
**TxCreator + TxUser[T] triad on every adapter** — Each adapter implements Tx(ctx) via HijackTx+NewTxDriver, WithTx(ctx, tx) via entdb.NewTxClientFromRawConfig, and Self() — all three are required for TransactingRepo to rebind correctly. (`func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter { return &adapter{db: entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()).Client()} }`)
**GenericErrorEncoder chain (bool-returning, first-match wins)** — Error encoders return bool; false passes to the next encoder. Domain errors must be models.Generic* sentinels matched by commonhttp.HandleErrorIfTypeMatches; HTTP status codes are never written directly in handler logic. (`func EncodeError(ctx context.Context, err error, w http.ResponseWriter) bool { return commonhttp.HandleErrorIfTypeMatches[models.GenericNotFoundError](ctx, http.StatusNotFound, err, w) }`)
**LockForTX / transaction.Run inside an active transaction** — lockr.Locker.LockForTX(ctx, key) must run inside an entutils.TransactingRepo / transaction.Run block; it calls pg_advisory_xact_lock in the ctx-bound tx and errors if no Postgres transaction is present. Use pgdriver.WithLockTimeout, not context.WithTimeout, for lock timeouts. (`return transaction.Run(ctx, creator, func(ctx context.Context) (*T, error) { if err := locker.LockForTX(ctx, key); err != nil { return nil, err }; ... })`)
**tracex.Start + Wrap for OTel span lifecycle** — Use tracex.Start/StartWithNoValue + Wrap instead of tracer.Start directly; Wrap records errors, sets span status, recovers panics, and ends the span. Always pass s.Ctx() (the child-span ctx) into operations inside Wrap. (`return tracex.Start(ctx, tracer, "svc.Op", func(s *tracex.Span[*Entity]) (*Entity, error) { return s.Wrap(adapter.Write(s.Ctx(), id)) }).Result()`)
**Standard mixin composition on Ent entity schemas** — Every openmeter/ent/schema entity composes entutils.IDMixin{} (ULID char(26)), NamespaceMixin{}, TimeMixin{} in Mixin(). Omitting NamespaceMixin breaks multi-tenancy; omitting TimeMixin breaks soft-delete. (`func (Entity) Mixin() []ent.Mixin { return []ent.Mixin{entutils.IDMixin{}, entutils.NamespaceMixin{}, entutils.TimeMixin{}} }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `entutils/transaction.go` | TransactingRepo/TransactingRepoWithNoValue and TxDriver — ctx-propagated Ent transaction reuse with Postgres savepoints for nesting. Core of all adapter DB access. | Never call creator.Tx() directly or store *entdb.Tx on an adapter struct; always go through TransactingRepo. A missing wrapper silently degrades to Self() (non-tx) and causes partial writes. |
| `entutils/mixins.go` | Ent mixin definitions: IDMixin (ULID char(26)), NamespaceMixin, TimeMixin, ResourceMixin and friends used by all ~30 entity schemas. | Never add id/namespace/created_at/updated_at fields manually on a schema — always compose the mixins. |
| `commonhttp/errors.go` | RFC 7807 problem-detail chain: GenericErrorEncoder, HandleErrorIfTypeMatches, HandleIssueIfHTTPStatusKnown, and ValidationIssue HTTP-status attribute helpers. | Call WithHTTPStatusCodeAttribute on a ValidationIssue before returning it; without it HandleIssueIfHTTPStatusKnown returns false and the error falls through to 500. |
| `lockr/locker.go` | Transaction-scoped advisory lock (pg_advisory_xact_lock) with key hashed to uint64 via xxhash; auto-released on tx commit/rollback. | Must be inside an active Ent transaction in ctx. Use pgdriver.WithLockTimeout instead of context.WithTimeout — ctx cancel kills the pgx connection. |
| `lockr/session.go` | Connection-scoped advisory lock (pg_advisory_lock) over a dedicated *sql.Conn for admin flows where the lock must outlive individual transactions. | Not goroutine-safe under high contention; always call Close() to release the dedicated connection or it leaks from the pool. |
| `transport/httptransport/handler.go` | Generic Handler[Request,Response] implementing http.Handler — the decode→operation→encode pipeline used by every v1 and v3 endpoint; holds the last-resort 500 fallback. | Never implement ServeHTTP directly in a domain httpdriver package — always construct via httptransport.NewHandler. Handler is value-receiver with copy-on-With/Chain semantics. |
| `transaction/transaction.go` | DB-agnostic Run[R]/RunWithNoValue helpers and the Driver (Commit/Rollback/SavePoint) propagated through context for start-or-reuse transaction semantics with nested savepoints. | Never call SetDriverOnContext manually outside Run; DriverConflictError signals tx reuse and must not be treated as fatal. Never use context.Background() in a Run callback — it loses the Driver. |
| `tracex/tracex.go` | Generic OTel Span[T]/SpanNoValue wrapper with automatic error recording, span status, and panic recovery (panic re-raised after End). | Always use s.Ctx() inside Wrap callbacks; never call span.End() manually after Wrap — it double-ends and truncates the span. |

## Anti-Patterns

- Importing any openmeter/* domain package from a pkg/framework sub-package — creates import cycles across every adapter; this layer must stay a domain-free leaf.
- Using context.Background()/context.TODO() inside TransactingRepo callbacks, transaction.Run callbacks, or tracex.Wrap functions — drops the ctx-bound transaction, OTel span, and cancellation.
- Calling a.db.Foo()/raw SQL or creator.Tx() directly, or storing *entdb.Tx on an adapter struct, instead of TransactingRepo — falls off the ctx-bound transaction and produces partial writes under concurrency.
- Adding ORDER BY before .Cursor() or applying .Limit()/.Offset() before .Paginate() — the entcursor/entpaginate templates append their own ordering and reset limit/offset, producing undefined or discarded results.
- Putting business logic inside operation.Middleware — middleware is for cross-cutting concerns (auth, logging, tracing) only; business logic belongs in the Operation function.

## Decisions

- **TransactingRepo reads the transaction from ctx rather than accepting *entdb.Tx as a parameter, and TxDriver uses Postgres savepoints for nesting.** — Ent transactions propagate implicitly via ctx; explicit *entdb.Tx parameters leak tx plumbing into every call site and cannot enforce nesting via savepoints.
- **A single generic httptransport.Handler[Request,Response] separates decode/operate/encode for all endpoints.** — Enforces consistent request validation, RFC 7807 error encoding, and OTel tracing across 60+ endpoints without duplicating boilerplate per domain handler.
- **Two advisory lock types — Locker (tx-scoped) and SessionLocker (connection-scoped).** — Different lifetime requirements: billing operations need the lock released on tx commit/rollback (Locker), while some admin flows need locks that outlive individual transactions (SessionLocker).

## Example: Adapter method participating in a ctx-bound transaction with an advisory lock

```
import (
	"context"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
)

func (a *adapter) Update(ctx context.Context, in UpdateInput, key lockr.Key) (*Entity, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (*Entity, error) {
		if err := a.locker.LockForTX(ctx, key); err != nil {
			return nil, err
		}
		return toDomain(tx.db.Entity.UpdateOneID(in.ID).Save(ctx))
	})
}
```

<!-- archie:ai-end -->
