# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing ledgeraccount.Repo for ledger account and sub-account persistence. All DB operations are transaction-aware via entutils.TransactingRepo; sub-account routes use an ON CONFLICT upsert pattern keyed on a deterministic canonical routing key.

## Patterns

**TransactingRepo on every public method** — Every exported method body must be wrapped in entutils.TransactingRepo(ctx, r, func(ctx, tx *repo) ...) so the ctx-bound Ent transaction is honored. Never use r.db directly in a public method without this wrapper. (`return entutils.TransactingRepo(ctx, r, func(ctx context.Context, tx *repo) (*ledgeraccount.AccountData, error) { entity, err := tx.db.LedgerAccount.Create()... })`)
**TxUser triad: Tx / WithTx / Self** — repo must implement entutils.TxUser[*repo] via WithTx (rebinds to caller tx) and Self (returns self for standalone ops), plus transaction.TxCreator via Tx (HijackTx + NewTxDriver). Both var _ assertions in repo.go must stay satisfied. (`func (r *repo) WithTx(ctx context.Context, tx *entutils.TxDriver) *repo { return &repo{db: entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()).Client()} }`)
**Route upsert via ON CONFLICT + canonical routing key** — resolveOrCreateRoute normalizes the route, computes ledger.BuildRoutingKey(RoutingKeyVersionV1, normalizedRoute), then does an ON CONFLICT DO NOTHING insert followed by a re-fetch by (namespace, accountID, routingKeyVersion, routingKey). EnsureSubAccount similarly uses ON CONFLICT on (namespace, accountID, routeID). (`routeKey, _ := ledger.BuildRoutingKey(ledger.RoutingKeyVersionV1, normalizedRoute); create.OnConflict(sql.ConflictColumns(...), sql.ResolveWithIgnore()).Exec(ctx)`)
**Normalize route before key computation and DB storage** — input.Route.Normalize() must be called before BuildRoutingKey and before setting any Ent fields so that decimal variants like '0.7' and '0.70' map to the same route row. (`normalizedRoute, err := input.Route.Normalize(); routeKey, err := ledger.BuildRoutingKey(ledger.RoutingKeyVersionV1, normalizedRoute)`)
**WithRoute+WithAccount eager loading for sub-account reads** — GetSubAccountByID and ListSubAccounts must always call .WithRoute().WithAccount() on the Ent query. MapSubAccountData returns an error if either edge is nil, so omitting eager loads causes silent failures. (`tx.db.LedgerSubAccount.Query().Where(...).WithRoute().WithAccount().Only(ctx)`)
**MapAccountData / MapSubAccountData as pure converters** — Both exported functions convert Ent entities to domain types with no repo calls inside. MapSubAccountData must guard both edges before accessing them. (`func MapSubAccountData(entity *db.LedgerSubAccount) (ledgeraccount.SubAccountData, error) { if entity.Edges.Account == nil { return ..., fmt.Errorf("account edge is required") } ... }`)
**resolveOrCreateRoute is internal, not public** — resolveOrCreateRoute is always called from within the outer TransactingRepo in EnsureSubAccount. Adding a direct public call without a wrapping TransactingRepo bypasses tx rebinding. (`// EnsureSubAccount calls resolveOrCreateRoute only inside its TransactingRepo closure`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `repo.go` | Defines repo struct, NewRepo constructor, and Tx/WithTx/Self triad. DI entry point. | Both var _ compile-time assertions (ledgeraccount.Repo and entutils.TxUser[*repo]) must remain satisfied; dropping either breaks transaction rebinding for callers. |
| `account.go` | CRUD for LedgerAccount rows (CreateAccount, GetAccountByID, ListAccounts) plus exported MapAccountData converter. | ListAccounts filters by AccountTypes only when non-empty; omitting the guard returns unscoped results across all account types. |
| `subaccount.go` | EnsureSubAccount (idempotent upsert), GetSubAccountByID, ListSubAccounts, internal resolveOrCreateRoute, and MapSubAccountData. | resolveOrCreateRoute is not wrapped in its own TransactingRepo — it must only be called from within EnsureSubAccount's outer TransactingRepo. ListSubAccounts normalizes the route filter before building predicates; skipping normalization causes cost-basis filter mismatches. |
| `repo_test.go` | Integration tests using testutils.InitPostgresDB and migrate.Up. Wires repo directly from adapter.NewRepo without app/common. | Tests use t.Context() not context.Background(). DBSchemaMigrate runs the full golang-migrate Up; skip it and FK constraints will fail. Never import app/common in test setup. |

## Anti-Patterns

- Calling r.db.LedgerAccount/LedgerSubAccount directly in a public method without TransactingRepo — falls off the caller's transaction
- Querying LedgerSubAccount without .WithRoute().WithAccount() — MapSubAccountData errors on nil edges
- Manually constructing routingKey strings instead of using ledger.BuildRoutingKey — breaks canonical uniqueness
- Using context.Background() in tests instead of t.Context()
- Importing app/common in test setup — causes import cycles; use adapter.NewRepo directly

## Decisions

- **Routes are a separate table (LedgerSubAccountRoute) shared across sub-accounts, not embedded.** — Routes give a meaningful view of which currencies/cost bases are held without joining through sub-account structure; upsert on routing key makes route reuse automatic.
- **Cost basis is canonicalized via Route.Normalize() before key computation and DB storage.** — Decimal representations like '0.7' and '0.70' must map to the same route row to preserve uniqueness; normalization happens once in the adapter, never in callers.

## Example: Add a new filtered list method (e.g. ListAccountsByType)

```
func (r *repo) ListAccountsByType(ctx context.Context, ns string, accType ledger.AccountType) ([]*ledgeraccount.AccountData, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, tx *repo) ([]*ledgeraccount.AccountData, error) {
		entities, err := tx.db.LedgerAccount.Query().
			Where(ledgeraccountdb.Namespace(ns), ledgeraccountdb.AccountType(accType)).
			All(ctx)
		if err != nil {
			return nil, fmt.Errorf("list accounts by type: %w", err)
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
