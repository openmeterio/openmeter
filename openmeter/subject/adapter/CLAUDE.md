# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing subject.Adapter — all CRUD and list operations for subjects, with transaction-aware execution via entutils.TransactingRepo. This is the sole persistence layer for the subject domain; correctness depends on every method honoring the ctx-carried transaction.

## Patterns

**TransactingRepo wrapping** — Every public method wraps its Ent queries inside entutils.TransactingRepo (returning value) or entutils.TransactingRepoWithNoValue (no return value) so the ctx-bound Ent transaction is honored. Never call tx.db.* directly outside this wrapper. (`return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (subject.Subject, error) { ... tx.db.Subject.Query()... })`)
**TxUser interface satisfaction** — adapter implements entutils.TxUser[*adapter] by providing Tx, WithTx, and Self methods. adapter.go is the sole file defining these; subject.go adds no tx infrastructure. (`var _ entutils.TxUser[*adapter] = (*adapter)(nil)`)
**Soft-delete via DeletedAt** — Subjects are soft-deleted: Delete sets DeletedAt=now. All query methods filter with subjectdb.Or(subjectdb.DeletedAtIsNil(), subjectdb.DeletedAtGTE(now)) to exclude expired deletes; GetById does NOT apply this filter. (`subjectdb.Or(subjectdb.DeletedAtIsNil(), subjectdb.DeletedAtGTE(now))`)
**Error type mapping** — Ent db.IsNotFound → models.NewGenericNotFoundError; db.IsConstraintError → models.NewGenericConflictError; validation failures → models.NewGenericValidationError. Never return raw Ent errors to callers. (`if db.IsNotFound(err) { return subject.Subject{}, models.NewGenericNotFoundError(...) }`)
**Input validation before DB access** — Every write method calls input.Validate() (or key/id.Validate()) and wraps the error in models.NewGenericValidationError before entering the TransactingRepo closure. (`if err := input.Validate(); err != nil { return subject.Subject{}, fmt.Errorf("invalid input: %w", models.NewGenericValidationError(err)) }`)
**OptionalNullable patch pattern** — Update uses subject.OptionalNullable[T]{IsSet, Value} to distinguish JSON field absent vs. null. When IsSet=true and Value=nil, call ClearField(); when IsSet=true and Value!=nil, call SetField(*Value); skip field entirely if IsSet=false. (`if input.DisplayName.IsSet { if input.DisplayName.Value != nil { query.SetDisplayName(*input.DisplayName.Value) } else { query.ClearDisplayName() } }`)
**mapEntity as the only domain mapper** — All Ent *db.Subject → subject.Subject conversions go through the package-private mapEntity function. Nil metadata is normalised to an empty map inside mapEntity; do not add mapping logic elsewhere. (`return pagination.MapResult(result, mapEntity), nil`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Defines the adapter struct, New constructor, and the three TxUser methods (Tx, WithTx, Self). All tx infrastructure lives here. | Do not add Ent query code here; keep it in subject.go. Do not skip the nil-db guard in New. |
| `subject.go` | Implements all subject.Adapter methods: Create, Update, GetByKey, GetById, GetByIdOrKey, List, Delete, and the mapEntity helper. | Update calls GetById at the end to return the refreshed entity — do not return the stale pre-update row. Delete is idempotent: it returns nil if already deleted. |

## Anti-Patterns

- Calling tx.db.* directly outside a TransactingRepo/TransactingRepoWithNoValue closure
- Returning raw Ent errors instead of mapping to models.Generic*Error types
- Adding domain/business logic (hooks, event publishing) inside adapter methods
- Querying without the soft-delete filter (DeletedAtIsNil / DeletedAtGTE) in list/get-by-key operations
- Editing openmeter/ent/db/ generated files instead of the schema

## Decisions

- **OptionalNullable[T] instead of pointer-pointer for nullable patch fields** — Ent cannot distinguish between a JSON field being absent vs. explicitly null from a *T alone; IsSet flag makes the distinction explicit without needing a second parse of the raw body at the adapter layer.
- **Soft-delete (SetDeletedAt) rather than hard-delete** — Subjects are referenced by usage events in ClickHouse; hard-deleting would break historical query attribution. Soft-delete preserves the row while making the subject invisible to active queries.

## Example: Adding a new write method to the subject adapter

```
import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	subjectdb "github.com/openmeterio/openmeter/openmeter/ent/db/subject"
	"github.com/openmeterio/openmeter/openmeter/subject"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (a *adapter) Deactivate(ctx context.Context, id models.NamespacedID) (subject.Subject, error) {
	if err := id.Validate(); err != nil {
		return subject.Subject{}, fmt.Errorf("invalid id: %w", models.NewGenericValidationError(err))
	}
// ...
```

<!-- archie:ai-end -->
