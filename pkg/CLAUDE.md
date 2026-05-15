# pkg

<!-- archie:ai-start -->

> Shared infrastructure and utility leaf layer for the entire monorepo — provides generic HTTP transport (pkg/framework/httptransport), Ent transaction management (pkg/framework/entutils), domain primitives (pkg/models), time/billing utilities (pkg/timeutil, pkg/datetime), Kafka plumbing (pkg/kafka), pagination (pkg/pagination), filters (pkg/filter), and dozens of small focused packages. Primary constraint: pkg/ must never import openmeter/* domain packages — it is a pure leaf dependency used by all seven binaries and their domains.

## Patterns

**pkg/framework as universal infrastructure layer** — All domain adapter code depends on pkg/framework/entutils for TransactingRepo, pkg/framework/httptransport for the HTTP handler pipeline, pkg/framework/lockr for advisory locks, and pkg/framework/commonhttp for RFC 7807 error encoding. Never bypass these abstractions by calling Ent, net/http, or Postgres APIs directly from domain code. (`entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (*Entity, error) { ... })`)
**pkg/models as the foundational domain primitive — zero openmeter/* imports** — pkg/models provides ValidationIssue, NamespacedID, ServiceHookRegistry, RFC 7807 Problem, typed error sentinels, and CadencedModel. It must import nothing from openmeter/* — violations create circular deps that break the entire type system. (`return nil, models.NewGenericNotFoundError(models.NamespacedID{Namespace: ns, ID: id})`)
**clock.Now() everywhere in production code** — All time reads in production code must use pkg/clock.Now() not time.Now(). Tests use clock.FreezeTime / clock.UnFreeze for deterministic billing period calculations. Forgetting defer clock.UnFreeze() leaks frozen state across test cases. (`import "github.com/openmeterio/openmeter/pkg/clock"; t := clock.Now()`)
**Typed errors via models.Generic* sentinels** — Domain service and adapter methods must return typed errors (models.GenericNotFoundError, models.GenericConflictError, models.GenericValidationError, etc.) not raw fmt.Errorf — the GenericErrorEncoder maps these to correct HTTP status codes via type matching. (`return nil, models.NewGenericConflictError(models.NamespacedID{Namespace: ns, ID: id})`)
**pkg/datetime for all calendar arithmetic** — Month/year duration arithmetic must use pkg/datetime.DateTime.Add(ISODuration) not time.Duration arithmetic — month lengths vary and naive Duration math produces wrong billing period boundaries. (`next := start.Add(datetime.DurationMonth)`)
**pkg/filter dual-output filter types** — Query filters must implement both Ent selector predicates and go-sqlbuilder WHERE expressions via the pkg/filter.Filter interface. Single-operator validation via Validate() must be called before applying filters to queries. (`filter.FilterString{In: []string{"a", "b"}}.Select(predicate)`)
**pkg/pagination Page/Result contract for all list operations** — All domain List methods must accept pagination.Page and return pagination.Result[T]. Never compute SQL OFFSET or LIMIT directly; use Page.Offset() and Page.Limit(). Construct results via MapResult/MapResultErr to avoid missed TotalCount assignments. (`return pagination.MapResult(page, items, func(i Item) APIItem { return toAPI(i) })`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `pkg/framework/entutils/transaction.go` | TransactingRepo / TransactingRepoWithNoValue — reads and rebinds the ctx-bound Ent transaction; central to all adapter DB access | Never call creator.Tx() directly or store *entdb.Tx on adapter structs — use TransactingRepo so helpers rebind to the caller's ctx transaction |
| `pkg/framework/transport/httptransport/handler.go` | Generic Handler[Request,Response] decode/operate/encode pipeline with chained ErrorEncoders — every domain HTTP handler uses this | Chain adds middleware; AppendOptions appends error encoders — do not replace h.options entirely; GenericErrorEncoder is always appended via defaultHandlerOptions |
| `pkg/framework/lockr/locker.go` | pg_advisory_xact_lock distributed lock — requires active Postgres transaction in ctx before LockForTX is called | Locker returns an error if no tx is in ctx; use SessionLocker (session.go) for connection-scoped locks that outlive transactions |
| `pkg/models/validationissue.go` | Immutable ValidationIssue value type with With* copy-on-write builder; carries field paths, component names, severity, and HTTP status attributes | Never mutate a ValidationIssue struct field directly — always use With* methods which clone; direct mutation corrupts shared instances in multi-layer error propagation |
| `pkg/models/servicehook.go` | Generic ServiceHookRegistry[T] for cross-domain lifecycle callbacks with re-entrancy prevention via pointer-identity context key | Loop-prevention uses fmt.Sprintf('%p', r) — do not copy the registry value; RegisterHooks is called at Wire time in app/common, not in domain constructors |
| `pkg/timeutil/recurrence.go` | Anchor-based billing period recurrence (Recurrence, NewRecurrence, NewRecurrenceFromISODuration) for subscription and metering cadences | Always construct via NewRecurrence — struct literal skips Validate(); use ISODuration for month/year intervals not time.Duration; Overlaps and OverlapsInclusive differ for touching periods |
| `pkg/kafka/config.go` | Typed Kafka config structs producing kafka.ConfigMap with sequential fail-fast validation and localhost IPv4 auto-fix | Never set kafka.ConfigMap keys directly — bypasses validation; metadata.max.age.ms must always be 3× TopicMetadataRefreshInterval |
| `pkg/pagination/page.go` | Page/Result[T] contract for all domain List methods; Result.MarshalJSON flattens Page fields into the JSON root | Use MapResult/MapResultErr — manual Result construction misses TotalCount and Page echo; CollectAll caps at 10,000 pages and returns nil on any page error |

## Anti-Patterns

- Importing openmeter/* domain packages from any pkg/ sub-package — creates circular dependencies; pkg/ is a pure leaf that all domains depend on
- Using time.Now() instead of clock.Now() in any pkg/ or openmeter/ production code — breaks test determinism for billing period calculations
- Returning plain fmt.Errorf from service/adapter code where models.Generic* typed errors are expected — the GenericErrorEncoder falls through to 500 Internal Server Error
- Using time.Duration arithmetic for month/year periods instead of pkg/datetime ISODuration — month lengths vary and produce wrong billing period boundaries
- Calling entutils.TransactingRepo with a raw *entdb.Client in a helper invoked inside a transaction — must still wrap so the ctx-bound transaction is honored

## Decisions

- **pkg/ has zero imports from openmeter/* domain packages** — Prevents circular dependency; all seven domain binaries and their test suites depend on pkg/ — any openmeter/* import would create an unresolvable cycle
- **pkg/framework/entutils.TransactingRepo reads transaction from ctx rather than accepting *entdb.Tx as a parameter** — Ent transactions propagate implicitly via ctx; parameter-passing would leak tx plumbing into every adapter call site and make nested transaction participation impossible
- **pkg/models.ValidationIssue uses a private constructor and With* copy-on-write pattern rather than a mutable builder** — Immutability prevents accidental mutation as issues propagate through service → adapter → HTTP encoder chain; multiple service layers can annotate the same issue safely

## Example: Domain adapter method using TransactingRepo with typed error sentinel

```
import (
    entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
    "github.com/openmeterio/openmeter/pkg/framework/entutils"
    "github.com/openmeterio/openmeter/pkg/models"
)

func (a *adapter) GetByID(ctx context.Context, ns, id string) (*Entity, error) {
    return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (*Entity, error) {
        row, err := tx.db.Entity.Query().
            Where(entity.Namespace(ns), entity.ID(id)).
            Only(ctx)
        if entdb.IsNotFound(err) {
            return nil, models.NewGenericNotFoundError(models.NamespacedID{Namespace: ns, ID: id})
        }
        return toDomain(row), err
// ...
```

<!-- archie:ai-end -->
