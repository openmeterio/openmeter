# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing the resolvers.CustomerAccountRepo interface for mapping ledger account IDs to customers. Single file (repo.go) that persists ledger_customer_account rows and is the only DB-write path for the resolver layer.

## Patterns

**TransactingRepo wrapping for every DB method** — All methods that read or write DB must use entutils.TransactingRepo (returns value) or entutils.TransactingRepoWithNoValue (void) so the method rebinds to any transaction already carried in ctx instead of using the raw client directly. (`return entutils.TransactingRepoWithNoValue(ctx, r, func(ctx context.Context, tx *repo) error { ... tx.db.LedgerCustomerAccount.Create()... })`)
**Interface assertion at declaration** — Compile-time interface compliance is verified with var _ resolvers.CustomerAccountRepo = (*repo)(nil) and var _ entutils.TxUser[*repo] = (*repo)(nil) at the top of the file. (`var _ resolvers.CustomerAccountRepo = (*repo)(nil)`)
**TxUser triple: WithTx + Self + Tx** — The struct must implement all three entutils.TxUser[*repo] methods: Tx (hijacks a new tx via db.HijackTx), WithTx (rebinds to an existing tx driver), and Self (returns self) — required for TransactingRepo machinery to work. (`func (r *repo) WithTx(ctx context.Context, tx *entutils.TxDriver) *repo { return &repo{db: entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()).Client()} }`)
**Constraint error → domain error conversion** — Ent constraint errors on unique indexes must be caught with entdb.IsConstraintError and converted to typed domain errors (e.g. resolvers.CustomerAccountAlreadyExistsError) rather than surfacing raw DB errors. (`if entdb.IsConstraintError(err) { existing, _ := tx.db.LedgerCustomerAccount.Query()...; return &resolvers.CustomerAccountAlreadyExistsError{...} }`)
**Constructor returns interface, not concrete type** — NewRepo returns the resolvers.CustomerAccountRepo interface, not *repo, so callers depend on the abstraction. (`func NewRepo(db *entdb.Client) resolvers.CustomerAccountRepo { return &repo{db: db} }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `repo.go` | Sole file in the package. Implements CustomerAccountRepo with CreateCustomerAccount and GetCustomerAccountIDs backed by LedgerCustomerAccount Ent entity. | Any DB access outside TransactingRepo/TransactingRepoWithNoValue will bypass ctx-carried transactions and produce partial writes. Constraint errors not caught with entdb.IsConstraintError will leak raw DB errors to callers. |

## Anti-Patterns

- Calling tx.db.LedgerCustomerAccount... directly inside a method body without wrapping in TransactingRepo/TransactingRepoWithNoValue
- Returning *repo from NewRepo instead of the resolvers.CustomerAccountRepo interface
- Adding business logic (account type routing, balance calculation) to this adapter — it belongs in the service layer above
- Using context.Background() or context.TODO() instead of propagating the caller's ctx
- Removing the var _ interface assertion lines — they are the only compile-time proof of interface compliance

## Decisions

- **Package contains only a single repo.go with no sub-files** — The adapter's surface is tiny (two methods); splitting into multiple files would add navigation overhead for no benefit.
- **Tx is implemented via db.HijackTx rather than db.BeginTx** — HijackTx returns both a TxDriver and the raw pgx config needed for NewTxClientFromRawConfig, allowing the standard entutils transaction rebinding pattern used across all adapters.

## Example: Adding a new write method to the repo that must honor the ambient transaction

```
import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/ledger/resolvers"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

func (r *repo) DeleteCustomerAccount(ctx context.Context, input resolvers.DeleteCustomerAccountInput) error {
	return entutils.TransactingRepoWithNoValue(ctx, r, func(ctx context.Context, tx *repo) error {
		_, err := tx.db.LedgerCustomerAccount.Delete().
			Where( /* filters */ ).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to delete ledger customer account: %w", err)
// ...
```

<!-- archie:ai-end -->
