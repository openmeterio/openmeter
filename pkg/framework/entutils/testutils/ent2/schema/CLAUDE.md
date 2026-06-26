# schema

<!-- archie:ai-start -->

> Test-fixture Ent schema package defining the Example2 entity used to exercise entutils transaction/mixin helpers against a second generated Ent client (ent2). Not production code — it exists only so testutils can generate db/ and validate cross-client behavior.

## Patterns

**Standard Ent schema struct** — Each entity is a struct embedding ent.Schema with the five Ent hook methods (Mixin, Fields, Indexes, Edges). Empty hooks return empty slices, never nil-by-omission. (`type Example2 struct { ent.Schema }; func (Example2) Edges() []ent.Edge { return []ent.Edge{} }`)
**Reuse entutils mixins, don't hand-roll fields** — Shared columns come from entutils mixins (e.g. entutils.TimeMixin{}) returned from Mixin(), not redeclared as fields. This mirrors production schemas under openmeter/ent/schema. (`func (Example2) Mixin() []ent.Mixin { return []ent.Mixin{ entutils.TimeMixin{} } }`)
**Immutable string id** — The primary key is an explicit field.String("id").Unique().Immutable() rather than a default int id. (`field.String("id").Unique().Immutable()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `example2.go` | Defines the Example2 test entity (id + example_value_2 string field, TimeMixin) consumed by the generated ent2/db client. | Field/mixin changes require regenerating ent2/db (the dependent generated package). Keep this aligned with the parallel ent1 fixture so transaction tests cover both clients consistently. |

## Anti-Patterns

- Adding business/domain logic here — this is a throwaway test fixture, not a real schema.
- Editing the generated ent2/db package by hand instead of changing this schema and regenerating.
- Returning nil from Indexes()/Edges() instead of empty slices, breaking the established convention.

## Decisions

- **Schema lives under pkg/framework/entutils/testutils so transaction helpers can be tested without depending on production openmeter/ent schemas.** — Keeps testutils self-contained and avoids import cycles with real domain packages while still exercising real Ent codegen + entutils mixins.

## Example: Defining a fixture Ent entity that pulls shared columns from entutils mixins

```
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type Example2 struct{ ent.Schema }

func (Example2) Mixin() []ent.Mixin { return []ent.Mixin{entutils.TimeMixin{}} }

func (Example2) Fields() []ent.Field {
	return []ent.Field{
// ...
```

<!-- archie:ai-end -->
