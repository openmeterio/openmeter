# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing entitlement.EntitlementRepo and balanceworker.BalanceWorkerRepository for the entitlement domain. All DB access goes through this package; it is the sole persistence layer for entitlement rows and usage-reset records.

## Patterns

**TransactingRepo wrapping** — Every method body is wrapped with entutils.TransactingRepo or TransactingRepoWithNoValue so the ctx-bound Ent transaction is honored. Never call repo.db.* directly in the method body. (`entutils.TransactingRepo(ctx, a, func(ctx context.Context, repo *entitlementDBAdapter) (*entitlement.Entitlement, error) { ... })`)
**TxUser + TxCreator implementation in transaction.go** — entitlementDBAdapter and usageResetDBAdapter each implement Tx(), WithTx(), and Self() in transaction.go, enabling entutils to rebind to an existing ctx transaction or start a new one. (`func (e *entitlementDBAdapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *entitlementDBAdapter { txClient := db.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return NewPostgresEntitlementRepo(txClient.Client()) }`)
**Soft-delete filter** — Deleted entitlements are excluded by appending db_entitlement.Or(db_entitlement.DeletedAtGT(now), db_entitlement.DeletedAtIsNil()). Never omit this filter when querying active entitlements. (`query.Where(db_entitlement.Or(db_entitlement.DeletedAtGT(now), db_entitlement.DeletedAtIsNil()))`)
**withAllUsageResets helper** — List/Get queries call withAllUsageResets(...) to eager-load usage reset edges needed for UsagePeriod calculations. Omitting this causes mapEntitlementEntity to fail. (`withAllUsageResets(repo.db.Entitlement.Query(), []string{namespace}).Where(...)`)
**Compile-time interface assertion** — var _ repo = (*entitlementDBAdapter)(nil) and var _ interface{TxCreator; TxUser} = (*entitlementDBAdapter)(nil) guard interface compliance at compile time. (`var _ repo = (*entitlementDBAdapter)(nil)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `entitlement.go` | Main adapter: implements all entitlement.EntitlementRepo and balanceworker.BalanceWorkerRepository methods using Ent queries. | Any new method must use TransactingRepo; always include withAllUsageResets for queries that map to domain Entitlement structs. |
| `transaction.go` | Implements Tx/WithTx/Self for both entitlementDBAdapter and usageResetDBAdapter. Required by entutils transaction machinery. | If you add a new adapter struct, copy this pattern or entutils will panic. |
| `usage_reset.go` | usageResetDBAdapter implementing meteredentitlement.UsageResetRepo.Save — writes a usage_reset row inside a transaction. | Must call usageResetTime.Validate() before the Ent create call. |
| `entitlement_test.go` | Integration tests for the adapter using a real Postgres DB provisioned by testutils.InitPostgresDB. Constructs adapters directly, not through app/common. | Uses context.Background() — ok in tests; do not copy this pattern into production code. |

## Anti-Patterns

- Calling repo.db.* directly inside a method body without TransactingRepo wrapping.
- Constructing adapters via app/common in tests — build directly from NewPostgresEntitlementRepo.
- Omitting withAllUsageResets on any query that feeds mapEntitlementEntity.
- Adding business logic to the adapter — keep it pure persistence.
- Using context.Background() inside production adapter methods.

## Decisions

- **Two separate adapters (entitlementDBAdapter, usageResetDBAdapter) each with their own TxUser implementation.** — Ent transactions are client-scoped; splitting keeps each struct small and avoids mixing unrelated schema edges in the same rebind path.
- **Soft-delete via DeletedAt timestamp rather than hard delete.** — Entitlement history is needed for billing reconciliation; hard deletes would destroy audit trails.

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
