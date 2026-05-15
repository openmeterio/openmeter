# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL implementation of customer.Adapter — the sole layer that touches the database for the customer domain. All reads and writes are wrapped in entutils.TransactingRepo so every method honors the ctx-bound Ent transaction.

## Patterns

**TransactingRepo on every method body** — Every exported method wraps its logic in entutils.TransactingRepo (for value-returning methods) or entutils.TransactingRepoWithNoValue (for void methods). Direct repo.db.* calls occur only inside the callback, never on the outer adapter receiver. (`return entutils.TransactingRepo(ctx, a, func(ctx context.Context, repo *adapter) (*customer.Customer, error) { return toDomain(repo.db.Customer.Query()...) })`)
**TxCreator + TxUser triad** — adapter implements Tx() via db.HijackTx+entutils.NewTxDriver, WithTx() via entdb.NewTxClientFromRawConfig, and Self() returning itself. All three are required for TransactingRepo to rebind to caller-supplied transactions. (`func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter { txClient := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return &adapter{db: txClient.Client(), logger: a.logger} }`)
**Input validation before DB access** — Every method calls input.Validate() and wraps errors in models.NewGenericValidationError before any DB query is issued. (`if err := input.Validate(); err != nil { return nil, models.NewGenericValidationError(fmt.Errorf("error creating customer: %w", err)) }`)
**Soft-delete pattern** — Deletion sets DeletedAt (clock.Now().UTC()) on Customer, CustomerSubjects, AppCustomer, AppStripeCustomer, and AppCustomInvoicingCustomer rows. All cascade updates filter by DeletedAtIsNil() to preserve timestamps from earlier soft-deletes. (`repo.db.Customer.Update().Where(customerdb.DeletedAtIsNil()).SetDeletedAt(deletedAt).Save(ctx)`)
**Compile-time interface assertion** — adapter.go declares var _ customer.Adapter = (*adapter)(nil) immediately after the struct; any missing interface method causes a build failure. (`var _ customer.Adapter = (*adapter)(nil)`)
**Entity mapping separated in entitymapping.go** — All DB-to-domain conversions (CustomerFromDBEntity, subjectKeysFromDBEntity, resolveActiveSubscriptionIDs) live in entitymapping.go. Edges must be loaded before calling these functions; db.IsNotLoaded(err) must be checked. (`subjectEntities, err := customerEntity.Edges.SubjectsOrErr(); if db.IsNotLoaded(err) { return nil, errors.New("subjects must be loaded") }`)
**Domain error wrapping** — Ent constraint errors are converted to customer.NewKeyConflictError or models.NewGenericConflictError; not-found results from zero rows become models.NewGenericNotFoundError. Raw ent errors must never bubble up to callers. (`if entdb.IsConstraintError(err) { return nil, customer.NewKeyConflictError(input.Namespace, *lo.CoalesceOrEmpty(input.Key)) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Struct definition, Config validation, Tx/WithTx/Self triad, constructor, and compile-time interface assertion. | When adding fields to the adapter struct, clone them in WithTx; forgetting breaks ctx-propagated transactions for new fields. |
| `customer.go` | All adapter method implementations: ListCustomers, CreateCustomer, DeleteCustomer, GetCustomer, GetCustomerByUsageAttribution, UpdateCustomer, ListCustomerUsageAttributions. | Never call repo.db.* on the outer adapter receiver — only inside a TransactingRepo callback. Always call WithSubjects() before CustomerFromDBEntity or SubjectsOrErr panics. |
| `entitymapping.go` | Pure DB-to-domain conversion functions with no DB calls. CustomerFromDBEntity, subjectKeysFromDBEntity, resolveActiveSubscriptionIDs. | Accessing Edges fields without loading them (WithSubjects, WithActiveSubscriptions) causes db.IsNotLoaded errors; check them explicitly. |

## Anti-Patterns

- Calling a.db.* directly on the outer adapter receiver instead of inside a TransactingRepo callback — bypasses ctx-bound Ent transaction.
- Skipping input.Validate() before DB queries — allows invalid inputs to reach the database.
- Hard-deleting customer or customer_subjects rows — the domain uses soft-delete via DeletedAt everywhere.
- Adding DB query logic inside entitymapping.go — it must remain a pure conversion layer.
- Returning raw Ent errors without wrapping in models.GenericNotFoundError / GenericValidationError — breaks the HTTP error-mapping chain.

## Decisions

- **TransactingRepo wraps every method body rather than relying on callers to pass a transaction client.** — Ent transactions propagate implicitly via ctx; helpers that use the raw *entdb.Client bypass the ctx-bound transaction and cause partial writes under concurrency.
- **Soft-delete instead of hard-delete for customers and subjects.** — Billing and subscription entities reference customers; hard deletes orphan related records and break audit trails.
- **Entity mapping isolated in entitymapping.go with no DB access.** — Pure conversion functions are independently testable and prevent accidental re-querying during domain mapping.

## Example: Add a new adapter method that reads and writes within the same transaction

```
import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (a *adapter) ArchiveCustomer(ctx context.Context, input customer.ArchiveCustomerInput) (*customer.Customer, error) {
	if err := input.Validate(); err != nil {
		return nil, models.NewGenericValidationError(fmt.Errorf("archive customer: %w", err))
	}
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, repo *adapter) (*customer.Customer, error) {
		// use repo.db.* here, never a.db.*
// ...
```

<!-- archie:ai-end -->
