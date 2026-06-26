# entsetorclear

<!-- archie:ai-start -->

> A code-generation-time Ent extension (entc.Extension) that injects a custom Go template into Ent's generator, emitting SetOrClear<Field> convenience methods on every entity updater for optional, non-immutable fields. It exists only to participate in `go generate` for openmeter/ent — it ships no runtime logic.

## Patterns

**entc.Extension via embedded template** — Extension embeds entc.DefaultExtension and overrides Templates() to return one gen.Template parsed from the //go:embed'd setorclear.tpl. New() returns *Extension. This is the only public surface. (`func (Extension) Templates() []*gen.Template { return []*gen.Template{gen.MustParse(gen.NewTemplate("entsetorclear").Parse(tmplfile))} }`)
**Template embedded at compile time** — setorclear.tpl is bound to the tmplfile string via `//go:embed setorclear.tpl`; the .tpl file must stay co-located in this package and keep its name, or the embed directive breaks the build. (`//go:embed setorclear.tpl\nvar tmplfile string`)
**Generated method is nil-pointer-driven set/clear** — For each optional non-immutable field the template emits SetOrClear<StructField>(value *T) on both the bulk updater (UpdateName) and the *One updater: nil value calls Clear<Field>(), non-nil calls Set<Field>(*value). (`func (u *FooUpdate) SetOrClearBar(value *string) *FooUpdate { if value == nil { return u.ClearBar() } return u.SetBar(*value) }`)
**Views are skipped** — The template guards `{{ if not ($n.IsView) }}` — ent.View schemas have no updaters, so SetOrClear methods are never generated for them. New view-related logic must respect this guard. (`{{ if not ($n.IsView) }} ... {{ end }}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `setorclear.go` | Defines the entc.Extension wrapper (Extension struct, Templates(), New()) and embeds the template string. | Keep New() returning *Extension — openmeter/ent/entc.go registers it via entsetorclear.New() inside entc.Generate options. Do not add runtime/business logic here; this package runs only during code generation. |
| `setorclear.tpl` | Ent Go-template (text/template + ent gen funcs) that renders the SetOrClear<Field> methods into the generated db package. | Must call `{{ template "header" $ }}` so the generated file gets the standard 'Code generated, DO NOT EDIT' header. Only Optional && !Immutable fields are eligible. Both UpdateName and UpdateName+'One' paths must stay in sync (ent has two update paths). |

## Anti-Patterns

- Adding runtime application logic here — this package is generator-only and is imported solely by openmeter/ent/entc.go at generate time.
- Renaming or moving setorclear.tpl without updating the //go:embed directive, which silently breaks compilation.
- Removing the `{{ if not ($n.IsView) }}` guard — views lack updaters and would produce invalid generated code.
- Hand-editing the generated SetOrClear methods in openmeter/ent/db; they carry the DO NOT EDIT header and are overwritten by `make generate`.

## Decisions

- **Implement set-vs-clear as a generated template method rather than per-field hand-written helpers.** — Optional pointer fields need uniform nil=clear / value=set semantics across every entity; the pattern (sourced from ent issue #1119) auto-applies to all eligible fields and stays in sync as the schema grows.
- **Package the template as an entc.Extension instead of a standalone codegen script.** — Lets openmeter/ent/entc.go register it alongside other Ent generation options in a single entc.Generate invocation.

## Example: Registering the extension in the Ent generator (openmeter/ent/entc.go).

```
import "github.com/openmeterio/openmeter/pkg/framework/entutils/entsetorclear"

err := entc.Generate("./schema", cfg, entc.Extensions(
    entsetorclear.New(),
    // ...other extensions
))
```

<!-- archie:ai-end -->
