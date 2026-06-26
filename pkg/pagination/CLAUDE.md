# pagination

<!-- archie:ai-start -->

> Offset/limit (v1) pagination primitives used by ~119 packages: Page{PageSize,PageNumber}, Result[T] page wrapper with custom JSON flattening, a closure-backed Paginator[T] interface, and CollectAll to drain all pages. The newer cursor/keyset model lives in the pkg/pagination/v2 sub-package.

## Patterns

**Page is 1-based with derived Offset/Limit** — PageNumber starts at 1; Offset()=PageSize*(PageNumber-1), Limit()=PageSize. Validate() rejects negative size and PageNumber<1. Build via NewPage / NewPageFromRef (nil-pointer-safe for query params). (`page.Offset() / page.Limit() in page.go feed SQL OFFSET/LIMIT`)
**Paginator via NewPaginator closure, not a custom struct** — Wrap a `func(ctx, Page) (Result[T], error)` in NewPaginator[T]; the unexported paginator struct satisfies Paginator[T]. Never define a new Paginator implementation struct. (`NewPaginator[int](func(ctx, page) (Result[int], error){...})`)
**Result JSON flattens Page fields** — Result.MarshalJSON hoists PageSize/PageNumber to top-level and emits `{pageSize,page,totalCount,items}` in that exact order (field ordering is test-enforced). Result tags Page with json:"-". (`Result[int] marshals to {"pageSize":10,"page":1,"totalCount":3,"items":[...]}`)
**Map results with MapResult / MapResultErr** — Transform Result[In] to Result[Out] preserving Page+TotalCount via MapResult (pure) or MapResultErr (fallible mapper). Use these in httpdriver layers rather than rebuilding Result manually. (`pagination.MapResult(domainResult, ToAPIThing)`)
**CollectAll is bounded and not partial-tolerant** — CollectAll drains pages until a short page, capped at MAX_SAFE_ITER (10_000) pages, and returns nil items on ANY error (unlike v2.CollectAll). Pass the real page size used by the paginator. (`CollectAll(ctx, paginator, pageSize) — short page ends the loop`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `page.go` | Page struct, NewPage/NewPageFromRef, Offset/Limit/Validate/IsZero, InvalidError. | PageNumber is 1-based; Offset underflows to negative if you pass PageNumber=0 — always Validate or use NewPageFromRef defaults. |
| `result.go` | Result[T] with custom MarshalJSON flattening + MapResult/MapResultErr converters. | JSON key order is asserted in pagination_test.go; changing MarshalJSON field order breaks the test and API contract. |
| `pagination.go` | Paginator[T] interface + NewPaginator closure factory. | Only the closure form is supported; the concrete paginator type is unexported. |
| `collect.go` | CollectAll drains a Paginator across pages with a 10_000-page safety cap. | Returns nil (not the partial slice) on error or when MAX_SAFE_ITER is exceeded; v2's CollectAll behaves differently. |

## Anti-Patterns

- Treating PageNumber as 0-based — Offset() assumes 1-based and goes negative otherwise.
- Reordering or adding fields in Result.MarshalJSON without updating the order-enforcing test and API consumers.
- Defining a bespoke Paginator[T] implementation instead of NewPaginator.
- Assuming CollectAll returns the items gathered so far on error — it returns nil.
- Using this offset/limit model where keyset/cursor semantics are needed — that is pkg/pagination/v2.

## Decisions

- **Result flattens Page into the JSON envelope** — Keeps the wire shape a single flat object {pageSize,page,totalCount,items} matching the OpenAPI paginated response, while Result stays a typed struct internally.
- **Paginator is a closure-backed generic interface** — Lets any repo expose pagination by supplying one function, enabling generic helpers like CollectAll/MapResult without per-entity boilerplate.
- **Keep v1 offset pagination separate from v2 cursor pagination** — Offset/limit and keyset cursors have incompatible state and guarantees; splitting them avoids mixing Page-based and Cursor-based APIs in one package.

## Example: Expose a repo result as a Paginator and drain it

```
p := pagination.NewPaginator[Item](func(ctx context.Context, page pagination.Page) (pagination.Result[Item], error) {
	rows, total, err := repo.list(ctx, page.Offset(), page.Limit())
	if err != nil {
		return pagination.Result[Item]{}, err
	}
	return pagination.Result[Item]{Page: page, TotalCount: total, Items: rows}, nil
})
all, err := pagination.CollectAll(ctx, p, 100)
```

<!-- archie:ai-end -->
