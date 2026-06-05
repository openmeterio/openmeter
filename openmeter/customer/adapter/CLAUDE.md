# adapter

<!-- archie:ai-start -->

> Ent persistence layer for the customer domain. Implements customer.Adapter (CRUD, listing, usage-attribution, soft-delete cascades) against *entdb.Client, plus DB-entity-to-domain mapping.

## Patterns

**Transaction-wrapping repo helpers** — Every read/write method body runs inside entutils.TransactingRepo (value-returning) or entutils.TransactingRepoWithNoValue (void), so the body rebinds to the tx carried in ctx via the repo *adapter. (`return entutils.TransactingRepo(ctx, a, func(ctx context.Context, repo *adapter) (*customer.Customer, error) { ... repo.db.Customer.Query()... })`)
**TxCreator triplet on adapter** — The adapter implements Tx/WithTx/Self: Tx hijacks via a.db.HijackTx with &sql.TxOptions{ReadOnly:false}, WithTx rebuilds via entdb.NewTxClientFromRawConfig, Self returns the receiver. Required for entutils helpers to work. (`func (a *adapter) WithTx(ctx, tx) *adapter { txClient := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return &adapter{db: txClient.Client(), logger: a.logger} }`)
**Validate input then wrap as GenericValidationError** — Each method first calls input.Validate() and on failure returns models.NewGenericValidationError(...) before any DB access. (`if err := input.Validate(); err != nil { return ..., models.NewGenericValidationError(err) }`)
**Soft-delete with deleted_at + namespace filters** — Deletes are UPDATEs setting deleted_at=clock.Now().UTC(), always scoped by namespace and DeletedAtIsNil(); reads default to excluding rows where DeletedAt < now via Or(DeletedAtIsNil, DeletedAtGTE(now)). (`repo.db.Customer.Update().Where(customerdb.ID(input.ID), customerdb.Namespace(input.Namespace), customerdb.DeletedAtIsNil()).SetDeletedAt(deletedAt)`)
**Manual delete cascade across app tables** — DeleteCustomer soft-deletes the customer then each association (CustomerSubjects, AppCustomer, AppStripeCustomer, AppCustomInvoicingCustomer) in the same tx, each filtered by DeletedAtIsNil so already-deleted children keep their original timestamp. (`repo.db.AppStripeCustomer.Update().Where(appstripecustomerdb.CustomerID(input.ID), ...DeletedAtIsNil()).SetDeletedAt(deletedAt).Exec(ctx)`)
**Edge-loaded mapping requires explicit loads** — CustomerFromDBEntity/subjectKeysFromDBEntity call Edges.SubjectsOrErr / SubscriptionOrErr and return an error if db.IsNotLoaded; callers must apply WithSubjects(now) and (for expands) WithActiveSubscriptions(now) on the query first. (`query := repo.db.Customer.Query(); query = WithSubjects(query, now); if slices.Contains(input.Expands, customer.ExpandSubscriptions) { query = WithActiveSubscriptions(query, now) }`)
**filter.ApplyToQuery for string filters** — List filters apply through pkg/filter (filter.ApplyToQuery / filter.SelectPredicate) against ent field constants rather than hand-built predicates. (`query = filter.ApplyToQuery(query, input.Key, customerdb.FieldKey)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Config{Client,Logger}+Validate, New() returning customer.Adapter, and the Tx/WithTx/Self TxCreator triplet. | Both Client and Logger are required by Validate; Tx uses ReadOnly:false — do not pass nil logger. |
| `customer.go` | All adapter methods: ListCustomers, ListCustomerUsageAttributions, CreateCustomer, DeleteCustomer, GetCustomer, etc. | CreateCustomer pre-checks key-vs-id and key-vs-subject overlap and returns NewKeyConflictError/NewSubjectKeyConflictError on entdb.IsConstraintError; subjects are bulk-created separately (AddSubjects produces an invalid query). |
| `entitymapping.go` | CustomerFromDBEntity, resolveActiveSubscriptionIDs, subjectKeysFromDBEntity — DB→domain mapping. | Returns error when edges not loaded; UsageAttribution only set when subjectKeys non-empty; subject keys are slices.Sort-ed for stable order; ActiveSubscriptionIDs wrapped in mo.Some only when expand present. |
| `customer_test.go` | Adapter integration tests via testutils.InitPostgresDB; seed() builds the full cascade chain; freezeTime truncates to microsecond. | Postgres has microsecond precision — freeze/compare times via Truncate(time.Microsecond); tests assert namespace isolation and preservation of pre-deleted children. |

## Anti-Patterns

- Accessing repo.db directly outside a TransactingRepo/TransactingRepoWithNoValue closure (breaks tx propagation)
- Reading customers without WithSubjects(now) — mapping will fail with 'subjects must be loaded for customer'
- Hard-deleting rows or forgetting the namespace + DeletedAtIsNil filters on cascade updates
- Using customer.AddSubjects to attach subjects in Create (generates an invalid query); use CustomerSubjects.CreateBulk
- Returning raw errors for invalid input instead of models.NewGenericValidationError / typed conflict errors

## Decisions

- **Soft delete via deleted_at with explicit per-table cascade rather than DB cascades** — Preserves audit history and lets already-deleted children keep their original timestamp by filtering DeletedAtIsNil.
- **Subjects inserted with CustomerSubjects.CreateBulk instead of ent edge AddSubjects** — ent edge AddSubjects produces an invalid SQL query; the bulk path issues the same number/shape of queries — devex-only difference.

## Example: Adapter method wrapping its body in a transacting repo closure

```
func (a *adapter) DeleteCustomer(ctx context.Context, input customer.DeleteCustomerInput) error {
	if err := input.Validate(); err != nil {
		return models.NewGenericValidationError(fmt.Errorf("error deleting customer: %w", err))
	}
	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, repo *adapter) error {
		deletedAt := clock.Now().UTC()
		rows, err := repo.db.Customer.Update().
			Where(customerdb.ID(input.ID), customerdb.Namespace(input.Namespace), customerdb.DeletedAtIsNil()).
			SetDeletedAt(deletedAt).Save(ctx)
		if err != nil { return fmt.Errorf("failed to delete customer: %w", err) }
		if rows == 0 { return models.NewGenericNotFoundError(fmt.Errorf("customer with id %s not found in %s namespace", input.ID, input.Namespace)) }
		return nil
	})
}
```

<!-- archie:ai-end -->
