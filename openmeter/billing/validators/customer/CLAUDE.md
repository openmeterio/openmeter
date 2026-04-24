# customer

<!-- archie:ai-start -->

> Implements customer.RequestValidator to enforce billing pre-conditions before customer mutations — specifically, blocks deletion of customers with non-final invoices or active gathering invoices. It is the cross-domain guard that prevents customer removal from leaving orphaned billing state.

## Patterns

**Embed NoopRequestValidator** — Validator embeds customer.NoopRequestValidator so only the methods it overrides need implementation; unimplemented validator methods default to no-op without breaking the interface. (`type Validator struct { customer.NoopRequestValidator; billingService billing.Service; ... }`)
**Compile-time interface assertion** — var _ customer.RequestValidator = (*Validator)(nil) at package level ensures the struct satisfies the interface at compile time. (`var _ customer.RequestValidator = (*Validator)(nil)`)
**Sync-before-validate pattern** — ValidateDeleteCustomer calls SynchronizeSubscription for active subscriptions before checking invoice state, ensuring any pending subscription activity is reflected in invoices before the gate check runs. (`v.syncService.SynchronizeSubscription(ctx, view, time.Now())`)
**Nil-guard constructors** — NewValidator returns an error if any required service dependency is nil, enforcing required deps at construction time rather than at call time. (`if billingService == nil { return nil, fmt.Errorf("billing service is required") }`)
**errors.Join for multi-invoice errors** — All invoice validation failures are accumulated into a slice and joined with errors.Join, so the caller receives all blocking invoices at once. (`return errors.Join(errs...)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `customer.go` | Single-file package: defines Validator struct, NewValidator constructor, and ValidateDeleteCustomer — the only override of NoopRequestValidator. | The 24-hour watermark for active subscription detection (time.Now().Add(-24 * time.Hour)) is a business rule embedded inline; changing billing windows requires updating this constant. Gathering invoices with non-nil DeletedAt are skipped (soft-deleted); standard invoices must have IsFinal() == true. |

## Anti-Patterns

- Adding ValidateCreateCustomer or ValidateUpdateCustomer overrides here without a billing-domain reason — customer package owns general lifecycle, this package only guards billing pre-conditions.
- Calling billing adapter or Ent directly instead of going through billing.Service — all reads must go through the service interface.
- Skipping the sync step before invoice validation — without SynchronizeSubscription, pending subscription charges may not be reflected and the gate check will be incorrect.
- Using context.Background() instead of propagating the incoming ctx parameter.

## Decisions

- **Validator registers itself with customer.Service.RegisterRequestValidator() at wiring time (in app/common), not at compile time.** — Avoids import cycle: billing depends on customer, so customer cannot import billing. The registration hook inverts the dependency at runtime.
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
