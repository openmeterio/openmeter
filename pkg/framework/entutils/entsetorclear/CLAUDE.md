# entsetorclear

<!-- archie:ai-start -->

> Ent codegen extension that generates SetOrClear<Field> convenience methods on every optional, mutable field for both bulk (*Update) and single-row (*UpdateOne) Ent updater types. Registered once in the entc.Extension chain; all output lives in openmeter/ent/db/ — never edited by hand.

## Patterns

**entc.Extension with Templates() only** — Extension embeds entc.DefaultExtension and overrides only Templates(). No other Extension interface methods are implemented. New() returns *Extension — callers must pass the pointer, not the value. (`type Extension struct { entc.DefaultExtension }
func (Extension) Templates() []*gen.Template { return []*gen.Template{gen.MustParse(...)} }
func New() *Extension { return &Extension{} }`)
**go:embed for template source** — The Go template lives in setorclear.tpl and is embedded at compile time with //go:embed. Registration uses gen.MustParse(gen.NewTemplate(name).Parse(tmplfile)). Never inline template text in Go code. (`//go:embed setorclear.tpl
var tmplfile string`)
**Dual updater coverage** — Template iterates both *UpdateName (bulk) and *UpdateNameOne (single-row) for every optional, non-immutable field. Both paths must always be generated together — omitting one forces nil-branching at call sites. (`func (u *EntityUpdate) SetOrClearFoo(v *string) *EntityUpdate
func (u *EntityUpdateOne) SetOrClearFoo(v *string) *EntityUpdateOne`)
**IsView exclusion guard** — Template skips nodes where $n.IsView is true. Ent view nodes have no updater types; without the guard the template panics. Any template modification must preserve {{ if not ($n.IsView) }}. (`{{ range $n := $.Nodes }}
  {{ if not ($n.IsView) }}
    ...
  {{ end }}
{{ end }}`)
**Standard ent file header** — Template opens with {{ template "header" $ }} to emit the ent-standard package declaration and DO-NOT-EDIT notice. Removing it breaks the generated file's package declaration. (`{{ template "header" $ }}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `setorclear.go` | Registers the Extension and wires the embedded template into the Ent codegen pipeline. New() is the only public constructor. | Pass *Extension (pointer) to entc.Extensions() — passing the value type silently omits the embedded template registration. |
| `setorclear.tpl` | Go text/template that emits SetOrClear<Field> for every optional non-immutable field on both updater types. Sourced from ent/ent#1119. | Removing the IsView guard or the header template call will cause panics or malformed generated files. |
| `CLAUDE.md` | Architecture notes for this package — loaded automatically by Claude Code. | Keep in sync when template logic changes. |

## Anti-Patterns

- Editing SetOrClear methods in openmeter/ent/db/ — they are generated output and will be overwritten by make generate.
- Implementing SetOrClear logic manually in domain adapter code instead of calling the generated method.
- Registering this extension more than once in entc.Generate — produces duplicate method declarations that fail compilation.
- Removing the {{ if not ($n.IsView) }} guard from the template — Ent view nodes have no updater type and the template will panic.
- Adding business logic or non-template behavior to the Extension struct — Extensions must only return template/hook descriptors, not execute logic.

## Decisions

- **Implement as entc.Extension with an embedded .tpl file rather than a post-generation script.** — Extensions run inside the Ent codegen pipeline and receive the full typed schema graph, enabling per-field/per-node introspection without parsing generated Go source.
- **Cover both *Update and *UpdateOne in the same template pass.** — Ent exposes two independent update builder paths; omitting one forces callers to nil-branch at every call site and breaks the uniform SetOrClear contract.

## Example: Register the extension in openmeter/ent/entc.go so it runs during make generate.

```
//go:build ignore

package main

import (
	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
	"github.com/openmeterio/openmeter/pkg/framework/entutils/entsetorclear"
)

func main() {
	entc.Generate("./schema", &gen.Config{},
		entc.Extensions(entsetorclear.New()),
	)
}
```

<!-- archie:ai-end -->
