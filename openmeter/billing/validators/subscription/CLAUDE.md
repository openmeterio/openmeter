# subscription

<!-- archie:ai-start -->

> Implements subscription.SubscriptionCommandHook to validate that a customer's billing app setup is valid whenever a subscription with billable items is created or updated — blocks lifecycle transitions when the invoicing app cannot handle tax calculation, invoicing, or payment collection.

## Patterns

**Embed NoOpSubscriptionCommandHook** — Validator embeds subscription.NoOpSubscriptionCommandHook so only AfterCreate/AfterUpdate need implementation. (`type Validator struct { subscription.NoOpSubscriptionCommandHook; billingService billing.Service }`)
**Early-exit on non-billable subscription** — hasBillableItems checks whether any rate card has a non-nil Price before calling the billing service, avoiding network calls for free-tier subscriptions. (`if !v.hasBillableItems(view) { return nil }`)
**Conflict error wrapping** — Billing validation failures wrap in models.NewGenericConflictError, mapping to HTTP 409. (`return models.NewGenericConflictError(fmt.Errorf("invalid billing setup: %w", err))`)
**Return interface from constructor** — NewValidator returns subscription.SubscriptionCommandHook (interface), not *Validator. (`func NewValidator(billingService billing.Service) (subscription.SubscriptionCommandHook, error)`)
**Capability slice validation** — Required app capabilities are passed as a slice to ValidateCustomer, keeping validation app-agnostic. (`customerApp.ValidateCustomer(ctx, customer, []app.CapabilityType{app.CapabilityTypeCalculateTax, app.CapabilityTypeInvoiceCustomers, app.CapabilityTypeCollectPayments})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `validator.go` | Single-file package: Validator, NewValidator, AfterCreate, AfterUpdate, validateBillingSetup, hasBillableItems. Both After* delegate to validateBillingSetup. | customerapp.AsCustomerApp type-asserts the app base — fails at runtime if the app type changes; GetCustomerOverrideInput.Expand must set both Apps:true and Customer:true. |

## Anti-Patterns

- Overriding BeforeCreate/BeforeDelete without strong justification — post-create/update validation runs against the fully persisted view intentionally.
- Returning a plain error instead of models.NewGenericConflictError for billing setup failures.
- Calling subscription service or Ent adapters directly — billing reads go through billing.Service.
- Adding app-type-specific branching — capability validation is intentionally app-agnostic.
- Using context.Background() instead of propagating the incoming ctx.

## Decisions

- **Validator registers via subscription.Service.RegisterHook() at wiring time in app/common, not via direct import.** — Billing imports subscription; subscription cannot import billing. Wire-time registration inverts the dependency without a cycle.
- **Validation runs after create/update, not before.** — The billing service needs the fully persisted subscription view (all phases/items resolved) to evaluate billability; a before-hook lacks this view.

## Example: Add a new required billing capability check

```
// In validateBillingSetup, extend the capability slice:
return customerApp.ValidateCustomer(ctx, customerProfile.Customer, []app.CapabilityType{
    app.CapabilityTypeCalculateTax,
    app.CapabilityTypeInvoiceCustomers,
    app.CapabilityTypeCollectPayments,
    app.CapabilityTypeRefundPayments, // new capability
})
```

<!-- archie:ai-end -->
