# app

<!-- archie:ai-start -->

> Thin integration contract between the customer domain and the app (marketplace) domain — defines the optional App interface (ValidateCustomer) and the AsCustomerApp type-assertion helper. No state, no DB access, no business logic.

## Patterns

**Optional interface via runtime type assertion** — AsCustomerApp performs a runtime type assertion on app.App; if the concrete type does not implement customerapp.App it returns models.NewGenericValidationError rather than panicking. (`customerApp, ok := customerAppCandidate.(App); if !ok { return nil, models.NewGenericValidationError(fmt.Errorf("not a customer app [id=%s]", ...)) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `app.go` | Declares customerapp.App interface (ValidateCustomer) and AsCustomerApp helper. Zero state. | Must not import openmeter/customer/adapter or openmeter/customer/service — only openmeter/app and openmeter/customer domain types. |

## Anti-Patterns

- Panicking on failed type assertion instead of returning models.NewGenericValidationError.
- Adding business logic, DB calls, or state to this package — it is purely an interface/assertion layer.
- Importing openmeter/customer/adapter or openmeter/customer/service — creates import cycles.

## Decisions

- **Separate package rather than adding ValidateCustomer to the core app.App interface.** — Not all apps need customer validation; the opt-in type assertion avoids polluting the core app.App interface for every new capability.

<!-- archie:ai-end -->
