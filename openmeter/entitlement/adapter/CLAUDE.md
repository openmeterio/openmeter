# adapter

<!-- archie:ai-start -->

> Postgres/Ent persistence layer for entitlements and usage resets. Implements entitlement.EntitlementRepo plus balanceworker.BalanceWorkerRepository over the Ent db.Client, and is the only place where entitlement DB rows are mapped to/from domain models.

## Patterns

**Wrap every method in TransactingRepo** — All repo methods enter entutils.TransactingRepo(ctx, a, func(ctx, repo){...}) so they rebind to the tx already carried in ctx instead of using the raw client directly. (`func (a *entitlementDBAdapter) GetEntitlement(ctx, id) (*entitlement.Entitlement, error) { return entutils.TransactingRepo(ctx, a, func(ctx, repo *entitlementDBAdapter) (...) {...}) }`)
**TxUser/Creator triple in transaction.go** — Each adapter implements Tx, WithTx and Self via HijackTx + NewTxClientFromRawConfig so it satisfies transaction.Creator and entutils.TxUser[*T]. (`func (e *entitlementDBAdapter) WithTx(ctx, tx *entutils.TxDriver) *entitlementDBAdapter { txClient := db.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return NewPostgresEntitlementRepo(txClient.Client()) }`)
**Interface compliance asserted with var _** — Compile-time assertions pin the concrete adapter to its interfaces, e.g. var _ repo = (*entitlementDBAdapter)(nil) where repo embeds EntitlementRepo and BalanceWorkerRepository. (`var _ repo = (*entitlementDBAdapter)(nil)`)
**Map db.Entitlement to domain via mapEntitlementEntity** — Never return raw Ent rows; every read funnels through repo.mapEntitlementEntity(res) after loading usage-reset edges with withAllUsageResets(...). (`return repo.mapEntitlementEntity(res)`)
**NotFound translated to domain errors** — db.IsNotFound(err) is converted to &entitlement.NotFoundError{...} or models.NewGenericNotFoundError(...) rather than leaking Ent errors. (`if db.IsNotFound(err) { return nil, &entitlement.NotFoundError{EntitlementID: entitlementID} }`)
**Soft-delete and active-window predicates** — Queries always guard with DeletedAtGT(at)/DeletedAtIsNil, customerNotDeletedAt(at) and EntitlementActiveAt(at) helpers rather than returning deleted/inactive rows. (`db_entitlement.Or(db_entitlement.DeletedAtGT(at), db_entitlement.DeletedAtIsNil())`)
**Raw SQL builder for cross-table fan-out** — Complex subject->customer / meter->feature resolution uses sql.Dialect(...).With(...) CTEs (EntitlementsByIngestedEventsQuery) executed via repo.db.QueryContext, with rows.Scan into balanceworker structs. (`query, args := EntitlementsByIngestedEventsQuery(dialect, ns, subject, meters...); rows, err := repo.db.QueryContext(ctx, query, args...)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `entitlement.go` | entitlementDBAdapter: all entitlement CRUD/list queries and the EntitlementsByIngestedEventsQuery CTE builder | GetActiveEntitlementOfCustomerAt uses .First() with a FIXME about not asserting single result; ListEntitlements splits FeatureIDsOrKeys into IDs vs keys via ulid.Parse |
| `usage_reset.go` | usageResetDBAdapter implementing meteredentitlement.UsageResetRepo (Save creates UsageReset rows) | Validate() is called inside the tx before insert; keep anchor/interval fields in sync with UsageResetUpdate |
| `transaction.go` | Tx/WithTx/Self plumbing for both adapters | Two structs share this file; adding a new adapter here means adding all three methods or TxUser compliance breaks |
| `entitlement_test.go` | adapter_test integration tests with real Postgres via testutils.InitPostgresDB | Schema created under a package-level sync.Mutex m; tests build repos via NewPostgresEntitlementRepo/NewPostgresUsageResetRepo, not app wiring |

## Anti-Patterns

- Calling repo.db.* outside a TransactingRepo wrapper, breaking tx propagation carried in ctx
- Returning raw *db.Entitlement rows instead of mapping through mapEntitlementEntity
- Leaking db.IsNotFound errors instead of entitlement.NotFoundError / GenericNotFoundError
- Querying entitlements without DeletedAt / active-window guards
- Adding business logic (defaulting, validation) here instead of in the service/connector layer

## Decisions

- **Adapter implements both EntitlementRepo and BalanceWorkerRepository via a private repo interface** — The balance worker reuses entitlement persistence, so one Ent-backed type serves both and stays consistent under a single tx.
- **Hand-written CTE SQL for ingest-event fan-out** — Resolving customer-by-subject and feature-by-meter in one round trip is impractical with the Ent fluent API.

## Example: Transaction-aware read returning a mapped domain entitlement

```
func (a *entitlementDBAdapter) GetEntitlement(ctx context.Context, id models.NamespacedID) (*entitlement.Entitlement, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, repo *entitlementDBAdapter) (*entitlement.Entitlement, error) {
		res, err := withAllUsageResets(repo.db.Entitlement.Query(), []string{id.Namespace}).
			Where(db_entitlement.ID(id.ID), db_entitlement.Namespace(id.Namespace),
				db_entitlement.Or(db_entitlement.DeletedAtGT(clock.Now()), db_entitlement.DeletedAtIsNil())).First(ctx)
		if err != nil {
			if db.IsNotFound(err) { return nil, &entitlement.NotFoundError{EntitlementID: id} }
			return nil, err
		}
		return repo.mapEntitlementEntity(res)
	})
}
```

<!-- archie:ai-end -->
