# pkg

<!-- archie:ai-start -->

> Shared infrastructure and domain-primitive leaf layer for the entire monorepo: pkg/framework supplies the universal plumbing (HTTP transport pipeline, Ent transaction propagation, advisory locks, RFC 7807 encoding), pkg/models the foundational typed errors/ValidationIssue/ServiceHookRegistry, and dozens of small focused utility packages (clock, datetime, timeutil, currencyx, filter, pagination, kafka). Its one hard constraint is that pkg/* must never import openmeter/* domain packages — it is the pure leaf every binary and domain depends on.

## Patterns

**pkg/framework as the universal infrastructure choke point** — All domain adapter/HTTP code routes through pkg/framework: entutils.TransactingRepo for ctx-bound Ent transactions, httptransport.Handler for the decode/operate/encode pipeline, lockr for pg_advisory locks, commonhttp.GenericErrorEncoder for RFC 7807 mapping. Never call Ent, net/http, or Postgres APIs directly from domain code. (`entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (*Entity, error) { ... })`)
**pkg/models is a zero-openmeter-import leaf** — pkg/models provides ValidationIssue, NamespacedID, ServiceHookRegistry[T], RFC 7807 StatusProblem, typed Generic* error sentinels, CadencedModel. It must import nothing from openmeter/* — any such import creates an unresolvable cycle across the whole type system. (`return nil, models.NewGenericNotFoundError(models.NamespacedID{Namespace: ns, ID: id})`)
**Typed Generic* error sentinels for HTTP status mapping** — Service/adapter methods return models.GenericNotFoundError/GenericConflictError/GenericValidationError etc. — never raw fmt.Errorf — so commonhttp.GenericErrorEncoder maps them to the correct HTTP status via type matching; unmatched errors fall through to 500. (`return nil, models.NewGenericConflictError(models.NamespacedID{Namespace: ns, ID: id})`)
**clock.Now() everywhere, never time.Now()** — All time reads in production code go through pkg/clock.Now() (and pkg/lrux TTL comparisons) so tests can FreezeTime/UnFreeze deterministically for billing-period arithmetic. (`import "github.com/openmeterio/openmeter/pkg/clock"; t := clock.Now()`)
**Calendar arithmetic via pkg/datetime / pkg/timeutil ISODuration** — Month/year duration math uses datetime.DateTime.Add(ISODuration) and timeutil.Recurrence, never time.Duration arithmetic — variable month lengths otherwise yield wrong billing boundaries; construct Recurrence via NewRecurrence (not struct literal) to keep Validate(). (`next := start.Add(datetime.DurationMonth)`)
**Dual-output filters and Page/Result pagination contract** — Query filters implement both Ent selector predicates and go-sqlbuilder WHERE expressions via pkg/filter.Filter with single-operator Validate(); all List methods accept pagination.Page and return Result[T] via MapResult/MapResultErr — never compute raw SQL OFFSET/LIMIT. (`return pagination.MapResult(page, items, func(i Item) APIItem { return toAPI(i) })`)
**Currency-safe monetary arithmetic via pkg/currencyx** — All monetary rounding and proportional splits go through currencyx.Calculator.RoundToPrecision and AllocateByWeight/AllocateByAmount (largest-remainder) — fixed-precision constants or hand-splitting drop the rounding remainder. (`calc.RoundToPrecision(amount)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `pkg/framework/entutils/transaction.go` | TransactingRepo / TransactingRepoWithNoValue — reads and rebinds the ctx-bound Ent transaction; central to all adapter DB access. | Never call creator.Tx() directly or store *entdb.Tx on adapter structs — wrap in TransactingRepo so helpers rebind to the caller's ctx transaction; missing wrapper produces silent partial writes. |
| `pkg/framework/transport/httptransport/handler.go` | Generic Handler[Request,Response] decode/operate/encode pipeline with chained ErrorEncoders used by every domain HTTP handler. | GenericErrorEncoder is always appended via defaultHandlerOptions; AppendOptions adds encoders — do not replace h.options wholesale; business logic belongs in the Operation, not middleware. |
| `pkg/framework/lockr/locker.go` | pg_advisory_xact_lock distributed lock requiring an active Postgres transaction in ctx before LockForTX. | Returns an error if no tx is in ctx; use SessionLocker (session.go) for connection-scoped locks and always Close() it; never wrap LockForTX in context.WithTimeout. |
| `pkg/models/validationissue.go` | Immutable ValidationIssue with copy-on-write With* builder carrying field paths, component, severity, and HTTP status attributes. | Never mutate a ValidationIssue field directly — all With* methods clone; direct mutation corrupts shared instances during multi-layer error propagation. |
| `pkg/models/servicehook.go` | Generic ServiceHookRegistry[T] for cross-domain lifecycle callbacks with re-entrancy prevention via a pointer-identity context key. | Loop-prevention uses fmt.Sprintf('%p', r) — do not copy the registry value; RegisterHooks is called at Wire time in app/common, never in domain constructors. |
| `pkg/timeutil/recurrence.go` | Anchor-based billing-period recurrence for subscription and metering cadences. | Construct via NewRecurrence (struct literal skips Validate()); use ISODuration for month/year intervals; Overlaps and OverlapsInclusive differ for touching sequential periods. |
| `pkg/kafka/config.go` | Typed Kafka config structs compiling to kafka.ConfigMap via AsConfigMap with fail-fast validation and localhost IPv4 auto-fix. | Never set ConfigMap keys directly — bypasses validation; *.ms keys must use TimeDurationMilliSeconds; metadata.max.age.ms must always be 3× TopicMetadataRefreshInterval. |
| `pkg/pagination/page.go` | Page/Result[T] contract for all domain List methods; Result.MarshalJSON flattens Page into the JSON root. | Use MapResult/MapResultErr — manual Result construction misses TotalCount and Page echo; CollectAll caps at 10,000 pages and returns nil items on any page error. |

## Anti-Patterns

- Importing any openmeter/* domain package from a pkg/ sub-package — creates an unresolvable circular dependency; pkg/ is a pure leaf depended on by all binaries and domains.
- Using time.Now() instead of clock.Now() in pkg/ or domain production code — breaks deterministic billing-period tests.
- Returning plain fmt.Errorf where models.Generic* typed errors are expected — GenericErrorEncoder falls through to 500 Internal Server Error.
- Using time.Duration arithmetic for month/year periods instead of pkg/datetime / pkg/timeutil ISODuration — variable month lengths produce wrong billing boundaries.
- Calling a.db.Foo()/raw SQL or storing *entdb.Tx on an adapter instead of wrapping helpers in TransactingRepo — falls off the ctx-bound transaction.

## Decisions

- **pkg/ has zero imports from openmeter/* domain packages.** — All seven binaries and their domain packages depend on pkg/; any reverse import would create an unresolvable cycle across the entire dependency graph.
- **entutils.TransactingRepo reads the transaction from ctx rather than accepting *entdb.Tx as a parameter, with savepoint-based nesting.** — Ent transactions propagate implicitly via ctx; parameter-passing would leak tx plumbing into every call site and make nested-transaction participation impossible.
- **pkg/models.ValidationIssue uses a private constructor and copy-on-write With* methods instead of a mutable builder.** — Immutability lets the same issue be safely annotated as it propagates through service → adapter → HTTP encoder without accidental shared-state mutation.

## Example: Domain adapter method using TransactingRepo with a typed error sentinel

```
import (
    entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
    "github.com/openmeterio/openmeter/pkg/framework/entutils"
    "github.com/openmeterio/openmeter/pkg/models"
)

func (a *adapter) GetByID(ctx context.Context, ns, id string) (*Entity, error) {
    return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (*Entity, error) {
        row, err := tx.db.Entity.Query().Where(entity.Namespace(ns), entity.ID(id)).Only(ctx)
        if entdb.IsNotFound(err) {
            return nil, models.NewGenericNotFoundError(models.NamespacedID{Namespace: ns, ID: id})
        }
        return toDomain(row), err
    })
}
```

<!-- archie:ai-end -->
