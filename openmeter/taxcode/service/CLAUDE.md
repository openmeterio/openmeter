# service

<!-- archie:ai-start -->

> Business-logic layer for tax codes implementing taxcode.Service. Enforces system-managed protection, get-or-create conflict retry, and wraps all adapter calls in transaction.Run to guarantee atomicity.

## Patterns

**transaction.Run wraps every adapter call** — Every service method wraps its adapter delegation in transaction.Run (or transaction.RunWithNoValue for void returns) so multi-step operations (guard read + write) are atomic. (`return transaction.Run(ctx, s.adapter, func(ctx context.Context) (taxcode.TaxCode, error) { return s.adapter.CreateTaxCode(ctx, input) })`)
**IsManagedBySystem guard before mutations** — UpdateTaxCode and DeleteTaxCode fetch the existing record inside the transaction and return models.NewGenericConflictError(taxcode.ErrTaxCodeManagedBySystem) if IsManagedBySystem() is true and input.AllowAnnotations is false. (`if existing.IsManagedBySystem() && !input.AllowAnnotations { return taxcode.TaxCode{}, models.NewGenericConflictError(taxcode.ErrTaxCodeManagedBySystem) }`)
**GetOrCreate with conflict retry** — GetOrCreateByAppMapping handles a create→conflict race by catching models.IsGenericConflictError and retrying with GetTaxCodeByAppMapping. This is idempotent under concurrent requests. (`if models.IsGenericConflictError(err) { return s.adapter.GetTaxCodeByAppMapping(ctx, taxcode.GetTaxCodeByAppMappingInput(input)) }`)
**input.Validate() at service entry** — Every exported method validates input before any adapter call, returning early on error. (`if err := input.Validate(); err != nil { return taxcode.TaxCode{}, err }`)
**Key derivation convention for auto-created tax codes** — GetOrCreateByAppMapping derives the key as '<AppType>_<TaxCode>' when auto-creating a new record from an app mapping. (`key := fmt.Sprintf("%s_%s", input.AppType, input.TaxCode)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service struct constructor and Config validation. Holds the taxcode.Repository adapter dependency. | Service depends only on taxcode.Repository — never import Ent directly here. |
| `taxcode.go` | All TaxCode CRUD plus GetOrCreateByAppMapping. The get-then-mutate pattern inside transaction.Run provides atomicity. | The get-then-mutate pattern is only safe because the adapter's TransactingRepo rebinds to the same transaction. Breaking that rebinding makes the IsManagedBySystem guard non-atomic. |
| `organizationdefaulttaxcodes.go` | GetOrganizationDefaultTaxCodes and UpsertOrganizationDefaultTaxCodes. Validates that both referenced tax code IDs belong to the same namespace before upsert. | Cross-namespace check is done by calling s.GetTaxCode (not the adapter directly) so namespace scoping is enforced. |
| `taxcode_test.go` | Integration tests for system-managed protection and user-managed CRUD via taxcodetestutils.NewTestEnv. | Tests use t.Context() — never context.Background(). env.DBSchemaMigrate(t) must be called before first DB access. |
| `organizationdefaulttaxcodes_test.go` | Integration tests for upsert idempotency, cross-namespace rejection, expand behavior. | Same t.Context() and DBSchemaMigrate requirements as taxcode_test.go. |

## Anti-Patterns

- Calling adapter methods outside transaction.Run — loses atomicity for get-then-mutate flows.
- Skipping IsManagedBySystem check in new mutating methods — allows external callers to overwrite system-synced records.
- Using context.Background() instead of t.Context() in tests.
- Implementing key derivation or conflict retry logic inside the adapter — business logic belongs in the service layer.
- Adding adapter dependencies beyond taxcode.Repository — the service must stay decoupled from Ent directly.

## Decisions

- **transaction.Run at the service layer in addition to entutils.TransactingRepo in the adapter.** — Multi-step flows (guard read + write) must be atomic; starting the transaction at the service layer ensures both the IsManagedBySystem read and the write share the same transaction.
- **AllowAnnotations flag on UpdateTaxCodeInput/DeleteTaxCodeInput to bypass system-managed protection.** — The LLM cost sync job needs to update system-managed records; a flag is safer than removing the guard or exposing a separate internal method.

## Example: Add a new mutating service method with system-managed guard and transaction wrapping.

```
func (s *Service) PatchTaxCode(ctx context.Context, input taxcode.PatchTaxCodeInput) (taxcode.TaxCode, error) {
	if err := input.Validate(); err != nil {
		return taxcode.TaxCode{}, err
	}
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (taxcode.TaxCode, error) {
		existing, err := s.adapter.GetTaxCode(ctx, taxcode.GetTaxCodeInput{NamespacedID: input.NamespacedID})
		if err != nil {
			return taxcode.TaxCode{}, err
		}
		if existing.IsManagedBySystem() && !input.AllowAnnotations {
			return taxcode.TaxCode{}, models.NewGenericConflictError(taxcode.ErrTaxCodeManagedBySystem)
		}
		return s.adapter.PatchTaxCode(ctx, input)
	})
}
```

<!-- archie:ai-end -->
