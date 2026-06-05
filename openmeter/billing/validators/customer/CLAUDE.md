# customer

<!-- archie:ai-start -->

> Cross-domain guard that hooks into the customer domain's request-validation pipeline to block deletion of a customer while any of its invoices are still gathering or non-final. Implements customer.RequestValidator by embedding customer.NoopRequestValidator and overriding only ValidateDeleteCustomer.

## Patterns

**RequestValidator via Noop embedding** — Validator embeds customer.NoopRequestValidator and overrides only the hooks it cares about; a compile-time assertion enforces interface conformance. (`var _ customer.RequestValidator = (*Validator)(nil); type Validator struct { customer.NoopRequestValidator; ... }`)
**Constructor nil-checks injected services** — NewValidator returns (*Validator, error) and fails fast with fmt.Errorf when billingService or syncService is nil instead of panicking. (`if billingService == nil { return nil, fmt.Errorf("billing service is required") }`)
**Sync-then-check invoice state** — Before listing invoices, recently-active subscriptions are force-synced via syncService.SyncByID so gathering invoices reflect current target state; only then are invoice states asserted. (`v.syncService.SyncByID(ctx, sub.NamespacedID, time.Now())`)
**Aggregate validation errors with errors.Join** — Each blocking invoice appends to errs []error; the method returns errors.Join(errs...) rather than returning on the first offending invoice. (`errs = append(errs, fmt.Errorf("invoice %s is not in final state...", stdInvoice.ID)); return errors.Join(errs...)`)
**Invoice type discrimination via As* accessors** — Use inv.Type() against billing.InvoiceTypeGathering and the AsGatheringInvoice/AsStandardInvoice accessors rather than reading struct fields directly; gathering invoices with DeletedAt set are skipped. (`if inv.Type() == billing.InvoiceTypeGathering { g, _ := inv.AsGatheringInvoice() } else { s, _ := inv.AsStandardInvoice() }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `customer.go` | Sole file: defines Validator, NewValidator, and ValidateDeleteCustomer enforcing the all-invoices-final precondition for customer deletion. | The 24h watermark (time.Now().Add(-24*time.Hour)) gates which subscriptions get re-synced — subs that are inactive, deleted, or whose ActiveTo is older than the watermark are still synced via the ActiveTo==nil/DeletedAt!=nil/Before checks; changing this logic affects deletion correctness. Deleted gathering invoices (DeletedAt!=nil) are intentionally skipped, not blocking. |

## Anti-Patterns

- Returning early on the first non-final invoice instead of collecting all into errs and returning errors.Join — callers expect the full list of blockers.
- Skipping the subscriptionsync step before listing invoices — without it gathering invoices may be stale and deletion validation gives wrong results.
- Reading invoice fields directly instead of going through inv.Type()/AsGatheringInvoice/AsStandardInvoice.
- Constructing Validator literally instead of via NewValidator, bypassing the nil dependency guards.

## Decisions

- **Live in the billing module but implement a customer-domain interface (RequestValidator).** — The rule (no delete with open invoices) is a billing invariant but must fire inside the customer delete flow; placing it here keeps the billing dependency direction outward and wires in via customer's hook mechanism.

## Example: Blocking customer deletion when invoices are not final

```
func (v *Validator) ValidateDeleteCustomer(ctx context.Context, input customer.DeleteCustomerInput) error {
	if err := input.Validate(); err != nil { return err }
	invoices, err := v.billingService.ListInvoices(ctx, billing.ListInvoicesInput{Namespaces: []string{input.Namespace}, Customers: []string{input.ID}})
	if err != nil { return err }
	errs := make([]error, 0, len(invoices.Items))
	for _, inv := range invoices.Items {
		if inv.Type() == billing.InvoiceTypeGathering { /* skip if deleted, else block */ }
		stdInvoice, err := inv.AsStandardInvoice()
		if err != nil { return err }
		if !stdInvoice.Status.IsFinal() { errs = append(errs, fmt.Errorf("invoice %s is not in final state", stdInvoice.ID)) }
	}
	return errors.Join(errs...)
}
```

<!-- archie:ai-end -->
