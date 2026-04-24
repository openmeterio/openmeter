# pagination

<!-- archie:ai-start -->

> Offset-based pagination primitives (Page, Result[T], Paginator[T]) and a cursor-based v2 sub-package. Provides the standard list-query contract used by all domain service List methods: callers pass pagination.Page, services return pagination.Result[T]. The v2 sub-package provides cursor pagination for time+ID ordered queries.

## Patterns

**Page/Result contract for all list services** — All domain List methods accept pagination.Page (PageSize + PageNumber) embedded in their ListInput struct and return pagination.Result[T] with Items, TotalCount, and the echoed Page. Use MapResult[Out, In] or MapResultErr[Out, In] to transform items without rebuilding the Result wrapper. (`func (s *service) List(ctx context.Context, params ListParams) (pagination.Result[Entity], error) {
    rows, total, err := s.adapter.List(ctx, params)
    return pagination.MapResult(rows, toDomain), nil
}`)
**Paginator[T] via NewPaginator for iterative collection** — Wrap any list function as a Paginator[T] using NewPaginator[T](fn). Use CollectAll[T](ctx, paginator, pageSize) to accumulate all pages up to MAX_SAFE_ITER (10,000). CollectAll returns (nil, err) on any page error; it does not return partial results. (`p := pagination.NewPaginator[Customer](func(ctx context.Context, page pagination.Page) (pagination.Result[Customer], error) {
    return svc.List(ctx, ListParams{Page: page})
})
all, err := pagination.CollectAll[Customer](ctx, p, 100)`)
**Result MarshalJSON flattens Page fields** — Result[T].MarshalJSON() promotes PageSize and PageNumber into the top-level JSON object (not nested under 'page'). The Page field in Result has json:"-". This is intentional for API wire format consistency — do not override with a custom marshaler. (`// Output: {"pageSize":10,"page":1,"totalCount":25,"items":[...]}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `pkg/pagination/page.go` | Page value type with Offset() and Limit() helpers. Validate() returns InvalidError if PageSize < 0 or PageNumber < 1. IsZero() is true when both are 0 (uninitialised). | Offset() uses 1-based PageNumber: (PageNumber-1)*PageSize. PageNumber must be >= 1 for valid pages. |
| `pkg/pagination/result.go` | Result[T] generic with MapResult and MapResultErr helpers. MarshalJSON flattens Page into the JSON root — this is load-bearing for API compatibility. | MapResultErr returns (Result[Out]{}, err) on first mapping error — no partial results. Use MapResult when the mapping is infallible. |
| `pkg/pagination/collect.go` | CollectAll iterates pages until Items count < pageSize, returning nil on error. MAX_SAFE_ITER = 10_000 caps infinite loops from misbehaving paginators. | Uses 1-based page numbering (starts at PageNumber=1). On error returns (nil, err) — not partial results. |

## Anti-Patterns

- Implementing offset-based pagination by computing SQL OFFSET directly in handlers — always use Page.Offset() and Page.Limit() so the contract stays consistent.
- Constructing pagination.Result manually instead of using MapResult/MapResultErr — the Page field echo and TotalCount assignment are easy to get wrong.
- Ignoring the error return from CollectAll — it returns nil items on error, so checking only len(items) will silently lose data.

## Decisions

- **Result[T] flattens Page into the JSON root rather than nesting it under a 'page' key.** — API wire format requires pageSize and page at the top level for SDK compatibility; the embedded Page struct is the internal type but must not appear in serialized responses.

<!-- archie:ai-end -->
