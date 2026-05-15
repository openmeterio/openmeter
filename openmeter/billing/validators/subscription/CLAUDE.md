# subscription

<!-- archie:ai-start -->

> Implements subscription.SubscriptionCommandHook to validate that a customer's billing app setup is valid whenever a subscription with billable items is created or updated — blocks subscription lifecycle transitions when the invoicing app cannot handle tax calculation, invoicing, or payment collection.

## Patterns

**Embed NoOpSubscriptionCommandHook** — Validator embeds subscription.NoOpSubscriptionCommandHook so only AfterCreate and AfterUpdate need implementation; all other hook methods default to no-op. (`type Validator struct { subscription.NoOpSubscriptionCommandHook; billingService billing.Service }`)
**Early-exit on non-billable subscription** — hasBillableItems checks whether any rate card in the subscription has a non-nil Price before calling the billing service, avoiding unnecessary network calls for free-tier subscriptions. (`if !v.hasBillableItems(view) { return nil }`)
**models.NewGenericConflictError wrapping** — Billing validation failures are wrapped in models.NewGenericConflictError, which maps to HTTP 409 in the error encoder chain. (`return models.NewGenericConflictError(fmt.Errorf("invalid billing setup: %w", err))`)
**Return interface not concrete type from constructor** — NewValidator returns subscription.SubscriptionCommandHook (interface), not *Validator, so callers depend on the interface, not the concrete type. (`func NewValidator(billingService billing.Service) (subscription.SubscriptionCommandHook, error)`)
**Capability slice validation** — Required app capabilities are passed as a slice to ValidateCustomer, keeping validation app-agnostic and easy to extend without changing control flow. (`customerApp.ValidateCustomer(ctx, customerProfile.Customer, []app.CapabilityType{app.CapabilityTypeCalculateTax, app.CapabilityTypeInvoiceCustomers, app.CapabilityTypeCollectPayments})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `validator.go` | Single-file package: Validator struct, NewValidator, AfterCreate, AfterUpdate, validateBillingSetup, hasBillableItems. Both AfterCreate and AfterUpdate delegate to validateBillingSetup, keeping the logic DRY. | customerapp.AsCustomerApp type-asserts the app base to a customer-app interface — if the app type changes, this assertion fails at runtime. The Expand field in GetCustomerOverrideInput must include both Apps: true and Customer: true; missing either causes a nil-dereference or incomplete check. |

## Anti-Patterns

- Overriding BeforeCreate/BeforeDelete without strong justification — post-create/post-update validation is intentional so the billing check runs against the fully persisted subscription view.
- Returning a plain error instead of models.NewGenericConflictError for billing setup failures — the HTTP layer expects a 409 error type.
- Calling subscription service or Ent adapters directly — all billing state reads must go through billing.Service.
- Adding app-type-specific branching (e.g., 'if Stripe do X') — capability validation is intentionally app-agnostic via the CapabilityType slice.
- Using context.Background() instead of propagating the incoming ctx.

## Decisions

- **Validator is registered with subscription.Service.RegisterHook() at wiring time in app/common, not via direct import.** — Billing imports subscription; subscription cannot import billing. The hook registration at wire time inverts the dependency without an import cycle.
- **Validation runs after create/update (AfterCreate, AfterUpdate) rather than before.** — The billing service needs the fully persisted subscription view (with all phases and items resolved) to evaluate billability; a before-hook would not have this complete view.

## Example: Adding a new required billing capability check (e.g., require CapabilityTypeRefundPayments)

```
// In validator.go, extend the capability slice in validateBillingSetup:
return customerApp.ValidateCustomer(ctx, customerProfile.Customer, []app.CapabilityType{
    app.CapabilityTypeCalculateTax,
    app.CapabilityTypeInvoiceCustomers,
    app.CapabilityTypeCollectPayments,
    app.CapabilityTypeRefundPayments, // new capability
})
```

<!-- archie:ai-end -->
