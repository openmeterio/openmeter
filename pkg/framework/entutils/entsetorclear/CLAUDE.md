# entsetorclear

<!-- archie:ai-start -->

> Ent codegen extension that adds a SetOrClear<Field> convenience method to every optional, mutable field on all Ent updater types, covering both the bulk-update and UpdateOne paths. Registered once in the Ent entc.Extension chain; produces code into the generated ent/db layer.

## Patterns

**entc.Extension implementation** — Extension embeds entc.DefaultExtension and overrides only Templates(). Any new codegen behavior must follow the same Extension interface — do not add logic to non-Extension types. (`type Extension struct { entc.DefaultExtension }; func (Extension) Templates() []*gen.Template { return []*gen.Template{...} }`)
**Embedded template via go:embed** — The Go template source lives in setorclear.tpl and is embedded at compile time with //go:embed. Template registration uses gen.MustParse(gen.NewTemplate(name).Parse(tmplfile)). (`//go:embed setorclear.tpl
var tmplfile string
gen.MustParse(gen.NewTemplate("entsetorclear").Parse(tmplfile))`)
**Dual updater coverage** — The template iterates both *UpdateName (bulk) and *UpdateOneOne updaters for every optional non-immutable field so both update paths get SetOrClear. (`func (u *EntityUpdate) SetOrClearFoo(value *string) *EntityUpdate
func (u *EntityUpdateOne) SetOrClearFoo(value *string) *EntityUpdateOne`)
**View exclusion guard** — The template skips nodes where $n.IsView is true. Any future template changes must preserve this guard — Ent views have no updater types. (`{{ if not ($n.IsView) }}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `setorclear.go` | Registers the extension and wires the embedded template into the Ent codegen pipeline via New(). | New() returns *Extension — callers must pass this pointer to entc.Generate options, not the value type. |
| `setorclear.tpl` | Go text/template that emits SetOrClear<Field> methods. Sourced from ent/ent#1119. | Template uses {{ template "header" $ }} which pulls in the ent-standard file header — removing it breaks the generated file's package declaration. |

## Anti-Patterns

- Editing generated files in openmeter/ent/db/ that contain SetOrClear methods — they are output from this extension.
- Adding SetOrClear logic directly in domain adapter code instead of relying on this generated method.
- Registering the extension more than once in entc.Generate — it produces duplicate method declarations.
- Removing the IsView guard — Ent view nodes have no updater and the template will panic.

## Decisions

- **Implement as an entc.Extension with an embedded .tpl file rather than a post-generation script.** — Extensions run inside the Ent codegen pipeline and receive the full typed schema graph, enabling per-field/per-node introspection without parsing generated Go source.
- **Cover both *Update and *UpdateOne in the same template.** — Ent exposes two independent update builder paths; omitting one forces callers to branch on nil at every call site.

## Example: Register the extension in ent/entc.go so it runs during `make generate`.

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
