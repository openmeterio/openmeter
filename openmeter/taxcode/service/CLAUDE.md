# service

<!-- archie:ai-start -->

> Business-logic layer (taxcode.Service) over taxcode.Repository: enforces cross-field invariants (namespace ownership, system-managed protection, org-default protection) and orchestrates the get-or-create-by-app-mapping flow inside transactions.

## Patterns

**Service constructor with validated Config** — New(Config) validates Adapter (taxcode.Repository) + Logger and returns *Service asserted against taxcode.Service. (`var _ taxcode.Service = (*Service)(nil)`)
**Validate then transaction.Run delegating to adapter** — Every method: input.Validate() then transaction.Run(ctx, s.adapter, func...) (RunWithNoValue for void). Business checks live inside the tx callback before delegating. (`return transaction.Run(ctx, s.adapter, func(ctx context.Context) (taxcode.TaxCode, error) { return s.adapter.CreateTaxCode(ctx, input) })`)
**Cross-field guards inside the transaction** — UpsertOrganizationDefaultTaxCodes re-fetches both tax code IDs via GetTaxCode to prove namespace ownership before upserting; Update/Delete fetch existing first to check IsManagedBySystem and org-default usage. (`if _, err := s.GetTaxCode(ctx, taxcode.GetTaxCodeInput{NamespacedID: models.NamespacedID{Namespace: input.Namespace, ID: input.InvoicingTaxCodeID}}); err != nil { return ..., err }`)
**System-managed protection with AllowAnnotations bypass** — Update/Delete reject IsManagedBySystem() codes with NewGenericConflictError(ErrTaxCodeManagedBySystem) unless input.AllowAnnotations is set (internal seeding path). (`if existing.IsManagedBySystem() && !input.AllowAnnotations { return models.NewGenericConflictError(taxcode.ErrTaxCodeManagedBySystem) }`)
**Concurrency-safe get-or-create** — GetOrCreateByAppMapping looks up, creates on not-found, and on conflict re-reads; an orphaned key (conflict then still-not-found) returns ErrTaxCodeOrphanedKey rather than poisoning the pg tx. (`if models.IsGenericConflictError(err) { tc, retryErr := s.adapter.GetTaxCodeByAppMapping(ctx, ...); ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service struct, Config, New constructor | Holds only adapter + logger; no direct DB client. Assert taxcode.Service interface compliance. |
| `taxcode.go` | TaxCode CRUD + GetTaxCodeByAppMapping + GetOrCreateByAppMapping | Delete blocks org-default and system-managed codes; GetOrCreate derives key as fmt.Sprintf("%s_%s", AppType, TaxCode) and must distinguish orphaned-key from generic not-found. |
| `organizationdefaulttaxcodes.go` | Get/Upsert org defaults with namespace-ownership validation | Both InvoicingTaxCodeID and CreditGrantTaxCodeID are GetTaxCode-checked so a code from another namespace surfaces as IsTaxCodeNotFoundError, not a silent cross-tenant link. |
| `taxcode_test.go` | Service tests for system-managed, org-default protection, app-mapping preference | Asserts ValidationIssue.Code() equals taxcode.ErrCodeTaxCodeManagedBySystem / ErrCodeTaxCodeIsOrganizationDefault; system-managed seed must be preferred over user duplicate by GetTaxCodeByAppMapping. |
| `organizationdefaulttaxcodes_test.go` | Service tests for upsert idempotency, cross-namespace rejection, expand | Idempotent upsert must keep row ID and CreatedAt stable; cross-namespace tax codes must not resolve. |

## Anti-Patterns

- Calling adapter methods outside transaction.Run, losing atomicity across the validate-then-mutate guards
- Allowing org-default upsert without re-verifying both tax code IDs belong to the namespace (cross-tenant leak)
- Bypassing the IsManagedBySystem() check on Update/Delete without honoring AllowAnnotations
- Putting Ent/DB queries directly in the service instead of going through taxcode.Repository
- Treating a post-conflict not-found as a normal not-found instead of ErrTaxCodeOrphanedKey, risking a poisoned pg transaction

## Decisions

- **Namespace ownership of referenced tax codes is enforced in the service, not the DB FK** — Org defaults reference tax codes by ID only; the service re-fetches each by (namespace, id) so a foreign-namespace ID surfaces as not-found rather than a valid but cross-tenant link.
- **Get-or-create handles the create-conflict race explicitly** — Concurrent first-touch of the same app mapping must converge on one row; the retry read plus orphaned-key error keeps the surrounding Postgres transaction usable.

## Example: Service method: validate, transact, business guard, delegate to adapter

```
func (s *Service) UpdateTaxCode(ctx context.Context, input taxcode.UpdateTaxCodeInput) (taxcode.TaxCode, error) {
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
		return s.adapter.UpdateTaxCode(ctx, input)
	})
}
```

<!-- archie:ai-end -->
