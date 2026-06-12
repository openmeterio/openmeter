# ent1

<!-- archie:ai-start -->

> Self-contained Ent codegen package whose `schema/` fixture (Example1) generates a standalone `db/` client used to test entutils transaction/mixin helpers (TransactingRepo, TxDriver, TimeMixin) in isolation from the production openmeter schema set.

## Patterns

**Ignored-build codegen driver** — entc.go is a `//go:build ignore` main package invoked by `go generate`; it is never compiled into the package. (`//go:build ignore
package main
func main() { entc.Generate("./schema", &gen.Config{Target: "./db", ...}) }`)
**Generate directive in package file** — generate.go carries the lone `//go:generate go run -mod=mod entc.go` directive; it is the only real source file in the package proper. (`package ent1
//go:generate go run -mod=mod entc.go`)
**Extension wiring stays in entc.go** — Cursor/expose/paginate extensions are registered here so the generated db client supports the same features the production client tests rely on. (`entc.Extensions(entcursor.New(), entexpose.New(), entpaginate.New())`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `entc.go` | Ent codegen entrypoint: targets `./db`, enables FeatureVersionedMigration/Lock/Upsert/ExecQuery, registers entcursor/entexpose/entpaginate extensions. | Must keep the `//go:build ignore` tag and the full extension list — ent1 is the richer fixture (includes entpaginate, unlike ent2). Dropping an extension silently removes generated methods the entutils tests depend on. |
| `generate.go` | Holds the `package ent1` declaration and the `//go:generate` directive that runs entc.go. | Do not move the directive elsewhere or hand-edit the generated `db/`; regenerate via `make generate` / `go generate`. |

## Anti-Patterns

- Hand-editing the generated db/ client instead of changing schema/ and regenerating.
- Removing entpaginate/entcursor/entexpose from the extension list when the entutils tests need that generated surface — ent1 intentionally carries the full set.
- Compiling entc.go into the package by removing its `//go:build ignore` tag.

## Decisions

- **Keep a tiny standalone Ent client separate from openmeter/ent.** — Lets entutils test its transaction and mixin helpers without importing the large production schema or risking import cycles.
- **ent1 is the feature-complete fixture (includes entpaginate).** — Provides a client exercising cursor + pagination + expose so those entutils sub-packages can be validated; ent2 is the leaner counterpart.

<!-- archie:ai-end -->
