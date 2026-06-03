# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapters for the credit domain: balanceSnapshotRepo persists balance snapshots and grantDBADapter persists grants, implementing balance.SnapshotRepo and grant.Repo. Primary constraint: every write path must be transaction-aware via entutils.TransactingRepo.

## Patterns

**TransactingRepo wrapping on every write** — All mutating methods wrap their body with entutils.TransactingRepo or TransactingRepoWithNoValue so the ctx-bound Ent transaction is honoured. (`return entutils.TransactingRepoWithNoValue(ctx, b, func(ctx context.Context, rep *balanceSnapshotRepo) error { return rep.db.BalanceSnapshot.Update()...Exec(ctx) })`)
**TxCreator + TxUser[T] confined to transaction.go** — Each repo struct implements Tx(ctx), WithTx(ctx, *entutils.TxDriver), Self() in the single transaction.go file using db.HijackTx + db.NewTxClientFromRawConfig. (`func (e *grantDBADapter) WithTx(ctx context.Context, tx *entutils.TxDriver) grant.Repo { txClient := db.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return NewPostgresGrantRepo(txClient.Client()) }`)
**Soft-delete via clock.Now()** — Deletes set DeletedAt/VoidedAt via clock.Now()/at, never hard delete. Queries filter DeletedAtIsNil() OR DeletedAtGT(now) explicitly. (`g.db.Grant.Update().SetDeletedAt(clock.Now()).Where(db_grant.OwnerID(ownerID.ID), db_grant.Namespace(ownerID.Namespace)).Exec(ctx)`)
**db.IsNotFound mapped to domain errors** — Convert Ent not-found to the domain error (credit.GrantNotFoundError, balance.NoSavedBalanceForOwnerError). Never surface raw Ent errors. (`if db.IsNotFound(err) { return grant.Grant{}, &credit.GrantNotFoundError{GrantID: grantID.ID} }`)
**mapXxxEntity canonical conversion + UTC normalization** — Each file defines mapGrantEntity / mapBalanceSnapshotEntity that translate *db.X to the domain struct, normalizing times to time.UTC. (`func mapGrantEntity(entity *db.Grant) grant.Grant { ...CreatedAt: entity.CreatedAt.In(time.UTC)... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `balance_snapshot.go` | Implements balance.SnapshotRepo: InvalidateAfter (soft-deletes snapshots after a time), GetLatestValidAt, Save (CreateBulk). | GetLatestValidAt orders ByAt(desc) then ByUpdatedAt(desc) to deterministically pick newest among same-timestamp duplicates; keep all three methods TransactingRepo-wrapped. |
| `grant.go` | Implements grant.Repo: CreateGrant, VoidGrant (soft via SetVoidedAt), ListGrants (limit/offset or Page), ListActiveGrantsBetween, GetGrant, DeleteOwnerGrants. | CreateGrant/VoidGrant have TODO about transaction+locking. ListActiveGrantsBetween silently ignores negative-amount grants (AmountGTE(0.0)). Soft-delete filters are duplicated across predicates — keep them consistent. |
| `transaction.go` | Single file implementing TxCreator + TxUser[T] for both grantDBADapter and balanceSnapshotRepo via HijackTx (ReadOnly:false) and NewTxClientFromRawConfig. | HijackTx must always use ReadOnly:false. Do not duplicate Tx/WithTx/Self in other files. |

## Anti-Patterns

- Calling the raw db.Client inside a helper without TransactingRepo wrapping — bypasses ctx-bound transactions.
- Hard-deleting grant or balance snapshot rows — the domain is soft-delete only.
- Surfacing raw Ent not-found/constraint errors to callers instead of domain error types.
- Implementing Tx/WithTx/Self outside transaction.go.
- Accepting *entdb.Tx as a struct field or parameter instead of using entutils.TransactingRepo.

## Decisions

- **balanceSnapshotRepo and grantDBADapter each implement the full TxCreator+TxUser contract separately rather than sharing a base struct.** — The two repos have different TxUser[T] type parameters, so a shared base struct is not type-safe in Go generics; keeping them side by side in transaction.go keeps the pattern visible.

## Example: Adding a transaction-aware write method to balanceSnapshotRepo

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
