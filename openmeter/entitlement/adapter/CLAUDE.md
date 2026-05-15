# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing entitlement.EntitlementRepo and balanceworker.BalanceWorkerRepository — the sole persistence layer for entitlement rows, usage-reset records, and balance-worker ingest-event queries.

## Patterns

**TransactingRepo on every method** — Every public method body wraps with entutils.TransactingRepo or TransactingRepoWithNoValue so the ctx-bound Ent transaction is honored. Never call repo.db.* directly in the method body. (`entutils.TransactingRepo(ctx, a, func(ctx context.Context, repo *entitlementDBAdapter) (*entitlement.Entitlement, error) { return repo.mapEntitlementEntity(repo.db.Entitlement.Query()...) })`)
**TxUser + TxCreator triad in transaction.go** — Both entitlementDBAdapter and usageResetDBAdapter implement Tx(), WithTx(), and Self() in transaction.go. Any new adapter struct must add the same three methods or TransactingRepo will panic at runtime. (`func (e *entitlementDBAdapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *entitlementDBAdapter { txClient := db.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return NewPostgresEntitlementRepo(txClient.Client()) }`)
**Soft-delete filter on every query** — Active entitlements are excluded by appending db_entitlement.Or(db_entitlement.DeletedAtGT(now), db_entitlement.DeletedAtIsNil()). Omitting this filter returns deleted entitlements and breaks billing correctness. (`query.Where(db_entitlement.Or(db_entitlement.DeletedAtGT(clock.Now()), db_entitlement.DeletedAtIsNil()))`)
**withAllUsageResets eager-load on queries** — Any query that feeds mapEntitlementEntity must call withAllUsageResets(...) to eager-load the usage_reset edges. Omitting this causes mapEntitlementEntity to fail when computing UsagePeriod fields. (`withAllUsageResets(repo.db.Entitlement.Query(), []string{namespace}).Where(...).First(ctx)`)
**Compile-time interface assertions** — var _ repo = (*entitlementDBAdapter)(nil) and var _ interface{TxCreator; TxUser} = (*entitlementDBAdapter)(nil) guard interface compliance. Add these for every new adapter type. (`var _ repo = (*entitlementDBAdapter)(nil)`)
**Raw SQL for complex cross-table queries only** — ListEntitlementsAffectedByIngestEvents uses raw SQL via EntitlementsByIngestedEventsQuery because it requires CTEs joining customer, subject, feature, and meter tables. All other queries use Ent builders. (`query, args := EntitlementsByIngestedEventsQuery(repo.db.GetConfig().Driver.Dialect(), ns, subject, meters...); rows, err := repo.db.QueryContext(ctx, query, args...)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `entitlement.go` | Main adapter: all entitlement.EntitlementRepo and balanceworker.BalanceWorkerRepository method implementations using Ent query builders. | Any new method must wrap with TransactingRepo and include withAllUsageResets for queries that call mapEntitlementEntity. CreateEntitlement does a second query after Save() to reload edges — do the same for other create methods. |
| `transaction.go` | Implements Tx/WithTx/Self for both entitlementDBAdapter and usageResetDBAdapter. Required by pkg/framework/entutils transaction machinery. | If a new adapter struct is added to this package, copy all three methods here or the struct cannot participate in ctx-propagated transactions. |
| `usage_reset.go` | usageResetDBAdapter implements meteredentitlement.UsageResetRepo.Save — writes a usage_reset row inside a transaction. | Always call usageResetTime.Validate() before the Ent create call (already present); do not bypass it. |
| `entitlement_test.go` | Integration tests provisioning a real Postgres DB via testutils.InitPostgresDB. Constructs adapters directly from NewPostgresEntitlementRepo, not through app/common. | Uses context.Background() — acceptable in tests only. Do not copy this pattern into production adapter methods. |

## Anti-Patterns

- Calling repo.db.* directly inside a method body without a TransactingRepo wrapper.
- Omitting withAllUsageResets on any query that feeds mapEntitlementEntity.
- Importing app/common in test files — construct adapters directly.
- Adding business logic to the adapter — keep it pure persistence.
- Using context.Background() in production adapter methods.

## Decisions

- **Two separate adapter structs (entitlementDBAdapter, usageResetDBAdapter) each with their own TxUser implementation.** — Ent transactions are client-scoped; splitting keeps each struct small and avoids mixing unrelated schema edges in the same rebind path.
- **Soft-delete via DeletedAt timestamp rather than hard delete.** — Entitlement history is needed for billing reconciliation; hard deletes would destroy audit trails required for grant burn-down replay.

## Example: Adding a new write method to entitlementDBAdapter

```
func (a *entitlementDBAdapter) UpdateFoo(ctx context.Context, id models.NamespacedID, val string) error {
	_, err := entutils.TransactingRepo[interface{}, *entitlementDBAdapter](
		ctx, a,
		func(ctx context.Context, repo *entitlementDBAdapter) (interface{}, error) {
			return nil, repo.db.Entitlement.UpdateOneID(id.ID).
				SetFoo(val).Exec(ctx)
		},
	)
	return err
}
```

<!-- archie:ai-end -->
