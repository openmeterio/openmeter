# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing entitlement.EntitlementRepo, balanceworker.BalanceWorkerRepository, and meteredentitlement.UsageResetRepo — the sole persistence layer for entitlement rows, usage-reset records, and balance-worker ingest-event queries.

## Patterns

**TransactingRepo on every method** — Every public method body wraps with entutils.TransactingRepo / TransactingRepoWithNoValue so the ctx-bound Ent transaction is honored. Never call repo.db.* outside the wrapper closure. (`entutils.TransactingRepo(ctx, a, func(ctx, repo *entitlementDBAdapter) (*entitlement.Entitlement, error) { return repo.mapEntitlementEntity(...) })`)
**TxUser+TxCreator triad in transaction.go** — Both entitlementDBAdapter and usageResetDBAdapter implement Tx()/WithTx()/Self() in transaction.go. A new adapter struct missing these makes TransactingRepo panic. (`func (e *entitlementDBAdapter) WithTx(ctx, tx *entutils.TxDriver) *entitlementDBAdapter { return NewPostgresEntitlementRepo(db.NewTxClientFromRawConfig(ctx, *tx.GetConfig()).Client()) }`)
**Soft-delete filter on every query** — Active entitlements exclude soft-deleted rows via db_entitlement.Or(DeletedAtGT(now), DeletedAtIsNil()). Omitting returns deleted rows and breaks billing correctness. (`Where(db_entitlement.Or(db_entitlement.DeletedAtGT(clock.Now()), db_entitlement.DeletedAtIsNil()))`)
**withAllUsageResets eager-load** — Any query feeding mapEntitlementEntity must call withAllUsageResets to eager-load usage_reset edges, or UsagePeriod computation fails. (`withAllUsageResets(repo.db.Entitlement.Query(), []string{namespace}).Where(...).First(ctx)`)
**Compile-time interface assertions** — var _ repo = (*entitlementDBAdapter)(nil) and var _ interface{transaction.Creator; entutils.TxUser[...]} guard compliance. Add for every new adapter type. (`var _ repo = (*entitlementDBAdapter)(nil)`)
**Raw SQL only for complex cross-table CTEs** — ListEntitlementsAffectedByIngestEvents uses EntitlementsByIngestedEventsQuery raw SQL because it joins customer/subject/feature/meter via CTEs. All other queries use Ent builders. (`query, args := EntitlementsByIngestedEventsQuery(repo.db.GetConfig().Driver.Dialect(), ns, subject, meters...); rows, err := repo.db.QueryContext(ctx, query, args...)`)
**Reload edges after Create** — CreateEntitlement does a second Query().Only(ctx) after Save() to reload customer/subject edges before mapEntitlementEntity. Replicate for other create methods. (`res, _ := cmd.Save(ctx); entWithEdges, _ := repo.db.Entitlement.Query().Where(db_entitlement.ID(res.ID)).Only(ctx)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `entitlement.go` | Main adapter: all EntitlementRepo + BalanceWorkerRepository method implementations using Ent query builders plus the raw-SQL CTE query. | New methods must wrap in TransactingRepo and call withAllUsageResets when feeding mapEntitlementEntity. |
| `transaction.go` | Tx/WithTx/Self for both entitlementDBAdapter and usageResetDBAdapter via HijackTx + NewTxDriver. | New adapter struct in this package needs all three methods copied or it cannot join ctx transactions. |
| `usage_reset.go` | usageResetDBAdapter.Save writes a usage_reset row inside a transaction. | Always call usageResetTime.Validate() before the Ent create call; do not bypass. |
| `entitlement_test.go` | Integration tests provisioning real Postgres via testutils.InitPostgresDB, constructing adapters directly from NewPostgresEntitlementRepo. | Uses context.Background() — acceptable in tests only, never copy into production methods. |

## Anti-Patterns

- Calling repo.db.* outside a TransactingRepo wrapper.
- Omitting withAllUsageResets on a query that feeds mapEntitlementEntity.
- Importing app/common in test files — construct adapters directly.
- Adding business logic to the adapter — keep it pure persistence.
- Using context.Background() in production adapter methods.

## Decisions

- **Two separate adapter structs (entitlementDBAdapter, usageResetDBAdapter) each with their own TxUser.** — Ent transactions are client-scoped; splitting keeps each rebind path small and unrelated edges separate.
- **Soft-delete via DeletedAt timestamp rather than hard delete.** — Entitlement history is needed for billing reconciliation and grant burn-down replay; hard delete destroys the audit trail.

## Example: Adding a write method to entitlementDBAdapter

```
func (a *entitlementDBAdapter) UpdateFoo(ctx context.Context, id models.NamespacedID, val string) error {
	_, err := entutils.TransactingRepo[interface{}, *entitlementDBAdapter](ctx, a, func(ctx context.Context, repo *entitlementDBAdapter) (interface{}, error) {
		return nil, repo.db.Entitlement.UpdateOneID(id.ID).SetFoo(val).Exec(ctx)
	})
	return err
}
```

<!-- archie:ai-end -->
