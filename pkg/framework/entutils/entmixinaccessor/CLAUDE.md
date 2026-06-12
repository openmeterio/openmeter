# entmixinaccessor

<!-- archie:ai-start -->

> An Ent codegen extension (entc.Extension) that injects a Go template generating Get<Field>() accessor methods for every field added to schemas via a mixin. It gives mixin-contributed fields (e.g. from ResourceMixin, AnnotationsMixin) uniform getters across all generated entities, so generic code can read shared fields without per-entity casts.

## Patterns

**entc.Extension via embedded template** — The extension is a struct embedding entc.DefaultExtension and overriding only Templates(), returning one gen.Template parsed from the embedded .tpl. New() returns *Extension. This matches the shape of sibling extensions (entcursor, entexpose, entpaginate, entsetorclear). (`type Extension struct { entc.DefaultExtension }; func (Extension) Templates() []*gen.Template { return []*gen.Template{gen.MustParse(gen.NewTemplate("entmixinaccessor").Parse(tmplfile))} }`)
**Template embedded with //go:embed** — Template logic lives in mixinaccessor.tpl, pulled in via `//go:embed mixinaccessor.tpl` into `var tmplfile string` with a blank `_ "embed"` import. No template logic in Go string literals. (`//go:embed mixinaccessor.tpl\nvar tmplfile string`)
**MixedIn-gated generation** — The template emits a getter only when the field/ID position is MixedIn ({{- if and $f.Position $f.Position.MixedIn }}). Schema-native fields are skipped. ID is handled separately via $n.ID.Position.MixedIn because Ent exposes it as $n.ID, not in $n.Fields. (`{{- if and $f.Position $f.Position.MixedIn }}func (e *{{ $n.Name }}) Get{{ $f.StructField }}() ...`)
**Nillable pointer-type fidelity** — Getters for Nillable fields return a pointer type (`{{ if $f.Nillable }}*{{ end }}{{ $f.Type }}`) to match the generated entity field, keeping the accessor signature identical to the struct field. (`func (e *X) GetDeletedAt() *time.Time { return e.DeletedAt }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `entmixinaccessor.go` | Extension definition: embeds the template, implements entc.Extension.Templates(), exposes New(). | The gen.NewTemplate name ("entmixinaccessor") must match the {{ define "entmixinaccessor" }} block in the .tpl, or gen.MustParse panics at generate time. |
| `mixinaccessor.tpl` | Codegen template: ranges $.Nodes and $n.Fields, emits Get<StructField>()/GetID() only for MixedIn positions, calls {{ template "header" $ }}. | Must invoke {{ template "header" $ }} for the standard ent header/package clause. Use $f.StructField (Go field name), not $f.Name (DB column). |

## Anti-Patterns

- Editing the generated Get*() methods in openmeter/ent/db/ directly — change this template and run `make generate`.
- Inlining the template as a Go string literal instead of keeping it in mixinaccessor.tpl with //go:embed.
- Generating getters for non-MixedIn (schema-native) fields, which would collide with other accessors.
- Returning a value type for Nillable fields (or vice versa), breaking the signature contract with the entity field.

## Decisions

- **Implement accessor generation as a standalone entc extension wired in openmeter/ent/entc.go alongside entcursor/entexpose/entpaginate/entsetorclear.** — Mixin fields are shared across many entities; uniform Get<Field>() lets generic code read them without reflection, and an extension keeps it inside the single entc.Generate codegen pass.
- **Gate on Position.MixedIn rather than an explicit field-name list.** — Keeps the template self-maintaining: any new mixin field automatically gets a getter and schema-native fields are never accidentally shadowed.

## Example: Wiring the extension into the Ent codegen pass (openmeter/ent/entc.go)

```
import "github.com/openmeterio/openmeter/pkg/framework/entutils/entmixinaccessor"

entc.Generate("./schema", &gen.Config{ /* ... */ },
	entc.Extensions(
		entcursor.New(),
		entexpose.New(),
		entmixinaccessor.New(),
		entpaginate.New(),
		entsetorclear.New(),
	),
)
```

<!-- archie:ai-end -->
