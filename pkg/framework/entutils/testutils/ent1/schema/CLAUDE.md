# schema

<!-- archie:ai-start -->

> Minimal Ent schema fixture used to test the entutils package (mixins, TransactingRepo, TxDriver). The single Example1 schema is the source of truth that the sibling ../db generated client is built from.

## Patterns

**Standard Ent schema struct** — Each entity is a struct embedding ent.Schema and implementing the Mixin/Fields/Indexes/Edges methods, even when some return empty slices. (`type Example1 struct { ent.Schema }; func (Example1) Fields() []ent.Field { ... }`)
**Reuse entutils mixins, do not hand-roll timestamps** — Timestamp/audit fields come from entutils mixins (TimeMixin here), not from manually-declared created_at/updated_at fields. (`func (Example1) Mixin() []ent.Mixin { return []ent.Mixin{ entutils.TimeMixin{} } }`)
**String immutable id** — Primary id is a String field marked Unique().Immutable() rather than an int autoincrement. (`field.String("id").Unique().Immutable()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `example1.go` | Defines the Example1 Ent schema (TimeMixin + string id + example_value_1) used as a fixture for entutils tests. | Changing fields requires regenerating ../db via `make generate` (entc.go/generate.go); editing the generated db package directly will be overwritten. |

## Anti-Patterns

- Adding production domain fields here — this is a test fixture, not a real entity; real schemas live in openmeter/ent/schema.
- Manually adding created_at/updated_at fields instead of using entutils.TimeMixin.
- Editing the generated ../db client to reflect a schema change instead of running code generation.

## Decisions

- **Keep a tiny standalone Ent schema separate from the main openmeter schema set.** — entutils transaction/mixin helpers need a generated Ent client to exercise TransactingRepo/TxDriver in isolation without depending on the full application schema.

## Example: Defining an Ent schema fixture with an entutils mixin

```
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type Example1 struct{ ent.Schema }

func (Example1) Mixin() []ent.Mixin { return []ent.Mixin{entutils.TimeMixin{}} }

func (Example1) Fields() []ent.Field {
	return []ent.Field{
// ...
```

<!-- archie:ai-end -->
