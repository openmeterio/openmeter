# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing ledgeraccount.Repo for ledger account and sub-account persistence. All DB operations must be transaction-aware via entutils.TransactingRepo; route creation uses an upsert pattern keyed on canonical routing key.

## Patterns

**TransactingRepo wrapping** — Every exported method body is wrapped in entutils.TransactingRepo(ctx, r, func(ctx, tx *repo) ...) so the ctx-bound Ent transaction is honored. Never use r.db directly in a public method without this wrapper. (`return entutils.TransactingRepo(ctx, r, func(ctx context.Context, tx *repo) (*ledgeraccount.AccountData, error) { ... r.db.LedgerAccount... })`)
**TxUser implementation on repo** — repo implements entutils.TxUser[*repo] via WithTx and Self, enabling TransactingRepo to rebind to the caller's transaction. Both must be present for any new repo struct. (`func (r *repo) WithTx(ctx context.Context, tx *entutils.TxDriver) *repo { return &repo{db: entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()).Client()} }`)
**Route upsert via canonical routing key** — EnsureSubAccount normalizes the route, builds a deterministic routing key via ledger.BuildRoutingKey(RoutingKeyVersionV1, ...), attempts insert, and on constraint error falls back to query by (namespace, accountID, routingKeyVersion, routingKey). Cost basis is canonicalized before key computation so '0.7' and '0.70' collide correctly. (`normalizedRoute, _ := input.Route.Normalize(); routeKey, _ := ledger.BuildRoutingKey(ledger.RoutingKeyVersionV1, normalizedRoute); create.Save(ctx) → on IsConstraintError → query by routingKey`)
**Constraint-error idempotent upsert for sub-accounts** — EnsureSubAccount attempts Create on LedgerSubAccount; on db.IsConstraintError it re-fetches the existing row by (namespace, accountID, routeID) instead of failing. This makes the operation safe to call multiple times. (`if db.IsConstraintError(err) { entity, err = r.db.LedgerSubAccount.Query().Where(...).Only(ctx) }`)
**WithRoute+WithAccount eager loading for subaccount reads** — GetSubAccountByID and ListSubAccounts always call .WithRoute().WithAccount() on the query. MapSubAccountData panics-as-error if either edge is nil, so never query sub-accounts without these eager loads. (`r.db.LedgerSubAccount.Query().Where(...).WithRoute().WithAccount().Only(ctx)`)
**MapAccountData / MapSubAccountData as pure domain converters** — Both are exported functions (not methods) that convert Ent entities to domain types. They must stay pure — no repo calls inside. MapSubAccountData returns an error if Account or Route edges are nil. (`func MapSubAccountData(entity *db.LedgerSubAccount) (ledgeraccount.SubAccountData, error) { if entity.Edges.Account == nil { return ..., fmt.Errorf(...) } ... }`)
**RouteFilter normalization before predicate building** — ListSubAccounts calls input.Route.Normalize() before constructing Ent predicates so that filter values match the canonicalized DB values (especially cost basis decimal normalization). (`normalizedRoute, err := input.Route.Normalize(); if normalizedRoute.CostBasis.IsPresent() { ... dbledgersubaccountroute.CostBasis(*costBasis) ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `repo.go` | Defines the repo struct, NewRepo constructor, and the Tx/WithTx/Self methods that satisfy entutils.TxUser[*repo] and transaction.TxCreator. This is the DI entry point. | Both var _ assertions (ledgeraccount.Repo and entutils.TxUser[*repo]) must stay satisfied; dropping either breaks transaction rebinding. |
| `account.go` | CRUD for LedgerAccount rows: CreateAccount, GetAccountByID, ListAccounts. Also holds the exported MapAccountData converter. | All three methods are wrapped in TransactingRepo; ListAccounts filters by AccountTypes when non-empty — omitting that filter returns unscoped results. |
| `subaccount.go` | EnsureSubAccount (idempotent upsert), GetSubAccountByID, ListSubAccounts, and the internal resolveOrCreateRoute (route upsert). Also holds MapSubAccountData. | resolveOrCreateRoute is not wrapped in TransactingRepo itself — it is always called from within an outer TransactingRepo in EnsureSubAccount. Adding a public call to it directly would bypass tx rebinding. |
| `repo_test.go` | Integration tests using testutils.InitPostgresDB + migrate.Up. TestEnv wires repo directly from adapter.NewRepo without app/common — follow this for new tests. | Tests call t.Context() (not context.Background()). DBSchemaMigrate runs the full golang-migrate Up; skip it and FK constraints will fail. |

## Anti-Patterns

- Calling r.db.LedgerAccount/LedgerSubAccount directly in a public method without TransactingRepo — falls off the caller's transaction
- Querying LedgerSubAccount without .WithRoute().WithAccount() — MapSubAccountData will error on nil edges
- Manually constructing routingKey strings instead of using ledger.BuildRoutingKey — breaks canonical uniqueness
- Using context.Background() in tests instead of t.Context()
- Importing app/common in test setup — causes import cycles; use adapter.NewRepo directly

## Decisions

- **Routes are a separate table (LedgerSubAccountRoute) shared across sub-accounts, not embedded in sub-accounts.** — Routes give a meaningful view of which currencies/cost bases are held without requiring a join through the structural sub-account grouping; upsert on routing key makes route reuse automatic.
- **Cost basis is canonicalized via Route.Normalize() before key computation and DB storage.** — Decimal representations like '0.7' and '0.70' must map to the same route row to preserve uniqueness guarantees; normalization happens once in the adapter, never in callers.

## Example: Add a new filtered list method on repo (e.g. ListAccountsByType)

```
func (r *repo) ListAccountsByType(ctx context.Context, ns string, accType ledger.AccountType) ([]*ledgeraccount.AccountData, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, tx *repo) ([]*ledgeraccount.AccountData, error) {
		entities, err := r.db.LedgerAccount.Query().
			Where(ledgeraccountdb.Namespace(ns), ledgeraccountdb.AccountType(accType)).
			All(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list accounts by type: %w", err)
		}
		out := make([]*ledgeraccount.AccountData, 0, len(entities))
		for _, e := range entities {
			ad, err := MapAccountData(e)
			if err != nil {
				return nil, err
			}
			out = append(out, ad)
// ...
```

<!-- archie:ai-end -->
