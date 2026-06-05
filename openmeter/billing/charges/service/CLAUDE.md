# service

<!-- archie:ai-start -->

> Root charges facade implementing charges.Service: orchestrates flatfee/usagebased/creditpurchase per-type services, the meta adapter, recognizer, and invoiceupdater into Create/AdvanceCharges/GetByID(s)/ListCharges/ApplyPatches/ListCustomersToAdvance plus subscription and standard-invoice hooks. Single entry point both production wiring and tests drive.

## Patterns

**Validate -> namespace lockdown -> transaction.Run** — Mutating methods run input.Validate(), then s.validateNamespaceLockdown(ns), then wrap the body in transaction.Run / RunWithNoValue over s.adapter. (`if err := s.validateNamespaceLockdown(input.Customer.Namespace); err != nil { return nil, err }; advancedCharges, err := transaction.Run(ctx, s.adapter, func(ctx) (...) {...})`)
**Per-type fan-out via ByType / chargesByType** — Heterogeneous charge intents/charges are split into flatFee/usageBased/creditPurchase buckets (input.Intents.ByType, helpers.chargesByType) and dispatched to the matching sub-service, then re-merged by original index with charges.WithIndex / charges.NewCharge. (`intentsByType, err := input.Intents.ByType(); flatFees, err := s.flatFeeService.Create(ctx, flatfee.CreateInput{Intents: lo.Map(intentsByType.FlatFee,...)})`)
**Constructor registers billing hooks** — New() builds invoiceupdater.New(billingService,logger) and calls BillingService.RegisterStandardInvoiceHooks(standardInvoiceEventHandler); Config.Validate requires all ten dependencies non-nil. (`config.BillingService.RegisterStandardInvoiceHooks(standardInvoiceEventHandler)`)
**Earnings recognition after every state change** — Create/Advance/credit-purchase transitions/standard-invoice events call recognizeCustomerEarnings (deduped customer+currency via lo.Uniq) through the recognizer.Service. (`if err := s.recognizeCustomerEarnings(ctx, input.Customer, currencies...); err != nil { return nil, err }`)
**Auto-advance after create in a separate transaction** — Create commits charge state, then autoAdvanceCreatedCharges advances credit-only charges whose AdvanceAfter is already due in a fresh call so creation persists even if advancing fails. (`return s.autoAdvanceCreatedCharges(ctx, result.charges)`)
**Tax-code defaulting on create** — applyDefaultTaxCodes fills nil TaxCodeID per intent from taxCodeService.GetOrganizationDefaultTaxCodes — invoicing default for flat-fee/usage-based, credit-grant default for credit purchases. (`defaultID := defaults.InvoicingTaxCodeID; if intents[idx].Type()==meta.ChargeTypeCreditPurchase { defaultID = defaults.CreditGrantTaxCodeID }`)
**InvocableCharge adapter for patches** — ApplyPatches wraps each searched charge in flatFeeInvocableCharge/usageBasedInvocableCharge, calls TriggerPatch to collect invoiceupdater.Patch, then s.invoiceUpdater.ApplyPatches; credit-purchase is not invocable. (`result, err := invocableCharge.TriggerPatch(ctx, patch); invoicePatches = append(invoicePatches, result.InvoicePatches...)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | service struct, Config + Validate (10 required deps), New() with hook registration, validateNamespaceLockdown. | metaAdapter is used directly (no meta service yet); New must register standard invoice hooks or invoice-driven credit purchase transitions won't fire. |
| `create.go` | Create pipeline: tax defaults, per-type bulk create, gathering-line collection, credit-purchase invoice-now, recognition, auto-advance. | Credit purchases bypass collection alignment when ServicePeriod.From <= now (billed immediately); invoice-now runs AFTER the tx commits, not inside it. |
| `advance.go` | AdvanceCharges: lists non-final charges, advances flat-fee then usage-based (with customer override + resolved feature meters), recognizes earnings. | Nil advancedCharge from a sub-service means no transition — skip it; usage-based needs ResolveFeatureMeters before advancing. |
| `get.go` | GetByID/GetByIDs and expandChargesWithTypes which groups search items by type, fetches each sub-service, and reorders via entutils.InIDOrder. | GetByIDs runs in a transaction; missing IDs surface as NewChargeNotFoundError only in GetByID. |
| `patch.go` | ApplyPatches: applies per-charge patches then runs Creates LAST (a patch may free a UniqueReferenceID a create reuses). | Validates every charge is owned by the input customer+namespace before triggering patches. |
| `invoice.go` | Standard-invoice hook: processorByType dispatch keyed on StandardInvoice.Status (DraftCreated/Issued/PaymentAuthorized/Paid) into credit-purchase Post* handlers. | flatFee/usageBased processors are mostly noop today; nil processor returns error, not skip. Always recognizeCustomerEarnings after handling. |
| `helpers.go` | chargesByType bucketing and the InvocableCharge interface + flatFee/usageBased wrappers. | newInvocableCharges errors on duplicate charge IDs and on credit-purchase type (unsupported for patches). |
| `base_test.go` | BaseSuite wiring the full charges stack on top of billingtest.BaseSuite with test handlers; createMockChargeIntent builds flatfee/usagebased intents. | TearDownTest must reset handlers + MockStreamingConnector and clock.UnFreeze/ResetTime; line engines must be registered on BillingService before use. |

## Anti-Patterns

- Calling per-type adapters directly instead of the flatFee/usageBased/creditPurchase sub-services.
- Running Create's invoice-now or auto-advance inside the creation transaction.
- Skipping validateNamespaceLockdown on a new mutating method.
- Forgetting recognizeCustomerEarnings after a state transition.
- Adding charge types without extending ByType/chargesByType/expandChargesWithTypes/newInvocableCharges dispatch.

## Decisions

- **A single root service multiplexes three per-type charge services rather than callers picking a type.** — Charges are heterogeneous within one customer/subscription; the facade preserves intent order and centralizes recognition, tax defaults, and invoice projection.
- **Auto-advance and invoice-now run outside the create transaction.** — Creation state must persist even if a downstream advance/invoice fails, since a worker will retry advancement.
- **ApplyPatches creates new charges last.** — Deleting a charge can free a UniqueReferenceID that a subsequent create reuses, so ordering avoids collisions.

## Example: Mutating method shape: validate, lockdown, transactional fan-out, recognize

```
func (s *service) AdvanceCharges(ctx context.Context, input charges.AdvanceChargesInput) (charges.Charges, error) {
	if err := input.Validate(); err != nil { return nil, err }
	if err := s.validateNamespaceLockdown(input.Customer.Namespace); err != nil { return nil, err }
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (charges.Charges, error) {
		inScope, err := s.ListCharges(ctx, charges.ListChargesInput{Namespace: input.Customer.Namespace, StatusNotIn: []meta.ChargeStatus{meta.ChargeStatusFinal}, CustomerIDs: []string{input.Customer.ID}, Expands: meta.Expands{meta.ExpandRealizations}})
		if err != nil { return nil, err }
		// ... dispatch to flatFeeService / usageBasedService, then:
		if err := s.recognizeCustomerEarnings(ctx, input.Customer, currencies...); err != nil { return nil, err }
		return advancedCharges, nil
	})
}
```

<!-- archie:ai-end -->
