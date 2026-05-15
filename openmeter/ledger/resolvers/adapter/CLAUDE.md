# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing resolvers.CustomerAccountRepo — the sole DB-write path for mapping ledger account IDs to customers via ledger_customer_account rows. Every method must rebind to any ctx-carried transaction via entutils.TransactingRepo.

## Patterns

**TransactingRepo wrapping on every DB method** — All methods reading or writing DB must wrap their body in entutils.TransactingRepo (value return) or TransactingRepoWithNoValue (void). Never call tx.db.LedgerCustomerAccount... directly in a method body. (`return entutils.TransactingRepoWithNoValue(ctx, r, func(ctx context.Context, tx *repo) error { _, err := tx.db.LedgerCustomerAccount.Create()...; return err })`)
**TxUser triple: Tx + WithTx + Self** — The repo struct must implement all three entutils.TxUser[*repo] methods: Tx (hijacks via db.HijackTx), WithTx (rebinds via entdb.NewTxClientFromRawConfig), Self (returns self). All three are required for TransactingRepo machinery. (`func (r *repo) WithTx(ctx context.Context, tx *entutils.TxDriver) *repo { return &repo{db: entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()).Client()} }`)
**Compile-time interface assertions** — Declare var _ resolvers.CustomerAccountRepo = (*repo)(nil) and var _ entutils.TxUser[*repo] = (*repo)(nil) at package level to catch interface drift at compile time. (`var _ resolvers.CustomerAccountRepo = (*repo)(nil)`)
**Constraint error → domain error conversion** — Catch Ent unique-index violations with entdb.IsConstraintError and convert to typed domain errors (resolvers.CustomerAccountAlreadyExistsError). Never surface raw DB errors to callers. (`if entdb.IsConstraintError(err) { existing, _ := tx.db.LedgerCustomerAccount.Query()...; return &resolvers.CustomerAccountAlreadyExistsError{...} }`)
**Constructor returns interface, not concrete type** — NewRepo returns resolvers.CustomerAccountRepo, not *repo, so callers depend on the abstraction and the concrete type stays unexported. (`func NewRepo(db *entdb.Client) resolvers.CustomerAccountRepo { return &repo{db: db} }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `repo.go` | Sole file in the package. Implements CreateCustomerAccount and GetCustomerAccountIDs backed by the LedgerCustomerAccount Ent entity. Includes all TxUser methods and compile-time assertions. | Any DB access outside TransactingRepo/TransactingRepoWithNoValue bypasses ctx-carried transactions and produces partial writes. Constraint errors not caught with entdb.IsConstraintError leak raw DB errors. Removing var _ assertion lines eliminates compile-time interface proof. |

## Anti-Patterns

- Calling tx.db.LedgerCustomerAccount... directly in a method body without wrapping in TransactingRepo/TransactingRepoWithNoValue
- Returning *repo from NewRepo instead of resolvers.CustomerAccountRepo interface
- Adding business logic (account type routing, balance calculation) — it belongs in the service layer above
- Using context.Background() or context.TODO() instead of propagating the caller's ctx
- Removing the var _ interface assertion lines — they are the only compile-time proof of interface compliance

## Decisions

- **Single repo.go file with no sub-files** — The adapter surface is tiny (two methods); splitting would add navigation overhead with no benefit.
- **Tx implemented via db.HijackTx rather than db.BeginTx** — HijackTx returns both a TxDriver and the raw pgx config needed for NewTxClientFromRawConfig, enabling the standard entutils transaction rebinding pattern used across all adapters.

## Example: Adding a new write method that must honor the ambient ctx-carried transaction

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
