# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing ledgeraccount.Repo for ledger account and sub-account persistence. All DB operations are transaction-aware via entutils.TransactingRepo; sub-account routes use an ON CONFLICT upsert keyed on a deterministic canonical routing key.

## Patterns

**TransactingRepo on every public method** — Wrap each exported method body in entutils.TransactingRepo(ctx, r, func(ctx, tx *repo)...) so the ctx-bound transaction is honored; never use r.db directly in a public method. (`return entutils.TransactingRepo(ctx, r, func(ctx context.Context, tx *repo) (*ledgeraccount.AccountData, error) { ... })`)
**TxUser triad: Tx / WithTx / Self** — repo implements entutils.TxUser[*repo] (WithTx rebinds, Self returns self) and transaction.TxCreator (Tx via HijackTx + NewTxDriver); both var _ assertions in repo.go must stay satisfied. (`func (r *repo) WithTx(ctx, tx *entutils.TxDriver) *repo { return &repo{db: entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()).Client()} }`)
**Route upsert via ON CONFLICT + canonical key** — resolveOrCreateRoute normalizes the route, computes ledger.BuildRoutingKey(RoutingKeyVersionV1, normalized), then ON CONFLICT DO NOTHING insert + re-fetch. EnsureSubAccount upserts on (namespace, accountID, routeID). (`routeKey, _ := ledger.BuildRoutingKey(ledger.RoutingKeyVersionV1, normalizedRoute); create.OnConflict(sql.ConflictColumns(...), sql.ResolveWithIgnore()).Exec(ctx)`)
**Normalize route before key computation and storage** — input.Route.Normalize() must run before BuildRoutingKey and before setting Ent fields so decimal variants like '0.7' and '0.70' map to the same route row. (`normalizedRoute, err := input.Route.Normalize(); routeKey, err := ledger.BuildRoutingKey(ledger.RoutingKeyVersionV1, normalizedRoute)`)
**WithRoute+WithAccount eager loading for sub-account reads** — GetSubAccountByID and ListSubAccounts must call .WithRoute().WithAccount(); MapSubAccountData errors if either edge is nil. (`tx.db.LedgerSubAccount.Query().Where(...).WithRoute().WithAccount().Only(ctx)`)
**MapAccountData / MapSubAccountData as pure converters** — Both exported converters turn Ent entities into domain types with no repo calls; MapSubAccountData guards both edges before access. (`func MapSubAccountData(entity *db.LedgerSubAccount) (ledgeraccount.SubAccountData, error) { if entity.Edges.Account == nil { return ..., fmt.Errorf("account edge is required") } ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `repo.go` | repo struct, NewRepo constructor, Tx/WithTx/Self triad; DI entry point. | Both var _ assertions (ledgeraccount.Repo and entutils.TxUser[*repo]) must remain satisfied; dropping either breaks transaction rebinding. |
| `account.go` | LedgerAccount CRUD (CreateAccount, GetAccountByID, ListAccounts) plus MapAccountData. | ListAccounts filters by AccountTypes only when non-empty; omitting the guard returns unscoped results across all types. |
| `subaccount.go` | EnsureSubAccount (idempotent upsert), GetSubAccountByID, ListSubAccounts, internal resolveOrCreateRoute, MapSubAccountData. | resolveOrCreateRoute has no own TransactingRepo — call only inside EnsureSubAccount's outer wrapper. ListSubAccounts normalizes the route filter before predicates. |
| `repo_test.go` | Integration tests using testutils.InitPostgresDB + migrate.Up, wiring repo via adapter.NewRepo (no app/common). | Use t.Context() not context.Background(); run the full golang-migrate Up or FK constraints fail; never import app/common in setup. |

## Anti-Patterns

- Calling r.db.LedgerAccount/LedgerSubAccount directly in a public method without TransactingRepo — falls off the caller's transaction.
- Querying LedgerSubAccount without .WithRoute().WithAccount() — MapSubAccountData errors on nil edges.
- Manually constructing routingKey strings instead of ledger.BuildRoutingKey — breaks canonical uniqueness.
- Using context.Background() in tests instead of t.Context().
- Importing app/common in test setup — causes import cycles; use adapter.NewRepo directly.

## Decisions

- **Routes are a separate shared table (LedgerSubAccountRoute), not embedded** — Routes give a view of which currencies/cost bases are held without joining through sub-account structure; upsert on routing key makes route reuse automatic.
- **Cost basis canonicalized via Route.Normalize() before key + storage** — Decimal representations like '0.7' and '0.70' must map to the same route row to preserve uniqueness; normalization happens once in the adapter, never in callers.

## Example: Add a new filtered list method (e.g. ListAccountsByType)

```
func (r *repo) ListAccountsByType(ctx context.Context, ns string, accType ledger.AccountType) ([]*ledgeraccount.AccountData, error) {
    return entutils.TransactingRepo(ctx, r, func(ctx context.Context, tx *repo) ([]*ledgeraccount.AccountData, error) {
        entities, err := tx.db.LedgerAccount.Query().Where(ledgeraccountdb.Namespace(ns), ledgeraccountdb.AccountType(accType)).All(ctx)
        if err != nil { return nil, fmt.Errorf("list accounts by type: %w", err) }
        out := make([]*ledgeraccount.AccountData, 0, len(entities))
        for _, e := range entities { ad, err := MapAccountData(e); if err != nil { return nil, err }; out = append(out, ad) }
        return out, nil
    })
}
```

<!-- archie:ai-end -->
