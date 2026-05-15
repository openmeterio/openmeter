# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing subject.Adapter — sole persistence layer for subjects with soft-delete semantics, ctx-propagated transaction support via entutils.TransactingRepo, and mapped domain errors. Every method must honor the ctx-carried Ent transaction.

## Patterns

**TransactingRepo wrapping on every method** — Every public method wraps its Ent queries in entutils.TransactingRepo (returning value) or entutils.TransactingRepoWithNoValue (no return value). Never call tx.db.* directly outside this wrapper. (`return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (subject.Subject, error) { return mapEntity(tx.db.Subject.Query()...First(ctx)) })`)
**TxUser[*adapter] triad in adapter.go** — adapter implements entutils.TxUser[*adapter] via Tx (HijackTx + NewTxDriver), WithTx (NewTxClientFromRawConfig), and Self(). These three methods live only in adapter.go; subject.go adds no tx infrastructure. (`var _ entutils.TxUser[*adapter] = (*adapter)(nil)`)
**Soft-delete via DeletedAt filter** — Delete sets DeletedAt=now rather than removing the row. List/GetByKey queries must filter with subjectdb.Or(subjectdb.DeletedAtIsNil(), subjectdb.DeletedAtGTE(now)). GetById skips this filter intentionally. (`subjectdb.Or(subjectdb.DeletedAtIsNil(), subjectdb.DeletedAtGTE(now))`)
**Ent error mapping to models.Generic*Error** — db.IsNotFound → models.NewGenericNotFoundError; db.IsConstraintError → models.NewGenericConflictError; validation failures → models.NewGenericValidationError. Never return raw Ent errors. (`if db.IsNotFound(err) { return subject.Subject{}, models.NewGenericNotFoundError(fmt.Errorf("subject not found [namespace=%s id=%s]", ...)) }`)
**Input validation before DB access** — Every write method calls input.Validate() (or key/id.Validate()) and wraps failures in models.NewGenericValidationError before entering the TransactingRepo closure. (`if err := input.Validate(); err != nil { return subject.Subject{}, fmt.Errorf("invalid input: %w", models.NewGenericValidationError(err)) }`)
**OptionalNullable[T] patch pattern in Update** — subject.OptionalNullable[T]{IsSet, Value} distinguishes JSON field absent vs. null. When IsSet=true and Value=nil call ClearField(); when IsSet=true and Value!=nil call SetField(*Value); skip when IsSet=false. (`if input.DisplayName.IsSet { if input.DisplayName.Value != nil { query.SetDisplayName(*input.DisplayName.Value) } else { query.ClearDisplayName() } }`)
**mapEntity as the sole domain mapper** — All *db.Subject → subject.Subject conversions go through the package-private mapEntity function. Nil metadata is normalised to an empty map inside mapEntity; do not add mapping logic elsewhere. (`return pagination.MapResult(result, mapEntity), nil`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Defines adapter struct, New constructor (nil-db guard), and the three TxUser methods (Tx, WithTx, Self). All transaction infrastructure lives here. | Do not add Ent query code here; keep queries in subject.go. Never skip the nil-db guard in New. |
| `subject.go` | Implements all subject.Adapter methods: Create, Update, GetByKey, GetById, GetByIdOrKey, List, Delete, and mapEntity. | Update calls GetById at the end to return the refreshed entity — never return the stale pre-update row. Delete is idempotent: returns nil if already deleted. GetById omits the soft-delete filter intentionally. |

## Anti-Patterns

- Calling tx.db.* directly outside a TransactingRepo/TransactingRepoWithNoValue closure
- Returning raw Ent errors instead of mapping to models.Generic*Error types
- Adding business logic or event publishing inside adapter methods
- Querying without the soft-delete filter (DeletedAtIsNil / DeletedAtGTE) in list/get-by-key operations
- Editing openmeter/ent/db/ generated files instead of openmeter/ent/schema/

## Decisions

- **OptionalNullable[T] instead of pointer-pointer for nullable patch fields** — Ent cannot distinguish a JSON field being absent vs. explicitly null from a *T alone; the IsSet flag makes the distinction explicit without a second raw-body parse at the adapter layer.
- **Soft-delete (SetDeletedAt) rather than hard-delete** — Subjects are referenced by usage events in ClickHouse; hard-deleting would break historical query attribution. Soft-delete preserves the row while making it invisible to active queries.

## Example: Adding a new write method to the subject adapter

```
import (
	"context"
	"fmt"

	subjectdb "github.com/openmeterio/openmeter/openmeter/ent/db/subject"
	"github.com/openmeterio/openmeter/openmeter/subject"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (a *adapter) Deactivate(ctx context.Context, id models.NamespacedID) (subject.Subject, error) {
	if err := id.Validate(); err != nil {
		return subject.Subject{}, fmt.Errorf("invalid id: %w", models.NewGenericValidationError(err))
	}
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (subject.Subject, error) {
// ...
```

<!-- archie:ai-end -->
