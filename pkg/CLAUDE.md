# pkg

<!-- archie:ai-start -->

> Shared infrastructure and utility layer for the entire monorepo: provides generic HTTP transport (pkg/framework/httptransport), Ent transaction management (pkg/framework/entutils), domain primitives (pkg/models), time/billing utilities (pkg/timeutil, pkg/datetime), Kafka plumbing (pkg/kafka), pagination (pkg/pagination), filters (pkg/filter), and dozens of small focused packages. Primary constraint: pkg/ must not import openmeter/* domain packages — it is a leaf dependency used by all domains.

## Patterns

**pkg/framework as the universal infrastructure layer** — All domain adapter code depends on pkg/framework/entutils for TransactingRepo, pkg/framework/httptransport for HTTP handler pipeline, pkg/framework/lockr for advisory locks, and pkg/framework/commonhttp for RFC 7807 error encoding. Never bypass these abstractions. (`entutils.TransactingRepo, httptransport.NewHandler[Req,Resp], lockr.Locker`)
**pkg/models as the foundational domain primitive — zero openmeter/* imports** — pkg/models provides ValidationIssue, NamespacedID, ServiceHookRegistry, RFC 7807 Problem, etc. It must import nothing from openmeter/* — violations create circular deps that break the entire type system. (`models.GenericNotFoundError, models.ValidationIssue, models.NewStatusProblem`)
**clock.Now() everywhere in production code** — All time reads in production code must use pkg/clock.Now() not time.Now(). Tests use clock.FreezeTime / clock.UnFreeze for deterministic billing period calculations. (`import "github.com/openmeterio/openmeter/pkg/clock"; t := clock.Now()`)
**Typed errors via models.Generic* sentinels** — Domain service and adapter methods must return typed errors (models.GenericNotFoundError, models.GenericConflictError, etc.) not raw fmt.Errorf — the HTTP error encoder maps these to correct status codes. (`return nil, models.NewGenericNotFoundError(models.NamespacedID{...})`)
**pkg/datetime for all calendar arithmetic** — Month/year duration arithmetic must use pkg/datetime.DateTime.Add(ISODuration) not time.Duration arithmetic — month lengths vary and naive Duration math produces wrong billing period boundaries. (`datetime.DateTime.Add(datetime.DurationMonth)`)
**pkg/filter dual-output filter types** — Query filters must implement both Ent selector predicates and go-sqlbuilder WHERE expressions via the pkg/filter.Filter interface. Setting multiple operator fields on one node is invalid — use Validate() to catch this early. (`filter.FilterString{In: []string{"a", "b"}}.Select(predicate)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `pkg/framework/entutils/transaction.go` | TransactingRepo / TransactingRepoWithNoValue — read/write ctx-bound Ent transaction; central to all adapter DB access | Never call creator.Tx() or store *entdb.Tx on adapter structs — use TransactingRepo instead |
| `pkg/framework/transport/httptransport/handler.go` | Generic Handler[Request,Response] decode/operate/encode pipeline with ErrorEncoder chain | Chain adds middleware; AppendOptions adds encoders — do not replace h.options |
| `pkg/framework/lockr/locker.go` | pg_advisory_xact_lock distributed lock — requires active Postgres transaction in ctx | Locker panics if no tx in ctx; use SessionLocker for connection-scoped locks |
| `pkg/models/validationissue.go` | ValidationIssue immutable value type with With* copy-on-write builder | Never mutate a ValidationIssue in place — always use With* methods |
| `pkg/models/servicehook.go` | Generic ServiceHookRegistry for cross-domain lifecycle callbacks with re-entrant loop prevention | Loop-prevention uses pointer-identity context key — do not copy the registry |
| `pkg/timeutil/recurrence.go` | Anchor-based billing period recurrence — not epoch-based | Always anchor to a truncated time; sub-millisecond drift produces off-by-nanosecond period boundaries |
| `pkg/kafka/config.go` | Typed Kafka config structs producing kafka.ConfigMap with validation | Never set kafka.ConfigMap keys directly — bypasses validation and localhost IPv4 auto-fix |
| `pkg/pagination/page.go` | Page/Result[T] contract for all domain List methods; Result.MarshalJSON flattens page fields | Use MapResult/MapResultErr — manual Result construction misses TotalCount and Page echo |

## Anti-Patterns

- Importing openmeter/* domain packages from any pkg/ sub-package — creates circular dependencies; pkg/ is a leaf
- Using time.Now() instead of clock.Now() in any pkg/ or openmeter/ production code
- Returning plain fmt.Errorf from service/adapter code where models.Generic* typed errors are expected — HTTP encoder falls through to 500
- Using time.Duration arithmetic for month/year periods instead of pkg/datetime ISODuration arithmetic
- Calling entutils.TransactingRepo with a raw *entdb.Client in a helper that is called inside a transaction — must still wrap so rebinding honors the ctx tx

## Decisions

- **pkg/ has zero imports from openmeter/* domain packages** — Prevents circular dependency; all seven domain binaries and their test suites depend on pkg/ — any openmeter/* import would create a cycle
- **pkg/framework/entutils.TransactingRepo reads transaction from ctx rather than accepting *entdb.Tx** — Ent transactions propagate implicitly via ctx; parameter-passing would leak tx plumbing into every adapter call site
- **pkg/models.ValidationIssue uses private constructor and With* copy-on-write pattern** — Immutability prevents accidental mutation as issues propagate through service → adapter → HTTP encoder chain

## Example: HTTP handler using the httptransport pipeline with error encoding

```
// pkg/framework/transport/httptransport used in openmeter/<domain>/httpdriver/handler.go
import (
    "github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
    "github.com/openmeterio/openmeter/pkg/models"
)

func (h *handler) ListFoo() http.Handler {
    return httptransport.NewHandler(
        func(ctx context.Context, r *http.Request) (ListFooInput, error) {
            // decode
            return ListFooInput{Namespace: ns}, nil
        },
        func(ctx context.Context, in ListFooInput) ([]Foo, error) {
            return h.svc.List(ctx, in)
        },
// ...
```

<!-- archie:ai-end -->
