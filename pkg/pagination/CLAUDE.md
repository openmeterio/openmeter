# pagination

<!-- archie:ai-start -->

> Offset-based pagination primitives (Page, Result[T], Paginator[T], CollectAll) used by all domain service List methods, plus a cursor-based v2 sub-package for time+ID keyset pagination. The primary constraint is that all list queries must use these types — never compute SQL OFFSET or LIMIT directly in handlers.

## Patterns

**Page/Result contract for all list services** — All domain List methods accept pagination.Page (PageSize + PageNumber, 1-based) embedded in their ListInput struct and return pagination.Result[T] with Items, TotalCount, and echoed Page. Use MapResult[Out, In] or MapResultErr[Out, In] to transform items. Never construct Result manually — use MapResult to get Page and TotalCount correct. (`func (s *service) List(ctx context.Context, params ListParams) (pagination.Result[Entity], error) {
    rows, total, err := s.adapter.List(ctx, params)
    if err != nil { return pagination.Result[Entity]{}, err }
    return pagination.MapResult(pagination.Result[dbEntity]{Items: rows, TotalCount: total, Page: params.Page}, toDomain), nil
}`)
**NewPaginator wraps list functions for CollectAll** — Wrap any list function as Paginator[T] via NewPaginator[T](fn). Use CollectAll[T](ctx, paginator, pageSize) to accumulate all pages up to MAX_SAFE_ITER (10,000). CollectAll returns (nil, err) on any page error — never partial results. Termination: Items count < pageSize signals last page. (`p := pagination.NewPaginator[Customer](func(ctx context.Context, page pagination.Page) (pagination.Result[Customer], error) {
    return svc.List(ctx, ListParams{Page: page})
})
all, err := pagination.CollectAll[Customer](ctx, p, 100)`)
**Page.Offset() and Page.Limit() for SQL queries** — Adapters compute SQL OFFSET and LIMIT via page.Offset() (= PageSize*(PageNumber-1)) and page.Limit() (= PageSize). Validate with page.Validate() before use — returns InvalidError if PageSize < 0 or PageNumber < 1. IsZero() is true when both fields are 0 (uninitialised). (`rows, err := db.Entity.Query().Offset(params.Page.Offset()).Limit(params.Page.Limit()).All(ctx)`)
**Result.MarshalJSON flattens Page fields** — Result[T].MarshalJSON() promotes PageSize and PageNumber into the top-level JSON object — not nested under 'page'. The Page field in Result has json:"-". Do not override with a custom marshaler; this flattening is load-bearing for API wire format compatibility with all SDK clients. (`// Output: {"pageSize":10,"page":1,"totalCount":25,"items":[...]}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `pkg/pagination/page.go` | Page value type with Offset() and Limit() helpers, Validate(), and IsZero(). Offset() uses 1-based PageNumber: (PageNumber-1)*PageSize. | PageNumber must be >= 1 for valid pages. IsZero() returns true only when both PageSize and PageNumber are 0 — zero-value struct is considered uninitialised, not page 1. |
| `pkg/pagination/result.go` | Result[T] generic type with MapResult and MapResultErr helpers. MarshalJSON custom implementation flattens Page into JSON root — do not change. | MapResultErr returns (Result[Out]{}, err) on first mapping error — no partial results. Use MapResult when mapping is infallible. |
| `pkg/pagination/collect.go` | CollectAll iterates pages until Items count < pageSize; MAX_SAFE_ITER=10,000 caps infinite loops from misbehaving paginators. | Returns (nil, err) on error — never partial results. Uses 1-based page numbering starting at PageNumber=1. |
| `pkg/pagination/pagination.go` | Paginator[T] interface and NewPaginator[T](fn) constructor. The unexported paginator[T] struct is the only implementation. | Never implement Paginator[T] directly in domain code — always use NewPaginator with a closure to keep the interface stable. |

## Anti-Patterns

- Computing SQL OFFSET or LIMIT directly in handlers or service code — always use Page.Offset() and Page.Limit() to maintain consistent contract.
- Constructing pagination.Result manually instead of using MapResult/MapResultErr — the Page field echo and TotalCount assignment are easy to get wrong.
- Ignoring the error return from CollectAll — it returns nil items on error, so checking only len(items) silently loses data.
- Implementing a custom Paginator[T] type instead of using NewPaginator — the unexported paginator struct is the only implementation; wrap your list function via NewPaginator.
- Using cursor-based v2 logic in contexts expecting offset Page/Result — offset and cursor contracts are incompatible; choose one per endpoint.

## Decisions

- **Result[T] flattens Page into the JSON root rather than nesting it under a 'page' key.** — API wire format requires pageSize and page at the top level for SDK compatibility; the embedded Page struct is the internal type but must not appear nested in serialized responses.
- **CollectAll caps at MAX_SAFE_ITER = 10,000 pages and returns (nil, error) rather than partial results on any page error.** — Prevents infinite loops from misbehaving paginators and makes error handling unambiguous — callers either get all items or nil, never a partial slice that could be mistaken for the complete set.

## Example: Adapter list method using Page.Offset/Limit and returning pagination.Result

```
import (
    "context"
    "github.com/openmeterio/openmeter/pkg/pagination"
)

func (a *adapter) ListEntities(ctx context.Context, params ListParams) (pagination.Result[Entity], error) {
    q := a.db.Entity.Query().Where(entity.Namespace(params.Namespace))
    total, err := q.Count(ctx)
    if err != nil { return pagination.Result[Entity]{}, err }
    rows, err := q.Offset(params.Page.Offset()).Limit(params.Page.Limit()).All(ctx)
    if err != nil { return pagination.Result[Entity]{}, err }
    return pagination.MapResult(
        pagination.Result[*db.Entity]{Items: rows, TotalCount: total, Page: params.Page},
        toDomainEntity,
    ), nil
// ...
```

<!-- archie:ai-end -->
