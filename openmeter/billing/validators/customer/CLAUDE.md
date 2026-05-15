# customer

<!-- archie:ai-start -->

> Implements customer.RequestValidator to enforce billing pre-conditions before customer deletion — blocks removal of customers with non-final standard invoices or active (non-soft-deleted) gathering invoices. It is the cross-domain guard preventing customer removal from leaving orphaned billing state.

## Patterns

**Embed NoopRequestValidator** — Validator embeds customer.NoopRequestValidator so only ValidateDeleteCustomer needs implementation; all other RequestValidator methods default to no-op without breaking the interface. (`type Validator struct { customer.NoopRequestValidator; billingService billing.Service; syncService subscriptionsync.Service; subscriptionService subscription.Service }`)
**Compile-time interface assertion** — var _ customer.RequestValidator = (*Validator)(nil) at package level ensures the struct satisfies the interface at compile time. (`var _ customer.RequestValidator = (*Validator)(nil)`)
**Sync-before-validate pattern** — ValidateDeleteCustomer syncs active subscriptions via SynchronizeSubscription before checking invoice state, ensuring pending charges are reflected before the gate check. (`v.syncService.SynchronizeSubscription(ctx, view, time.Now())`)
**Nil-guard constructors** — NewValidator returns an error if any required service dependency is nil, enforcing required deps at construction time rather than at call time. (`if billingService == nil { return nil, fmt.Errorf("billing service is required") }`)
**errors.Join for multi-invoice failures** — All invoice validation failures accumulate into a slice and are joined with errors.Join so the caller receives all blocking invoices at once. (`errs = append(errs, fmt.Errorf("invoice %s is not in final state", stdInvoice.ID)); return errors.Join(errs...)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `customer.go` | Single-file package: defines Validator struct, NewValidator constructor, and ValidateDeleteCustomer — the only override of NoopRequestValidator. | The 24-hour watermark (time.Now().Add(-24 * time.Hour)) is an embedded business rule for active subscription detection. Gathering invoices with non-nil DeletedAt are skipped (soft-deleted). Standard invoices must have IsFinal() == true. All reads go through billing.Service and subscription.Service — never the adapters directly. |

## Anti-Patterns

- Adding ValidateCreateCustomer or ValidateUpdateCustomer overrides without a billing-domain reason — this package only guards billing pre-conditions on delete.
- Calling billing adapter or Ent directly instead of going through billing.Service — all reads must go through the service interface.
- Skipping the SynchronizeSubscription step before invoice validation — without it, pending subscription charges may not be reflected and the gate check will be incorrect.
- Using context.Background() instead of propagating the incoming ctx parameter.
- Returning a plain error for billing blocks instead of letting errors.Join accumulate all failures — callers need all blocking invoices at once.

## Decisions

- **Validator registers with customer.Service.RegisterRequestValidator() at wiring time in app/common, not via direct import.** — Billing depends on customer; customer cannot import billing. Registration at wire time inverts the dependency without an import cycle.
- **Subscription sync is driven inside the validator rather than as a pre-hook in the subscription service.** — The validator needs a billing-consistent snapshot of invoices; syncing at validation time ensures the check is atomic with respect to the delete.

## Example: Adding a new pre-delete billing check (e.g., block deletion if open credit notes exist)

```
// In customer.go, extend ValidateDeleteCustomer after the existing invoice loop:
creditNotes, err := v.billingService.ListCreditNotes(ctx, billing.ListCreditNotesInput{
    Namespaces: []string{input.Namespace},
    Customers:  []string{input.ID},
})
if err != nil {
    return err
}
for _, cn := range creditNotes.Items {
    if !cn.Status.IsFinal() {
        errs = append(errs, fmt.Errorf("credit note %s is not final", cn.ID))
    }
}
return errors.Join(errs...)
```

<!-- archie:ai-end -->
