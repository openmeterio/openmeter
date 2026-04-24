# service

<!-- archie:ai-start -->

> Business-logic layer for tax codes implementing taxcode.Service; enforces system-managed protection, upsert-on-conflict for app mappings, and wraps all adapter calls in transaction.Run.

## Patterns

**transaction.Run wraps every adapter call** — Every service method wraps its adapter delegation in transaction.Run (or transaction.RunWithNoValue for void returns) so the operation is atomic even when called outside an existing tx. (`return transaction.Run(ctx, s.adapter, func(ctx context.Context) (taxcode.TaxCode, error) { return s.adapter.CreateTaxCode(ctx, input) })`)
**IsManagedBySystem guard before mutations** — UpdateTaxCode and DeleteTaxCode fetch the existing record first and return models.NewGenericConflictError(taxcode.ErrTaxCodeManagedBySystem) if IsManagedBySystem() is true and input.AllowAnnotations is false. (`if existing.IsManagedBySystem() && !input.AllowAnnotations { return taxcode.TaxCode{}, models.NewGenericConflictError(taxcode.ErrTaxCodeManagedBySystem) }`)
**GetOrCreate with conflict retry** — GetOrCreateByAppMapping handles a create→conflict race by catching models.IsGenericConflictError and retrying with GetTaxCodeByAppMapping. (`if models.IsGenericConflictError(err) { return s.adapter.GetTaxCodeByAppMapping(ctx, ...) }`)
**input.Validate() at service entry** — Every exported method validates input before any adapter call, returning early on error. (`if err := input.Validate(); err != nil { return taxcode.TaxCode{}, err }`)
**Key derivation convention for auto-created tax codes** — GetOrCreateByAppMapping derives the key as '<AppType>_<TaxCode>' when auto-creating a new record from an app mapping. (`key := fmt.Sprintf("%s_%s", input.AppType, input.TaxCode)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `taxcode.go` | Sole service implementation; all methods delegate to taxcode.Repository after validation and system-managed checks. | The get-then-mutate pattern inside transaction.Run is not atomic at the DB level without the adapter's underlying transaction — ensure the adapter's TransactingRepo rebinds correctly or concurrent requests can bypass the system-managed guard. |
| `taxcode_test.go` | Integration tests covering system-managed block/bypass and user-managed CRUD paths using taxcodetestutils.NewTestEnv. | Tests call t.Context() (not context.Background()); follow this for any new tests. The test bootstraps via env.DBSchemaMigrate(t) before first use. |

## Anti-Patterns

- Calling adapter methods outside transaction.Run — loses atomicity for multi-step operations like get-then-update.
- Skipping IsManagedBySystem check in new mutating methods — allows external callers to overwrite system-synced records.
- Using context.Background() instead of t.Context() in tests.
- Implementing business logic (key derivation, conflict retry) inside the adapter instead of the service layer.
- Adding adapter dependencies other than taxcode.Repository — the service must stay decoupled from Ent directly.

## Decisions

- **transaction.Run at the service layer in addition to entutils.TransactingRepo in the adapter.** — Multi-step flows (get + mutate) must be atomic; starting the transaction at the service layer ensures both the guard read and the write share the same transaction.
- **AllowAnnotations flag on UpdateTaxCodeInput/DeleteTaxCodeInput to bypass system-managed protection.** — The LLM cost sync job needs to update system-managed records; a flag is safer than removing the guard entirely or exposing a separate internal method.

<!-- archie:ai-end -->
