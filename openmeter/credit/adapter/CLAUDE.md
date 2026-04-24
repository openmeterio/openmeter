# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapters for the credit domain: persists balance snapshots (balanceSnapshotRepo) and grants (grantDBADapter), implementing the balance.SnapshotRepo and grant.Repo interfaces respectively. Primary constraint: every write path must be transaction-aware via entutils.TransactingRepo.

## Patterns

**TransactingRepo wrapping on every write** — All mutating methods on both balanceSnapshotRepo and grantDBADapter must wrap their body with entutils.TransactingRepo or entutils.TransactingRepoWithNoValue so the ctx-bound Ent transaction is honoured. (`return entutils.TransactingRepoWithNoValue(ctx, b, func(ctx context.Context, rep *balanceSnapshotRepo) error { return rep.db.BalanceSnapshot.Update()... })`)
**TxCreator + TxUser[T] implementation in transaction.go** — Each repo struct implements Tx(ctx) (context.Context, transaction.Driver, error), WithTx(ctx, *entutils.TxDriver) T, and Self() T in a separate transaction.go file. This is the standard entutils contract. (`func (e *grantDBADapter) WithTx(ctx context.Context, tx *entutils.TxDriver) grant.Repo { txClient := db.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return NewPostgresGrantRepo(txClient.Client()) }`)
**Soft-delete via clock.Now() timestamps** — Deletes set DeletedAt via clock.Now() (not hard deletes). Queries must filter db_grant.DeletedAtIsNil() or db_grant.DeletedAtGT(now) explicitly. (`g.db.Grant.Update().SetDeletedAt(clock.Now()).Where(...).Exec(ctx)`)
**db.IsNotFound error mapping to domain errors** — When an Ent query returns IsNotFound, the adapter converts it to the domain error type (e.g. credit.GrantNotFoundError, balance.NoSavedBalanceForOwnerError). Never surface raw Ent not-found errors to callers. (`if db.IsNotFound(err) { return grant.Grant{}, &credit.GrantNotFoundError{GrantID: grantID.ID} }`)
**mapXxxEntity helper converts Ent row to domain type** — Each file defines a mapXxxEntity function that translates the generated *db.X to the corresponding domain struct, including time.UTC normalization. This is the canonical mapping layer. (`func mapGrantEntity(entity *db.Grant) grant.Grant { ... entity.CreatedAt.In(time.UTC) ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `balance_snapshot.go` | Implements balance.SnapshotRepo: InvalidateAfter (soft-deletes snapshots after a time), GetLatestValidAt (returns most-recent non-deleted snapshot), Save (bulk-creates snapshots). | All three methods use entutils.TransactingRepo* — if you add a new method, do the same. GetLatestValidAt orders by At DESC then UpdatedAt DESC to handle same-timestamp duplicates. |
| `grant.go` | Implements grant.Repo CRUD: CreateGrant, VoidGrant (soft-void via VoidedAt), ListGrants (supports both limit/offset and page-based pagination), ListActiveGrantsBetween, GetGrant, DeleteOwnerGrants. | CreateGrant has a TODO about transactions/locking — callers should wrap in a tx. ListGrants duplicates the soft-delete filter in multiple places; maintain consistency. NegativeAmount grants are silently ignored in ListActiveGrantsBetween. |
| `transaction.go` | Single file implementing TxCreator and TxUser[T] for both grantDBADapter and balanceSnapshotRepo using db.HijackTx and db.NewTxClientFromRawConfig. | HijackTx must always use ReadOnly:false. Do not duplicate this pattern inside other files. |

## Anti-Patterns

- Using the raw db.Client inside a helper function without TransactingRepo wrapping — this bypasses ctx-bound transactions.
- Hard-deleting grant or balance snapshot rows — the domain uses soft deletes exclusively.
- Surfacing raw Ent not-found or constraint errors to callers — always convert to domain error types.
- Implementing Tx/WithTx/Self outside of transaction.go — keep the transactional plumbing in one file per adapter.

## Decisions

- **balanceSnapshotRepo and grantDBADapter both implement the full TxCreator+TxUser contract via transaction.go rather than sharing a single base struct.** — The two repos have different generic type parameters for TxUser[T], so sharing a base struct is not type-safe in Go generics. Separate implementations in one file keeps the pattern visible and consistent.

## Example: Adding a new write method to balanceSnapshotRepo

```
import "github.com/openmeterio/openmeter/pkg/framework/entutils"

func (b *balanceSnapshotRepo) MyNewWrite(ctx context.Context, owner models.NamespacedID) error {
	return entutils.TransactingRepoWithNoValue(ctx, b, func(ctx context.Context, rep *balanceSnapshotRepo) error {
		return rep.db.BalanceSnapshot.Update().
			Where(db_balancesnapshot.OwnerID(owner.ID), db_balancesnapshot.Namespace(owner.Namespace)).
			Exec(ctx)
	})
}
```

<!-- archie:ai-end -->
