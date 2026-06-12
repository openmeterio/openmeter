# app

<!-- archie:ai-start -->

> Defines the customer.App integration contract — the seam through which app providers (Stripe, sandbox, custom-invoicing) validate that they can run for a given customer.

## Patterns

**Capability-aware App interface** — App declares a single ValidateCustomer(ctx, *customer.Customer, []app.CapabilityType) error; provider apps implement it to gate per-customer execution. (`type App interface { ValidateCustomer(ctx context.Context, customer *customer.Customer, capabilities []app.CapabilityType) error }`)
**Type-asserting adapter from app.App** — AsCustomerApp(app.App) narrows a generic app.App to the customer App via a type assertion and returns a GenericValidationError when the cast fails, including id/type for diagnostics. (`customerApp, ok := candidate.(App); if !ok { return nil, models.NewGenericValidationError(fmt.Errorf("is not a customer app [id=%s, type=%s]", candidate.GetID(), candidate.GetType())) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `app.go` | Declares the App interface and the AsCustomerApp narrowing helper (package customerapp). | Package name is customerapp, not app; failed assertion must return models.NewGenericValidationError, not a bare error. |

## Anti-Patterns

- Putting persistence or HTTP logic in this package — it is purely an integration contract
- Returning a plain error from AsCustomerApp instead of a GenericValidationError

## Decisions

- **Keep customer app integration as a thin interface separate from openmeter/app** — Lets provider apps (stripe/sandbox/custominvoicing) depend on a minimal customer-validation contract without importing the full customer service.

## Example: Narrowing a generic app to a customer app

```
func AsCustomerApp(customerAppCandidate app.App) (App, error) {
	customerApp, ok := customerAppCandidate.(App)
	if !ok {
		return nil, models.NewGenericValidationError(
			fmt.Errorf("is not a customer app [id=%s, type=%s]", customerAppCandidate.GetID(), customerAppCandidate.GetType()),
		)
	}
	return customerApp, nil
}
```

<!-- archie:ai-end -->
