# entcursor

<!-- archie:ai-start -->

> Ent code-generation extension that attaches a Cursor(ctx, *pagination.Cursor) method to every Ent query type whose schema has a created_at field, enabling stable cursor pagination ordered by (created_at ASC, id ASC). Registered in openmeter/ent/entc.go; takes effect only after make generate.

## Patterns

**Ent Extension + embedded template** — The Extension embeds entc.DefaultExtension, implements Templates() returning a gen.MustParse'd template loaded via //go:embed cursor.tpl, and exposes New(). This is the only supported way to add generated methods here. (`func (Extension) Templates() []*gen.Template { return []*gen.Template{gen.MustParse(gen.NewTemplate("entcursor").Parse(tmplfile))} }`)
**created_at gate in template** — The template only emits Cursor(...) for nodes that have a field named created_at; schemas without it get no method. (`{{ if $hasCreatedAt }} ... {{ end }}`)
**Deterministic ordering created_at ASC, id ASC** — The generated method always appends ORDER BY created_at ASC, id ASC; callers must not add conflicting order clauses before calling .Cursor(). (`s.OrderBy(sql.Asc(s.C("created_at")), sql.Asc(s.C("id")))`)
**CAST(id AS TEXT) for cursor comparison** — The cursor WHERE clause casts id to TEXT for string comparison so it works with ULID/UUID IDs. Integer-ID schemas break this assumption (lexicographic, not numeric). (`b.WriteString("CAST("); b.WriteString(s.C("id")); b.WriteString(" AS TEXT) > "); b.Args(cursor.ID)`)
**Non-nil Items on empty result** — Generated code initialises items to an empty slice when nil so JSON serialises as [] not null. (`if items == nil { items = make([]*{{ $n.Name }}, 0) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `cursor.tpl` | Go template generating the Cursor method on every qualifying Ent query type at go generate time. | Rendered once per $.Nodes entry — changes affect ALL entities with created_at. The CAST assumes string IDs; numeric-ID schemas produce wrong results silently. |
| `cursor.go` | Registers the template as an Ent extension via entc.Extension. Must be referenced in openmeter/ent/entc.go to take effect. | If not listed in the entc.Generate call, no cursor methods are generated despite this code existing. |
| `cursor_test.go` | Integration test using testutils.InitPostgresDB; validates first-page, next-page, empty-result, and invalid-cursor paths. | Requires POSTGRES_HOST=127.0.0.1 and -tags=dynamic; uses the ent1 test schema, not production. |

## Anti-Patterns

- Adding ORDER BY before calling .Cursor() — the template appends its own order, making combined ordering undefined
- Using this extension on a schema with an integer ID — CAST-to-TEXT gives lexicographic, not numeric, ordering
- Editing cursor.tpl without running make generate — generated methods in openmeter/ent/db/ go stale
- Calling .Cursor() with a zero-value pagination.Cursor{} — cursor.Validate() returns an error

## Decisions

- **Composite (created_at, id) cursor key instead of offset pagination** — Offset pagination drifts under concurrent inserts; a (time, id) tuple is stable and avoids COUNT(*) on large tables.
- **Gate generation on created_at presence rather than an annotation** — All domain entities use the TimeMixin that adds created_at, so the gate is a zero-config convention.

## Example: Using the generated Cursor method to paginate an Ent query

```
import (
	paginationv2 "github.com/openmeterio/openmeter/pkg/pagination/v2"
)

func listPage(ctx context.Context, client *db.Client, cur *paginationv2.Cursor) (paginationv2.Result[*db.MyEntity], error) {
	return client.MyEntity.Query().
		Where(myentity.NamespaceEQ(ns)).
		Limit(50).
		Cursor(ctx, cur) // cur == nil returns first page
}
```

<!-- archie:ai-end -->
