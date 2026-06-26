# subscription

<!-- archie:ai-start -->

> Subscription command hook that, after a subscription is created or updated, validates the customer's billing setup is capable of invoicing whenever the subscription has billable (priced) ratecards. Implements subscription.SubscriptionCommandHook.

## Patterns

**SubscriptionCommandHook via NoOp embedding** — Validator embeds subscription.NoOpSubscriptionCommandHook and overrides only AfterCreate/AfterUpdate; NewValidator returns the interface type, not the concrete struct. (`type Validator struct { subscription.NoOpSubscriptionCommandHook; billingService billing.Service }; func NewValidator(...) (subscription.SubscriptionCommandHook, error)`)
**Skip validation when no billable items** — validateBillingSetup short-circuits via hasBillableItems(view), which walks view.Phases -> ItemsByKey -> items and returns true only if a ratecard's AsMeta().Price is non-nil. (`if !v.hasBillableItems(view) { return nil }`)
**Wrap validation failures as conflict errors** — AfterCreate/AfterUpdate wrap any billing-setup error in models.NewGenericConflictError so the API surfaces a 409-style conflict rather than a raw error. (`return models.NewGenericConflictError(fmt.Errorf("invalid billing setup: %w", err))`)
**Capability-based customer validation** — Resolves the customer's invoicing app via GetCustomerOverride (Expand Apps+Customer) and customerapp.AsCustomerApp, then calls ValidateCustomer requiring CalculateTax, InvoiceCustomers, and CollectPayments capabilities. (`customerApp.ValidateCustomer(ctx, customerProfile.Customer, []app.CapabilityType{app.CapabilityTypeCalculateTax, app.CapabilityTypeInvoiceCustomers, app.CapabilityTypeCollectPayments})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `validator.go` | Sole file: Validator, NewValidator, AfterCreate/AfterUpdate hooks, and the validateBillingSetup/hasBillableItems helpers. | The required capability set is hardcoded and commented as Stripe-only (CalculateTax + InvoiceCustomers + CollectPayments); adding payment providers means changing this list. AsCustomerApp failure is treated as should-not-happen but still propagated. Validator methods have value receivers (v Validator), so they must stay stateless. |

## Anti-Patterns

- Running ValidateCustomer for subscriptions without priced ratecards — hasBillableItems must gate it to avoid rejecting free subscriptions.
- Returning the raw billing error from AfterCreate/AfterUpdate instead of wrapping in models.NewGenericConflictError, losing the conflict semantics.
- Adding mutable fields and using value receivers — the hook is invoked as a value and must remain stateless.
- Bypassing GetCustomerOverride's Expand{Apps,Customer} when resolving the invoicing app, which would leave MergedProfile.Apps.Invoicing unpopulated.

## Decisions

- **Validate billing setup as a subscription AfterCreate/AfterUpdate hook rather than inside the subscription service.** — Keeps billing-specific capability checks out of the subscription core while still failing creation/update of subscriptions that cannot be billed for the customer's current provider setup.

## Example: Validating a customer can be billed when a subscription has priced ratecards

```
func (v Validator) validateBillingSetup(ctx context.Context, view subscription.SubscriptionView) error {
	if !v.hasBillableItems(view) { return nil }
	customerProfile, err := v.billingService.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
		Customer: customer.CustomerID{Namespace: view.Subscription.Namespace, ID: view.Subscription.CustomerId},
		Expand:   billing.CustomerOverrideExpand{Apps: true, Customer: true},
	})
	if err != nil { return err }
	customerApp, err := customerapp.AsCustomerApp(customerProfile.MergedProfile.Apps.Invoicing)
	if err != nil { return err }
	return customerApp.ValidateCustomer(ctx, customerProfile.Customer, []app.CapabilityType{
		app.CapabilityTypeCalculateTax, app.CapabilityTypeInvoiceCustomers, app.CapabilityTypeCollectPayments,
	})
}
```

<!-- archie:ai-end -->
