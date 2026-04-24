# app

<!-- archie:ai-start -->

> Thin integration contract between the customer domain and the app (marketplace) domain — defines the App interface for validating whether an installed app can operate on a specific customer, and provides a type-assertion helper AsCustomerApp.

## Patterns

**Type-assertion helper for optional interface** — AsCustomerApp performs a runtime type assertion on app.App and returns models.NewGenericValidationError if the concrete type does not implement customerapp.App — callers must handle the error rather than panicking. (`customerApp, ok := customerAppCandidate.(App); if !ok { return nil, models.NewGenericValidationError(...) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `app.go` | Declares the customerapp.App interface (ValidateCustomer) and AsCustomerApp helper. No state, no DB access. | This package must not import openmeter/customer/adapter or openmeter/customer/service — it only imports openmeter/app and openmeter/customer (domain types). |

## Anti-Patterns

- Adding business logic or DB calls to this package — it is a pure interface/assertion layer.
- Panicking on failed type assertion instead of returning models.NewGenericValidationError.

## Decisions

- **Separate package instead of adding ValidateCustomer to app.App directly.** — Not all apps need customer validation; the interface is opt-in via type assertion, avoiding changes to the core app.App interface for every new capability.

<!-- archie:ai-end -->
