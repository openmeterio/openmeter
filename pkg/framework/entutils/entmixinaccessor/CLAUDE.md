# entmixinaccessor

<!-- archie:ai-start -->

> Ent code-generation extension generating typed GetXxx() getters for every field (including ID) added to an entity schema via a mixin, enabling interface-based access to mixin fields (namespace, timestamps, ID) without reflection or direct struct access.

## Patterns

**Ent Extension + embedded template** — entmixinaccessor.go registers mixinaccessor.tpl via entc.DefaultExtension + Templates(). Register in openmeter/ent/entc.go. (`func New() *Extension { return &Extension{} }`)
**MixedIn position gate** — Getters are generated only for fields where $f.Position.MixedIn is true (contributed by a mixin); fields defined directly on the schema get none. (`{{- if and $f.Position $f.Position.MixedIn }}`)
**Nillable fields get pointer return type** — If a mixin field is Nillable, the getter returns *<Type> matching the struct field's actual type. (`func (e *{{ $n.Name }}) Get{{ $f.StructField }}() {{ if $f.Nillable }}*{{ end }}{{ $f.Type }} { return e.{{ $f.StructField }} }`)
**ID getter templated separately** — The ID field is not in $n.Fields; it is accessed via $n.ID and gets GetID() only when $n.ID.Position.MixedIn is true. (`{{- if and $n.ID $n.ID.Position $n.ID.Position.MixedIn }} func (e *{{ $n.Name }}) GetID() {{ $n.ID.Type }} { return e.ID } {{- end }}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `mixinaccessor.tpl` | Generates typed getters for mixin-injected fields on every entity, enabling adapters to access e.g. entity.GetNamespace() via a shared interface without type-asserting. | Only mixin fields get getters. A field added directly to a schema needs a manual getter or must be moved to a mixin for interface compliance. |
| `entmixinaccessor.go` | Extension registration — mounts the template into Ent codegen. | Must be included in the entc.Generate extension list to have any effect. |

## Anti-Patterns

- Accessing mixin fields like namespace via direct struct access in interface-based adapter code — use the generated GetNamespace() getter
- Defining a field directly on a schema when it should be a mixin contribution — direct fields get no getters

## Decisions

- **Generate getters only for mixin fields, not all fields** — Mixin fields (namespace, timestamps, ID) are shared across entities for generic adapter interfaces; getters for every field would bloat generated code.

<!-- archie:ai-end -->
