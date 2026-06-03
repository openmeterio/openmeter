# pagination

<!-- archie:ai-start -->

> Offset-based pagination primitives (Page, Result[T], Paginator[T], CollectAll) used by every domain List method, with a cursor-based v2 sub-package for time+ID keyset pagination. Constraint: list code must use these types — never compute SQL OFFSET/LIMIT in handlers, and never mix offset and cursor contracts on one endpoint.

## Patterns

**Page/Result contract for List methods** — List methods embed pagination.Page (1-based PageNumber + PageSize) in their input and return pagination.Result[T] {Items, TotalCount, Page}. Transform items with MapResult/MapResultErr — never build Result manually, so the echoed Page and TotalCount stay correct. (`return pagination.MapResult(pagination.Result[dbEntity]{Items: rows, TotalCount: total, Page: params.Page}, toDomain), nil`)
**Page.Offset()/Limit() in adapters** — Adapters compute SQL paging via page.Offset() (=PageSize*(PageNumber-1)) and page.Limit() (=PageSize). Validate with page.Validate() (InvalidError if PageSize<0 or PageNumber<1); IsZero() is true only when both fields are 0. (`rows, err := q.Offset(params.Page.Offset()).Limit(params.Page.Limit()).All(ctx)`)
**NewPaginator + CollectAll for full scans** — Wrap a list function as Paginator[T] via NewPaginator[T](fn), then CollectAll[T](ctx, paginator, pageSize) accumulates all pages until Items count < pageSize, capped at MAX_SAFE_ITER (10,000). On any page error it returns (nil, err) — never partial results. (`all, err := pagination.CollectAll[Customer](ctx, p, 100)`)
**Result.MarshalJSON flattens Page** — Result[T].MarshalJSON() promotes PageSize and PageNumber to the top-level JSON object (Page has json:"-"). This flattening is load-bearing for SDK wire compatibility — do not override it. (`// {"pageSize":10,"page":1,"totalCount":25,"items":[...]}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `page.go` | Page value type; Offset()/Limit()/Validate()/IsZero(); NewPage and NewPageFromRef (pointer query params). | PageNumber must be >=1 for valid pages; IsZero() (both 0) means uninitialised, not page 1. |
| `result.go` | Result[T] with MapResult and MapResultErr; custom MarshalJSON flattens Page into the JSON root. | MapResultErr returns (Result{}, err) on the first mapping failure — no partial results. |
| `collect.go` | CollectAll iterating pages until a short page; MAX_SAFE_ITER=10,000 caps runaway paginators. | Returns (nil, err) on error — checking only len(items) silently drops the error. |
| `pagination.go` | Paginator[T] interface and NewPaginator[T](fn); unexported paginator[T] is the only implementation. | Never implement Paginator[T] in domain code — always wrap a closure via NewPaginator. |

## Anti-Patterns

- Computing SQL OFFSET/LIMIT directly in handler or service code instead of Page.Offset()/Limit().
- Constructing pagination.Result manually instead of MapResult/MapResultErr.
- Ignoring the error from CollectAll — it returns nil items on error.
- Implementing a custom Paginator[T] type rather than using NewPaginator.
- Mixing v2 cursor logic with offset Page/Result on the same endpoint — the contracts are incompatible.

## Decisions

- **Result[T] flattens Page into the JSON root rather than nesting under a 'page' key.** — The API wire format requires pageSize and page at the top level for SDK compatibility; the embedded Page is internal-only.
- **CollectAll caps at MAX_SAFE_ITER=10,000 and returns (nil, error) on any page error.** — Prevents infinite loops from misbehaving paginators and makes error handling unambiguous — callers get all items or nil, never a deceptive partial slice.

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
