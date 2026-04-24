# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL implementation of customer.Adapter — the only layer that touches the database for the customer domain. All reads and writes go through entutils.TransactingRepo so every operation honors the ctx-bound transaction.

## Patterns

**TransactingRepo wrapper on every method** — Every exported adapter method body is wrapped in entutils.TransactingRepo (returns value) or entutils.TransactingRepoWithNoValue (void). Direct repo.db.* calls happen only inside the callback, never on the outer adapter receiver. (`return entutils.TransactingRepo(ctx, a, func(ctx context.Context, repo *adapter) (*customer.Customer, error) { ... repo.db.Customer.Query()... })`)
**Compile-time interface assertion** — adapter.go declares var _ customer.Adapter = (*adapter)(nil) immediately after the struct — adding a method to the interface that is not implemented causes a build failure here. (`var _ customer.Adapter = (*adapter)(nil)`)
**Config-validated constructor** — New(Config) validates Config.Client != nil and Config.Logger != nil before constructing the adapter. Never call New() with zero-value Config. (`func New(config Config) (customer.Adapter, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)
**WithTx / Self / Tx triad for transaction rebinding** — adapter implements entutils.TxCreator via Tx(), WithTx(), and Self(). These are required by TransactingRepo to rebind the adapter to an active transaction from context. (`func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter { txClient := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return &adapter{db: txClient.Client(), logger: a.logger} }`)
**Input validation before DB access** — Every adapter method calls input.Validate() and wraps errors in models.NewGenericValidationError before touching the database. (`if err := input.Validate(); err != nil { return nil, models.NewGenericValidationError(fmt.Errorf("error creating customer: %w", err)) }`)
**Soft-delete pattern** — Deletion sets DeletedAt timestamp (clock.Now().UTC()) on both Customer and CustomerSubjects rows; IsNil predicates exclude deleted rows in queries by default unless IncludeDeleted is true. (`repo.db.Customer.Update().Where(customerdb.DeletedAtIsNil()).SetDeletedAt(deletedAt).Save(ctx)`)
**Entity mapping in entitymapping.go** — DB-to-domain conversion (CustomerFromDBEntity) lives exclusively in entitymapping.go; Edges must be loaded (e.g. WithSubjects) before calling it — calling Edges.SubjectsOrErr() on an unloaded edge returns db.IsNotLoaded(err). (`subjectEntities, err := customerEntity.Edges.SubjectsOrErr(); if db.IsNotLoaded(err) { return nil, errors.New("subjects must be loaded") }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Struct definition, constructor, Config validation, Tx/WithTx/Self transaction triad, compile-time interface assertion. | Adding fields to adapter without updating WithTx clone; forgetting Tx() implementation breaks TransactingRepo at runtime. |
| `customer.go` | All adapter method implementations: ListCustomers, CreateCustomer, DeleteCustomer, GetCustomer, GetCustomerByUsageAttribution, UpdateCustomer, ListCustomerUsageAttributions. | Calling repo.db.* outside a TransactingRepo callback; forgetting to call WithSubjects() before CustomerFromDBEntity causes IsNotLoaded panics. |
| `entitymapping.go` | Pure conversion functions: CustomerFromDBEntity, subjectKeysFromDBEntity, resolveActiveSubscriptionIDs. No DB calls. | Accessing Edges fields without loading them via WithSubjects/WithActiveSubscriptions in the query builder — db.IsNotLoaded must be checked. |

## Anti-Patterns

- Calling repo.db.* directly on the outer adapter receiver instead of inside a TransactingRepo callback — bypasses ctx-bound transactions.
- Skipping input.Validate() before DB queries — allows invalid inputs to reach the database layer.
- Hard-deleting customer or customer_subjects rows — the domain uses soft-delete via DeletedAt everywhere.
- Adding DB query logic inside entitymapping.go — it must stay a pure conversion layer.
- Returning raw ent errors without wrapping in models.GenericNotFoundError / GenericValidationError / GenericConflictError — breaks the HTTP error-mapping chain.

## Decisions

- **TransactingRepo wraps every method body rather than relying on callers to pass tx clients.** — Ent transactions propagate implicitly via ctx; helpers that use the raw *entdb.Client bypass the ctx-bound transaction and cause partial writes under concurrency.
- **Soft-delete instead of hard-delete for customers and subjects.** — Billing and subscription entities reference customers; hard deletes would orphan related records and break audit trails.
- **Entity mapping separated into entitymapping.go with no DB access.** — Pure functions are testable in isolation; keeping conversion logic separate prevents accidentally re-querying the DB during mapping.

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
