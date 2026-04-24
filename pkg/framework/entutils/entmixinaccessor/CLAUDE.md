# entmixinaccessor

<!-- archie:ai-start -->

> Ent code-generation extension that generates typed `GetXxx()` getter methods for every field (including the ID) that was added to an entity schema via a mixin, enabling interface-based access to mixin fields without reflection.

## Patterns

**Ent Extension + embedded template** — Same shape as all other entutils extensions: `entmixinaccessor.go` registers `mixinaccessor.tpl` via `entc.DefaultExtension` + `Templates()`. Register in `openmeter/ent/entc.go`. (`func New() *Extension { return &Extension{} }`)
**MixedIn position gate** — Getters are only generated for fields where `$f.Position.MixedIn` is true, i.e., fields contributed by a mixin. Fields defined directly on the schema do not get generated getters. (`{{- if and $f.Position $f.Position.MixedIn }}`)
**Nillable fields get pointer return type** — If a mixin field is `Nillable`, the getter returns `*<Type>` matching the struct field's actual type. (`func (e *{{ $n.Name }}) Get{{ $f.StructField }}() {{ if $f.Nillable }}*{{ end }}{{ $f.Type }} { return e.{{ $f.StructField }} }`)
**ID getter is templated separately** — The ID field is not in `$n.Fields`; it is accessed via `$n.ID` and gets its own `GetID()` getter only when `$n.ID.Position.MixedIn` is true. (`{{- if and $n.ID $n.ID.Position $n.ID.Position.MixedIn }} func (e *{{ $n.Name }}) GetID() {{ $n.ID.Type }} { return e.ID } {{- end }}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `mixinaccessor.tpl` | Generates typed getters for mixin-injected fields on every entity. Enables domain adapter code to access e.g. `entity.GetNamespace()` via a shared interface without type-asserting. | Only mixin fields get getters. If you add a field directly to a schema and need a getter for interface compliance, add it manually or move it to a mixin. |
| `entmixinaccessor.go` | Extension registration. Single responsibility: mount the template into Ent codegen. | Must be included in the `entc.Generate` extension list to have any effect. |

## Anti-Patterns

- Accessing mixin fields like `namespace` via direct struct field access in interface-based adapter code — use the generated `GetNamespace()` getter so the interface stays stable.
- Defining a field directly on a schema when it should be a mixin contribution — direct fields do not get getters from this extension.

## Decisions

- **Generate getters only for mixin fields, not all fields** — Mixin fields (namespace, timestamps, ID) are the ones shared across many entities and needed for generic adapter interfaces; generating getters for every field would bloat the generated code.

<!-- archie:ai-end -->
