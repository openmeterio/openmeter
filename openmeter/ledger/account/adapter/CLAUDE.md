# adapter

<!-- archie:ai-start -->

> Ent-backed persistence layer for ledger accounts, sub-accounts, and their routing rows. Implements the ledgeraccount.Repo interface; all reads/writes must stay transaction-aware via entutils so they rebind to a tx already carried in ctx.

## Patterns

**TransactingRepo wrapping** — Every public repo method body is wrapped in entutils.TransactingRepo so it rebinds to the ctx-carried tx instead of r.db directly. (`func (r *repo) CreateAccount(ctx, input) (...) { return entutils.TransactingRepo(ctx, r, func(ctx, tx *repo) (...) { entity, err := tx.db.LedgerAccount.Create()... }) }`)
**TxUser implementation** — repo satisfies entutils.TxUser[*repo] via Tx/WithTx/Self; WithTx rebuilds db from raw tx config with entdb.NewTxClientFromRawConfig. (`var _ entutils.TxUser[*repo] = (*repo)(nil); func (r *repo) WithTx(ctx, tx) *repo { return &repo{db: entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()).Client()} }`)
**Map* projector functions** — DB entities are converted to domain Data structs via exported MapAccountData / MapSubAccountData; never return raw db.* entities. (`func MapAccountData(entity *db.LedgerAccount) (*ledgeraccount.AccountData, error)`)
**Idempotent ensure via OnConflict ResolveWithIgnore** — EnsureSubAccount and resolveOrCreateRoute upsert by unique constraint columns then re-query, so duplicate routes/sub-accounts resolve to the existing row. (`OnConflict(sql.ConflictColumns(FieldNamespace, FieldAccountID, FieldRouteID), sql.ResolveWithIgnore()).Exec(ctx)`)
**Route normalization before persistence/query** — Routes are normalized (input.Route.Normalize()) and keyed via ledger.BuildRoutingKey before insert and in list-filter predicate building, so cost-basis canonicalization (0.70 == 0.7) is enforced. (`normalizedRoute, _ := input.Route.Normalize(); routeKey, _ := ledger.BuildRoutingKey(normalizedRoute)`)
**Eager-load required edges then validate** — Sub-account queries use WithRoute().WithAccount(); MapSubAccountData errors if either edge is nil. (`if entity.Edges.Account == nil { return ..., fmt.Errorf("account edge is required") }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `repo.go` | repo struct, NewRepo constructor, and the TxUser plumbing (Tx/WithTx/Self) plus Repo interface assertion. | WithTx must rebuild db from the raw tx config; do not return r unchanged or transactions leak across the original client. |
| `account.go` | Account CRUD: CreateAccount, GetAccountByID, ListAccounts, and MapAccountData projector. | Wrap all bodies in TransactingRepo; ListAccounts only filters AccountTypeIn when input.AccountTypes is non-empty. |
| `subaccount.go` | Sub-account ensure/get/list, the private resolveOrCreateRoute upsert, and MapSubAccountData. | resolveOrCreateRoute is the only place routes are created; route filtering in ListSubAccounts uses mo.Option presence semantics (IsPresent + nil-inner => *IsNil predicate). |
| `repo_test.go` | Integration tests against a real Postgres (testutils.InitPostgresDB + migrate.OMMigrationsConfig); exercises route uniqueness and cost-basis canonicalization. | Requires Postgres; tests assert entdb.IsConstraintError on duplicate routing keys and that canonicalized cost basis collides. |

## Anti-Patterns

- Accessing r.db directly inside a method body instead of the tx-bound client from TransactingRepo.
- Returning raw db.LedgerAccount / db.LedgerSubAccount entities instead of mapped Data structs.
- Building route rows without Normalize() + BuildRoutingKey, breaking cost-basis canonical uniqueness.
- Mapping a sub-account without WithRoute()/WithAccount() loaded, hitting the nil-edge error.
- Implementing ensure as plain Create instead of OnConflict+ResolveWithIgnore, losing idempotency.

## Decisions

- **Routes are upserted independently of sub-accounts rather than being a dependent edge.** — Per subaccount.go comment: routes are shared/hidden internal rows; a standalone routes table reveals which currencies are held without sub-account grouping detail.
- **Cost basis is canonicalized through the routing key, not stored verbatim for uniqueness.** — Ensures 0.70 and 0.7 map to one sub-account/route, enforced both in BuildRoutingKey and the unique constraint (tested in TestRepo_SubAccountRouteUniquenessConstraints).

## Example: Tx-aware ensure with upsert and re-query

```
func (r *repo) EnsureSubAccount(ctx context.Context, input ledgeraccount.CreateSubAccountInput) (*ledgeraccount.SubAccountData, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, tx *repo) (*ledgeraccount.SubAccountData, error) {
		route, err := tx.resolveOrCreateRoute(ctx, input)
		if err != nil { return nil, fmt.Errorf("failed to resolve route: %w", err) }
		err = tx.db.LedgerSubAccount.Create().
			SetNamespace(input.Namespace).SetAccountID(input.AccountID).SetRouteID(route.ID).
			OnConflict(sql.ConflictColumns(dbledgersubaccount.FieldNamespace, dbledgersubaccount.FieldAccountID, dbledgersubaccount.FieldRouteID), sql.ResolveWithIgnore()).Exec(ctx)
		if err != nil { return nil, fmt.Errorf("failed to ensure ledger sub-account: %w", err) }
		return tx.GetSubAccountByID(ctx, models.NamespacedID{Namespace: input.Namespace, ID: route.ID})
	})
}
```

<!-- archie:ai-end -->
