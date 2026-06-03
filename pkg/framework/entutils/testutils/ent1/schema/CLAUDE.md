# schema

<!-- archie:ai-start -->

> Test fixture Ent schema used exclusively by entutils transaction tests. Provides a minimal single-entity schema (Example1) to exercise TransactingRepo, TxCreator, and related Ent test infrastructure without coupling to production domain schemas.

## Patterns

**Embed ent.Schema** — Every schema struct embeds ent.Schema as an anonymous field — required by Ent's code generator. (`type Example1 struct { ent.Schema }`)
**Use entutils.TimeMixin** — Apply entutils.TimeMixin{} in Mixin() to get standard created_at/updated_at timestamps consistent with production schemas. (`func (Example1) Mixin() []ent.Mixin { return []ent.Mixin{entutils.TimeMixin{}} }`)
**Immutable string ID field** — Declare the primary key as field.String("id").Unique().Immutable() — matches production ULID/string PK conventions. (`field.String("id").Unique().Immutable()`)
**Return empty slices for unused schema hooks** — Return []ent.Index{} and []ent.Edge{} when there are none — do not omit the methods. (`func (Example1) Indexes() []ent.Index { return []ent.Index{} }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `example1.go` | Sole test entity schema: a minimal two-field entity (id + example_value_1) with TimeMixin for testing entutils transaction helpers. | Do not add production business logic; schema changes require regenerating the testutils ent client via go generate in the testutils package. |

## Anti-Patterns

- Adding production domain fields or edges — this schema is a test fixture only.
- Omitting entutils.TimeMixin — breaks consistency with the production schema conventions entutils tests rely on.
- Using integer or UUID primary keys instead of string — mismatches the ULID string PK pattern.
- Skipping Indexes() or Edges() — Ent codegen expects all four interface methods present.

## Decisions

- **Minimal schema with only two fields** — Test fixtures need just enough structure to exercise transaction semantics (read/write/rollback) without the maintenance burden of real domain entities.
- **Reuse entutils.TimeMixin instead of raw timestamp fields** — Ensures the test schema exercises the same mixin path as production, catching TimeMixin regressions.

## Example: Defining a new test-fixture Ent schema entity with TimeMixin and string PK

```
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type Example1 struct { ent.Schema }
func (Example1) Mixin() []ent.Mixin { return []ent.Mixin{entutils.TimeMixin{}} }
func (Example1) Fields() []ent.Field { return []ent.Field{ field.String("id").Unique().Immutable(), field.String("example_value_1") } }
func (Example1) Indexes() []ent.Index { return []ent.Index{} }
func (Example1) Edges() []ent.Edge { return []ent.Edge{} }
```

<!-- archie:ai-end -->
