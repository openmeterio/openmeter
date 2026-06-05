# adapter

<!-- archie:ai-start -->

> Ent/Postgres persistence layer for the credit domain: implements grant.Repo (grant CRUD/list/void) and balance.SnapshotRepo (balance snapshot save/invalidate/get-latest). All access is transaction-aware via entutils.

## Patterns

**Transacting repo wrapper** — Read/write methods wrap their body in entutils.TransactingRepo / TransactingRepoWithNoValue so they rebind to a tx already in ctx; the repo also implements Tx/WithTx/Self (TxCreator + TxUser). (`func (b *balanceSnapshotRepo) Save(ctx, owner, balances) error { return entutils.TransactingRepoWithNoValue(ctx, b, func(ctx, rep *balanceSnapshotRepo) error { ... }) }`)
**Entity-to-domain mapping helper** — Each repo has a package-private mapXxxEntity(*db.X) domain.Y that converts ent rows to domain models, normalizing times to UTC via .In(time.UTC) and convert.SafeToUTC. (`func mapGrantEntity(entity *db.Grant) grant.Grant { ... CreatedAt: entity.CreatedAt.In(time.UTC) ... }`)
**Not-found maps to domain error** — On db.IsNotFound(err) return a typed domain error (credit.GrantNotFoundError, balance.NoSavedBalanceForOwnerError), never the raw ent error. (`if db.IsNotFound(err) { return grant.Grant{}, &credit.GrantNotFoundError{GrantID: grantID.ID} }`)
**Soft-delete via DeletedAt/VoidedAt** — Deletes and voids are updates that set DeletedAt/VoidedAt with clock.Now(); list queries filter with DeletedAtIsNil()/DeletedAtGT(now) rather than physical deletes. (`g.db.Grant.Update().SetDeletedAt(clock.Now()).Where(db_grant.OwnerID(...)).Exec(ctx)`)
**Dual pagination (page vs limit/offset)** — ListGrants branches on params.Page.IsZero(): zero page uses Limit/Offset and returns Items only; otherwise query.Paginate(ctx, page) fills TotalCount. (`if params.Page.IsZero() { query = query.Limit(...).Offset(...) } else { paged, _ := query.Paginate(ctx, params.Page) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `grant.go` | grantDBADapter implements grant.Repo: CreateGrant, VoidGrant, DeleteOwnerGrants, ListGrants, ListActiveGrantsBetween, GetGrant + mapGrantEntity. | CreateGrant/VoidGrant have TODO comments about missing tx+locking; negative-amount grants are silently filtered (AmountGTE(0)) in ListActiveGrantsBetween; VoidedAt is truncated to the minute on read for legacy consistency. |
| `balance_snapshot.go` | balanceSnapshotRepo implements balance.SnapshotRepo: InvalidateAfter, GetLatestValidAt, Save (CreateBulk) + mapBalanceSnapshotEntity. | GetLatestValidAt orders by At desc then UpdatedAt desc to pick newest snapshot for the same time; returned snapshot may have nil Usage which the balance service backfills. |
| `transaction.go` | Implements entutils.TxCreator/TxUser (Tx/WithTx/Self) for both repos via db.HijackTx and NewTxClientFromRawConfig. | Both repos use ReadOnly:false; WithTx rebuilds a fresh repo from the tx config — do not capture the non-tx *db.Client inside transacting closures. |

## Anti-Patterns

- Accessing rep.db directly without wrapping in entutils.TransactingRepo(WithNoValue) — breaks tx propagation from ctx.
- Returning raw ent errors instead of credit.GrantNotFoundError / balance.NoSavedBalanceForOwnerError.
- Physically deleting rows instead of setting DeletedAt/VoidedAt.
- Building domain times without .In(time.UTC) normalization in map* helpers.

## Decisions

- **Snapshots are append-only and invalidated (not updated) via InvalidateAfter setting DeletedAt.** — Balance is recomputed from the latest valid snapshot + streaming usage, so historical snapshots must remain immutable and orderable.

## Example: Transaction-aware read returning a typed not-found error

```
func (g *grantDBADapter) GetGrant(ctx context.Context, grantID models.NamespacedID) (grant.Grant, error) {
	ent, err := g.db.Grant.Query().Where(db_grant.ID(grantID.ID), db_grant.Namespace(grantID.Namespace)).Only(ctx)
	if err != nil {
		if db.IsNotFound(err) {
			return grant.Grant{}, &credit.GrantNotFoundError{GrantID: grantID.ID}
		}
		return grant.Grant{}, err
	}
	return mapGrantEntity(ent), nil
}
```

<!-- archie:ai-end -->
