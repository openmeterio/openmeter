# schema

<!-- archie:ai-start -->

> Test-only Ent schema fixture providing a second distinct database schema (ent2) for entutils multi-database transaction isolation tests. Contains no production logic — exists solely to give the test harness two independent Ent schemas.

## Patterns

**Embed ent.Schema as first field** — Every schema struct embeds ent.Schema first so Ent code generation recognizes it as an entity. (`type Example2 struct { ent.Schema }`)
**Include entutils.TimeMixin in Mixin()** — All schema structs return entutils.TimeMixin{} from Mixin() to get created_at/updated_at, matching production conventions. (`func (Example2) Mixin() []ent.Mixin { return []ent.Mixin{entutils.TimeMixin{}} }`)
**Immutable string primary key named 'id'** — The primary key is always a string field 'id', marked Unique() and Immutable(), matching the project-wide ID convention. (`field.String("id").Unique().Immutable()`)
**Return empty slices for unused hooks** — Indexes() and Edges() return empty slices (not nil) when unused. (`func (Example2) Indexes() []ent.Index { return []ent.Index{} }`)
**Minimal schema — no relations or complex fields** — Fixture schemas stay tiny: only an id and one or two value fields. No edges, foreign keys, or domain logic. (`field.String("example_value_2")`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `example2.go` | Defines the Example2 Ent entity — the second fixture schema for entutils multi-db transaction isolation tests. | Do not add real business logic, edges, or complex fields. Any schema change regenerates the test DB and can break entutils tests. |

## Anti-Patterns

- Adding production domain fields or edges to test fixture schemas.
- Omitting entutils.TimeMixin from Mixin().
- Using a non-string or mutable ID field — violates the ULID/string ID convention.
- Importing domain packages (openmeter/billing, openmeter/customer, etc.) into this test schema.
- Returning nil instead of an empty slice from Indexes() or Edges().

## Decisions

- **Separate ent2 schema from ent1 schema in testutils.** — entutils transaction helpers must be tested against two independent Ent databases to verify cross-db isolation; a distinct package avoids collision with the ent1 fixture.
- **Minimal schema with only id and one value field.** — Small fixtures keep generated test-DB migrations fast and decouple entutils tests from domain schema evolution.

## Example: Define a new test entity in this schema package

```
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type Example2 struct { ent.Schema }

func (Example2) Mixin() []ent.Mixin { return []ent.Mixin{entutils.TimeMixin{}} }

func (Example2) Fields() []ent.Field {
	return []ent.Field{field.String("id").Unique().Immutable(), field.String("example_value_2")}
}
// ...
```

<!-- archie:ai-end -->
