# adapter

<!-- archie:ai-start -->

> Ent-backed persistence adapter for the ledger resolvers package: it implements resolvers.CustomerAccountRepo, which manages the linking table (ledger_customer_account) mapping a customer to its ledger account IDs by AccountType. Single-file leaf adapter, feature-gated behind credits.enabled at the wiring layer.

## Patterns

**Interface-compliance assertion against parent package** — The unexported repo struct must satisfy resolvers.CustomerAccountRepo via a compile-time var _ assertion; NewRepo returns the interface type, not the concrete struct. (`var _ resolvers.CustomerAccountRepo = (*repo)(nil); func NewRepo(db *entdb.Client) resolvers.CustomerAccountRepo { return &repo{db: db} }`)
**entutils TxUser trio (Tx/WithTx/Self)** — Every repo implements Tx (HijackTx + NewTxDriver), WithTx (rebind via NewTxClientFromRawConfig), and Self, plus var _ entutils.TxUser[*repo]; this is what makes TransactingRepo helpers work. (`func (r *repo) WithTx(ctx context.Context, tx *entutils.TxDriver) *repo { return &repo{db: entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()).Client()} }`)
**Transaction-aware method bodies** — Read methods wrap bodies in entutils.TransactingRepo; mutation methods with no return value use entutils.TransactingRepoWithNoValue. Never call tx.db directly outside one of these wrappers. (`return entutils.TransactingRepoWithNoValue(ctx, r, func(ctx context.Context, tx *repo) error { _, err := tx.db.LedgerCustomerAccount.Create()... })`)
**Constraint-error to typed domain error** — On Create, detect duplicates with entdb.IsConstraintError, re-query the existing row, and return the typed resolvers.CustomerAccountAlreadyExistsError rather than leaking the raw Ent error. (`if entdb.IsConstraintError(err) { existing, _ := tx.db.LedgerCustomerAccount.Query()...Only(ctx); return &resolvers.CustomerAccountAlreadyExistsError{...} }`)
**Namespace + customer scoping on every query** — All LedgerCustomerAccount queries filter by ledgercustomeraccountdb.Namespace and CustomerID from the input/customer.CustomerID; multi-tenancy is enforced here, not by the caller. (`Where(ledgercustomeraccountdb.Namespace(customerID.Namespace), ledgercustomeraccountdb.CustomerID(customerID.ID))`)
**AccountType-keyed result maps** — GetCustomerAccountIDs returns map[ledger.AccountType]string built by ranging entities; AccountType is the map key so each customer holds at most one account per type. (`result := make(map[ledger.AccountType]string, len(entities)); for _, e := range entities { result[e.AccountType] = e.AccountID }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `repo.go` | Entire adapter: defines repo, NewRepo, the Tx/WithTx/Self trio, and the two CustomerAccountRepo methods CreateCustomerAccount and GetCustomerAccountIDs. | Adding a method that touches tx.db without a TransactingRepo wrapper breaks tx propagation; returning the raw Ent constraint error instead of CustomerAccountAlreadyExistsError loses the typed-error contract; forgetting the namespace filter leaks rows across tenants. |

## Anti-Patterns

- Returning the concrete *repo from NewRepo instead of the resolvers.CustomerAccountRepo interface.
- Calling r.db / tx.db.LedgerCustomerAccount outside entutils.TransactingRepo or TransactingRepoWithNoValue (loses ctx-carried transaction).
- Surfacing raw entdb constraint errors instead of mapping them to resolvers.CustomerAccountAlreadyExistsError.
- Querying LedgerCustomerAccount without scoping by Namespace and CustomerID.
- Defining domain input/output types here — CreateCustomerAccountInput and the repo interface live in the parent resolvers package; this folder only implements.

## Decisions

- **Adapter is a thin single-file leaf with no service logic.** — Business logic lives in the resolvers service layer; this folder only translates the CustomerAccountRepo interface into Ent calls against the ledger_customer_account linking table.
- **Duplicate creation is treated idempotently via constraint detection + re-query.** — Account backfill and concurrent provisioning can race; returning a typed AlreadyExists error carrying the existing AccountID lets callers reconcile without a separate lookup.

## Example: Read method scoped to a customer and rebound to the ctx transaction

```
func (r *repo) GetCustomerAccountIDs(ctx context.Context, customerID customer.CustomerID) (map[ledger.AccountType]string, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, tx *repo) (map[ledger.AccountType]string, error) {
		entities, err := tx.db.LedgerCustomerAccount.Query().
			Where(
				ledgercustomeraccountdb.Namespace(customerID.Namespace),
				ledgercustomeraccountdb.CustomerID(customerID.ID),
			).All(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get ledger customer accounts: %w", err)
		}
		result := make(map[ledger.AccountType]string, len(entities))
		for _, entity := range entities {
			result[entity.AccountType] = entity.AccountID
		}
		return result, nil
// ...
```

<!-- archie:ai-end -->
