# customer

<!-- archie:ai-start -->

> Implements customer.RequestValidator to enforce billing pre-conditions before customer deletion — blocks removal of customers with non-final standard invoices or active (non-soft-deleted) gathering invoices. The cross-domain guard that prevents customer removal from leaving orphaned billing state.

## Patterns

**Embed NoopRequestValidator** — Validator embeds customer.NoopRequestValidator so only ValidateDeleteCustomer needs implementation; other methods default to no-op. (`type Validator struct { customer.NoopRequestValidator; billingService billing.Service; syncService subscriptionsync.Service; subscriptionService subscription.Service }`)
**Compile-time interface assertion** — Package-level var _ check ensures the struct satisfies the interface. (`var _ customer.RequestValidator = (*Validator)(nil)`)
**Sync-before-validate** — ValidateDeleteCustomer syncs active subscriptions via SynchronizeSubscription before checking invoice state so pending charges are reflected. (`v.syncService.SynchronizeSubscription(ctx, view, time.Now())`)
**Nil-guard constructors** — NewValidator returns an error if any required service dependency is nil — enforced at construction, not call time. (`if billingService == nil { return nil, fmt.Errorf("billing service is required") }`)
**errors.Join for multi-invoice failures** — Invoice validation failures accumulate into a slice and join via errors.Join so the caller gets all blocking invoices at once. (`errs = append(errs, fmt.Errorf("invoice %s is not in final state", stdInvoice.ID)); return errors.Join(errs...)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `customer.go` | Single-file package: Validator struct, NewValidator, and ValidateDeleteCustomer (the only NoopRequestValidator override). | 24h watermark (time.Now().Add(-24*time.Hour)) is an embedded rule for active-subscription detection; soft-deleted gathering invoices (DeletedAt != nil) are skipped; standard invoices must be IsFinal(); all reads go through billing.Service/subscription.Service, never adapters. |

## Anti-Patterns

- Adding ValidateCreateCustomer/ValidateUpdateCustomer overrides without a billing reason — this package only guards delete.
- Calling billing adapter or Ent directly instead of billing.Service.
- Skipping SynchronizeSubscription before invoice validation — gate check becomes incorrect.
- Using context.Background() instead of propagating the incoming ctx.
- Returning a plain error instead of accumulating with errors.Join — callers need all blocking invoices.

## Decisions

- **Validator registers via customer.Service.RegisterRequestValidator() at wiring time in app/common, not via direct import.** — Billing depends on customer; customer cannot import billing. Wire-time registration inverts the dependency without a cycle.
- **Subscription sync is driven inside the validator, not as a subscription-service pre-hook.** — The validator needs a billing-consistent invoice snapshot; syncing at validation time makes the check atomic with the delete.

## Example: Add a new pre-delete billing check (block deletion if open credit notes exist)

```
creditNotes, err := v.billingService.ListCreditNotes(ctx, billing.ListCreditNotesInput{
    Namespaces: []string{input.Namespace}, Customers: []string{input.ID},
})
if err != nil { return err }
for _, cn := range creditNotes.Items {
    if !cn.Status.IsFinal() {
        errs = append(errs, fmt.Errorf("credit note %s is not final", cn.ID))
    }
}
return errors.Join(errs...)
```

<!-- archie:ai-end -->
