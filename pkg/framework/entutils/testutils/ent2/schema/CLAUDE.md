# schema

<!-- archie:ai-start -->

> Test-only Ent schema fixture providing a second distinct database schema (ent2) for entutils multi-database transaction isolation tests. Contains no production logic — exists solely to give the test harness two independent Ent schemas.

## Patterns

**Embed ent.Schema as first field** — Every schema struct must embed ent.Schema as its first field so Ent code generation recognises it as an entity. (`type Example2 struct { ent.Schema }`)
**Include entutils.TimeMixin in Mixin()** — All schema structs must return entutils.TimeMixin{} from Mixin() to get created_at/updated_at fields, consistent with production schema conventions. (`func (Example2) Mixin() []ent.Mixin { return []ent.Mixin{entutils.TimeMixin{}} }`)
**Immutable string primary key named 'id'** — The primary key is always a string field named 'id', marked Unique() and Immutable(), matching the project-wide ID convention. (`field.String("id").Unique().Immutable()`)
**Return empty slices for unused schema hooks** — Indexes() and Edges() must return empty slices (not nil) when unused. (`func (Example2) Indexes() []ent.Index { return []ent.Index{} }`)
**Minimal schema — no relations or complex fields** — Test fixture schemas must stay as small as possible: only an id and one or two value fields. No edges, no foreign keys, no domain logic. (`field.String("example_value_2")`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `example2.go` | Defines the Example2 Ent schema entity — the second fixture schema used in entutils multi-db transaction isolation tests. | Do not add real business logic, edges, or complex fields. Any schema change regenerates the test DB and can break entutils tests. |

## Anti-Patterns

- Adding production domain fields or edges to test fixture schemas
- Omitting entutils.TimeMixin from Mixin() — breaks consistency with production schema conventions
- Using a non-string or mutable ID field — violates the project-wide ULID/string ID convention
- Importing domain packages (openmeter/billing, openmeter/customer, etc.) into this test schema
- Returning nil instead of empty slice from Indexes() or Edges()

## Decisions

- **Separate ent2 schema from ent1 schema in testutils** — entutils transaction helpers must be tested against two independent Ent databases to verify cross-db transaction isolation; a distinct schema package avoids collision with the ent1 fixture.
- **Minimal schema with only id and one value field** — Keeping fixtures small makes generated test DB migrations fast and decouples entutils tests from domain schema evolution.

## Example: Define a new test entity in this schema package

```
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type Example2 struct {
	ent.Schema
}

func (Example2) Mixin() []ent.Mixin {
	return []ent.Mixin{
// ...
```

<!-- archie:ai-end -->
