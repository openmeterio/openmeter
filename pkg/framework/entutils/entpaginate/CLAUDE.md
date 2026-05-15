# entpaginate

<!-- archie:ai-start -->

> Ent code-generation extension that attaches a `Paginate(ctx, pagination.Page)` method to every Ent query type, providing offset-based pagination with an automatic COUNT(*) and guaranteed non-nil Items slice. Generated for ALL entity types unconditionally (unlike entcursor which gates on `created_at`).

## Patterns

**Ent Extension + embedded template** — `paginate.go` registers `paginate.tpl` following the same pattern as all entutils extensions. (`func New() *Extension { return &Extension{} }`)
**Paginate is generated for ALL nodes (no gate)** — Unlike entcursor, the paginate template generates `Paginate(...)` for every entity, regardless of fields present. (`{{ range $n := $.Nodes }} ... Paginate ... {{ end }}`)
**Clone for COUNT, original for data** — The template clones the query for the COUNT(*) call (clearing fields and order) then runs the data query on the original. Callers must not mutate the query after passing it to `Paginate`. (`countQuery := {{ $receiver }}.Clone(); countQuery.ctx.Fields = []string{}; countQuery.order = nil`)
**Zero-value Page returns all items** — If `page.IsZero()` is true, limit=totalCount and offset=0 — every item is returned with `TotalCount` populated. (`if page.IsZero() { offset = 0; limit = count }`)
**Non-nil Items on empty result** — Returns `make([]*Node, 0)` when count==0 so JSON serialises as `[]` not `null`. (`pagedResponse.Items = make([]*{{ $n.Name }}, 0)`)
**Static Paginator interface type-check** — Each generated `Paginate` method is validated against `pagination.Paginator[*Node]` via a compile-time `var _ =` assertion. (`var _ pagination.Paginator[*{{ $n.Name }}] = (*{{ $n.QueryName }})(nil)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `paginate.tpl` | Generates `Paginate(ctx, pagination.Page)` on every Ent query type. The two-query pattern (COUNT + data) means 2 DB round trips per paginated call. | Ordering applied before `.Paginate()` is preserved for data query but stripped for COUNT. Callers must apply ordering before calling Paginate, not after. |
| `paginate.go` | Extension registration only. | Must be registered in `openmeter/ent/entc.go` alongside the other entutils extensions. |
| `paginate_test.go` | Integration tests covering first page, ordering, filtering, multi-page, empty page, and zero-page (return all) scenarios against real Postgres. | Uses `testutils.InitPostgresDB` — requires `POSTGRES_HOST=127.0.0.1` and `-tags=dynamic`. |

## Anti-Patterns

- Applying `.Limit()` or `.Offset()` before `.Paginate()` — the template resets both to 0 on entry, discarding any caller-set values.
- Using `Paginate` for large unbounded exports — the zero-page path fetches ALL rows into memory; use cursor pagination via `entcursor` instead.
- Relying on COUNT accuracy under high concurrency — the two-query pattern has a TOCTOU gap; TotalCount may differ slightly from Items length.

## Decisions

- **Two-query approach (COUNT + data) rather than SQL window function** — Ent's query builder does not expose a window function API; a clone-and-count is the simplest correct approach with the existing Ent abstraction.
- **Generate for all nodes unconditionally** — Offset pagination requires no specific schema field (unlike cursor pagination's `created_at` requirement), so there is no meaningful gate.

## Example: Offset-paginated list of entities with ordering and filtering

```
import (
	"github.com/openmeterio/openmeter/pkg/pagination"
	"entgo.io/ent/dialect/sql"
)

func listPage(ctx context.Context, client *db.Client, ns string, page pagination.Page) (pagination.Result[*db.MyEntity], error) {
	return client.MyEntity.Query().
		Where(myentity.NamespaceEQ(ns)).
		Order(myentity.ByCreatedAt(sql.OrderDesc())).
		Paginate(ctx, page)
}
```

<!-- archie:ai-end -->
