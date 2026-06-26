# entcursor

<!-- archie:ai-start -->

> An Ent codegen extension that attaches a Cursor(ctx, *pagination.Cursor) method to every generated Query type whose node has a created_at field, implementing keyset (cursor) pagination ordered by (created_at asc, id asc). It is registered in the Ent generator pipeline, not called at runtime.

## Patterns

**entc.Extension wrapper** — Extension embeds entc.DefaultExtension and overrides Templates() to register one gen.Template parsed from an embedded .tpl file; New() returns *Extension. (`func (Extension) Templates() []*gen.Template { return []*gen.Template{gen.MustParse(gen.NewTemplate("entcursor").Parse(tmplfile))} }`)
**Embedded template file** — The .tpl is loaded via //go:embed cursor.tpl into var tmplfile string; template logic lives entirely in cursor.tpl, Go file is just registration glue. (`//go:embed cursor.tpl\nvar tmplfile string`)
**created_at gating** — The template only emits Cursor() for nodes that declare a created_at field (it scans $n.Fields). Nodes without created_at get no method. (`{{ range $f := $n.Fields }}{{ if eq $f.Name "created_at" }}{{ $hasCreatedAt = true }}{{ end }}{{ end }}`)
**Keyset ordering invariant** — Cursor pagination always orders by created_at asc, id asc and the WHERE predicate is (created_at > t) OR (created_at = t AND CAST(id AS TEXT) > id). Changing one without the other breaks paging. (`s.OrderBy(sql.Asc(s.C("created_at")), sql.Asc(s.C("id")))`)
**Non-nil empty result** — When no rows match, Items is initialized to make([]*Node, 0), never nil, so callers/JSON get [] not null; NextCursor is only set when len(items) > 0. (`if items == nil { items = make([]*Node, 0) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `cursor.go` | Registers the entcursor template as an entc.Extension via embedded cursor.tpl | Template name string passed to gen.NewTemplate must match the {{ define }} block name in the .tpl |
| `cursor.tpl` | Ent text/template generating the Cursor() method per node with a created_at field | Edits here regenerate code only after re-running Ent codegen; the id comparison casts to TEXT (string ID assumption) and validates cursor before applying |
| `cursor_test.go` | Integration test exercising generated Cursor() against testutils/ent1 example schema on a real Postgres DB | Uses testutils.InitPostgresDB(t); covers first-page, next-page, invalid-cursor error, and empty-result paths |

## Anti-Patterns

- Editing the generated Cursor() methods in ent/db output instead of cursor.tpl
- Changing the WHERE predicate or ORDER BY in a way that desyncs them, breaking stable keyset paging
- Returning nil Items for empty results (must be make([]*Node, 0))
- Assuming Cursor() exists on nodes without a created_at field

## Decisions

- **Generate keyset pagination via an Ent template extension rather than hand-writing per-entity** — Uniform cursor semantics across all entities that have created_at without per-package boilerplate
- **Cast id to TEXT in the tiebreaker comparison** — Provides a deterministic total order for the secondary key regardless of the underlying id column type

<!-- archie:ai-end -->
