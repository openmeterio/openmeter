# entpaginate

<!-- archie:ai-start -->

> An Ent codegen extension that attaches an offset/limit Paginate(ctx, pagination.Page) method to every generated Query type, returning pagination.Result with TotalCount and asserting the query satisfies pagination.Paginator. Provides page-number pagination as the counterpart to entcursor's keyset pagination.

## Patterns

**entc.Extension wrapper** — Same registration shape as the sibling extensions: Extension embeds entc.DefaultExtension, Templates() registers embedded paginate.tpl under name 'entpaginate', New() returns *Extension. (`gen.MustParse(gen.NewTemplate("entpaginate").Parse(tmplfile))`)
**Clone for count vs page** — Paginate clones the query for the COUNT (clearing Fields and order on the count query) and uses the original for the paged fetch, so total count is computed independently of select/order. (`countQuery := receiver.Clone(); countQuery.ctx.Fields = []string{}; countQuery.order = nil`)
**Zero-page returns all** — If page.IsZero() the method sets offset=0, limit=count and returns all items while still populating TotalCount; empty count short-circuits to Items = make([]*Node,0). (`if page.IsZero() { offset = 0; limit = count }`)
**Paginator type check** — Each node emits a compile-time assertion that *XQuery implements pagination.Paginator[*Node]. (`var _ pagination.Paginator[*Node] = (*XQuery)(nil)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `paginate.go` | Registers the entpaginate template as an entc.Extension via embedded paginate.tpl | Registration name must match the {{ define }} block ('paginate') vs gen.NewTemplate name ('entpaginate') |
| `paginate.tpl` | Generates Paginate() (offset/limit + TotalCount) plus the Paginator interface assertion per node | Manipulates unexported query internals (receiver.ctx.Limit/Offset, .order, .ctx.Fields) — sensitive to Ent codegen internals; resets limit/offset to zero before paging |
| `paginate_test.go` | Integration test of generated Paginate() over testutils/ent1 example schema on real Postgres | Covers ordering, filtering, paging, empty page, and zero-page-returns-all behaviors; uses testutils.InitPostgresDB(t) |

## Anti-Patterns

- Editing generated Paginate() methods instead of paginate.tpl
- Returning nil Items for empty results (must be make([]*Node, 0))
- Assuming Paginate with a zero Page errors instead of returning all items
- Relying on select fields or ordering surviving into the count query (they are cleared)

## Decisions

- **Run a separate cloned COUNT query with fields/order stripped** — Total count must be independent of the page's projection and ordering for correct TotalCount
- **Treat a zero Page value as 'return all items'** — Lets callers request the full set through the same API without a special unpaged path

<!-- archie:ai-end -->
