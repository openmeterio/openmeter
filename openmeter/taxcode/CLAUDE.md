# taxcode

<!-- archie:ai-start -->

> Domain package for tax codes and per-namespace OrganizationDefaultTaxCodes: the root declares the Service/Repository interfaces, all input/domain types (TaxCode, TaxCodeAppMappings, OrganizationDefaultTaxCodes), ValidationIssue-based errors, system-managed annotations, and a namespace.Handler that seeds tax codes on namespace creation. adapter/ (Ent), service/, and testutils/ implement and test it.

## Patterns

**ValidationIssue error catalog** — errors.go declares package-level models.ErrorCode constants + models.NewValidationIssue vars (with HTTP status attributes) and typed detectors (IsTaxCodeNotFoundError, IsOrganizationDefaultTaxCodesNotFoundError, etc.). (`var ErrTaxCodeNotFound = models.NewValidationIssue(ErrCodeTaxCodeNotFound, ..., commonhttp.WithHTTPStatusCodeAttribute(http.StatusNotFound))`)
**Errors-slice Validate** — Every input/domain Validate collects into var errs []error (NamespacedID/AppMappings/field checks) and returns models.NewNillableGenericValidationError(errors.Join(errs...)). (`_ models.Validator = (*CreateTaxCodeInput)(nil)`)
**System-managed annotation marker** — annotations.go defines AnnotationKeyManagedBy / AnnotationValueManagedBySystem; TaxCode.IsManagedBySystem reads them; system-created codes are protected from update/delete. (`Annotations: models.Annotations{AnnotationKeyManagedBy: AnnotationValueManagedBySystem}`)
**App-type-specific tax code format validation** — TaxCodeAppMapping.Validate switches on AppType (e.g. app.AppTypeStripe requires TaxCodeStripeRegexp `^txcd_\d{8}$`); TaxCodeAppMappings enforces unique app types via lo.UniqBy. (`if !TaxCodeStripeRegexp.MatchString(t.TaxCode) { errs = append(errs, ErrTaxCodeStripeInvalid) }`)
**Idempotent transactional namespace seeding** — NamespaceHandler.CreateNamespace runs all seed creates + org-default upsert in one transaction.RunWithNoValue, pre-listing existing codes and re-listing on a conflict (ensureTaxCode); validate() enforces exactly one DefaultInvoicing and one DefaultCreditGrant seed. (`transaction.RunWithNoValue(ctx, h.transactionManager, func(ctx) error {...})`)
**Orphaned-key sentinel error** — ErrTaxCodeOrphanedKey (a plain errors.New, not a ValidationIssue) signals a key-exists-but-mapping-changed race to avoid poisoning the pg transaction (25P02). (`func IsTaxCodeOrphanedKeyError(err error) bool { return errors.Is(err, ErrTaxCodeOrphanedKey) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service = TaxCodeService + OrganizationDefaultTaxCodesService; declares all *Input types with Validate and the AllowAnnotations inputOptions. | GetOrCreateByAppMapping is service-only (not on Repository); IncludeDeleted on Get/List inputs is internal-only, never from API handlers. |
| `repository.go` | Repository = entutils.TxCreator + tax code + org-default persistence methods. | Repository has no GetOrCreateByAppMapping — that orchestration lives in the service inside a transaction. |
| `taxcode.go` | Domain types: TaxCode, TaxCodeAppMapping(s), OrganizationDefaultTaxCodes + Expand, with Validate/Equal/IsManagedBySystem/GetAppMapping. | TaxCode.Equal excludes ManagedModel timestamps, Metadata, and Annotations from comparison. |
| `errors.go` | Error code constants, ValidationIssue vars, constructors and Is* detectors. | ErrTaxCodeOrphanedKey is a plain error, not a ValidationIssue — detect with errors.Is, others with errors.As + Code(). |
| `namespacehandler.go` | NamespaceHandler implements namespace.Handler; seeds tax codes + org defaults idempotently in a transaction. | Config.validate() requires exactly one DefaultInvoicing and one DefaultCreditGrant seed; DeleteNamespace is intentionally a no-op. |
| `annotations.go` | Defines the managed-by annotation key/value used to mark system-created tax codes. | Only system-managed codes are protected from mutation; pre-existing codes without the annotation are left untouched by seeding. |

## Anti-Patterns

- Returning raw errors from Validate() instead of wrapping with models.NewNillableGenericValidationError.
- Detecting ErrTaxCodeOrphanedKey with errors.As/ValidationIssue instead of errors.Is (it is a plain sentinel).
- Mutating or deleting a system-managed tax code without honoring AllowAnnotations / the IsManagedBySystem guard.
- Running seed creation or org-default upsert outside the single CreateNamespace transaction.
- Adding a Stripe app mapping whose tax code does not match the `^txcd_\d{8}$` regexp.

## Decisions

- **Org defaults seeded require exactly one invoicing and one credit-grant seed.** — OrganizationDefaultTaxCodes must reference both a single invoicing and a single credit-grant tax code per namespace.
- **Use a typed sentinel ErrTaxCodeOrphanedKey for the create-conflict race.** — Prevents a raw pg constraint error (25P02) from poisoning the surrounding transaction.
- **Mark system-created codes via annotation rather than a DB column.** — IsManagedBySystem can gate update/delete protection without a schema change, and pre-existing user codes stay unmanaged.

## Example: Idempotent seeding of tax codes inside one transaction

```
func (h *NamespaceHandler) CreateNamespace(ctx context.Context, ns string) error {
	return transaction.RunWithNoValue(ctx, h.transactionManager, func(ctx context.Context) error {
		listed, err := h.service.ListTaxCodes(ctx, ListTaxCodesInput{Namespace: ns})
		if err != nil {
			return fmt.Errorf("list tax codes: %w", err)
		}
		existingByKey := lo.SliceToMap(listed.Items, func(tc TaxCode) (string, TaxCode) { return tc.Key, tc })
		var invoicingID, creditGrantID string
		for _, seed := range h.seeds {
			id, err := h.ensureTaxCode(ctx, ns, seed, existingByKey)
			if err != nil {
				return fmt.Errorf("seed tax code %q: %w", seed.Key, err)
			}
			if seed.DefaultInvoicing { invoicingID = id }
			if seed.DefaultCreditGrant { creditGrantID = id }
// ...
```

<!-- archie:ai-end -->
