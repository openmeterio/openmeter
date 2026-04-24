# schema

<!-- archie:ai-start -->

> Test-only Ent schema fixture for the entutils test harness (ent2 database). Provides a second distinct Ent schema used to validate multi-database transaction helpers without touching production schemas.

## Patterns

**Embed ent.Schema** — Every schema struct must embed ent.Schema as its first field. (`type Example2 struct { ent.Schema }`)
**Use entutils.TimeMixin** — All schema structs must include entutils.TimeMixin{} in their Mixin() return to get created_at/updated_at fields consistently. (`func (Example2) Mixin() []ent.Mixin { return []ent.Mixin{entutils.TimeMixin{}} }`)
**Immutable string ID** — The primary key field is a string named 'id', marked Unique() and Immutable(). (`field.String("id").Unique().Immutable()`)
**Return empty slices for unused schema hooks** — Indexes() and Edges() must return empty slices rather than nil when unused. (`func (Example2) Indexes() []ent.Index { return []ent.Index{} }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `example2.go` | Defines the Example2 Ent schema entity used as the second fixture schema in entutils multi-db transaction tests. | Do not add real business logic or relations here; this is a minimal test fixture. Adding edges or complex fields will change the generated test DB schema and break entutils tests. |

## Anti-Patterns

- Adding production domain fields or edges to test fixture schemas
- Omitting entutils.TimeMixin from Mixin() — breaks consistency with production schema conventions
- Using a non-string or mutable ID field — violates the project-wide ID convention
- Importing domain packages (openmeter/billing, openmeter/customer, etc.) into this test schema

## Decisions

- **Separate ent2 schema from ent1 schema in testutils** — entutils transaction helpers must be tested against two independent Ent databases to verify cross-db transaction isolation; using a distinct schema package avoids collision with the ent1 fixture.
- **Minimal schema with only id and one value field** — Test fixtures should be as small as possible to keep generated test DB migrations fast and to avoid coupling entutils tests to domain schema evolution.

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
