# subscription

<!-- archie:ai-start -->

> Implements subscription.SubscriptionCommandHook to validate that a customer's billing app setup is valid whenever a subscription with billable items is created or updated — blocks subscription lifecycle transitions when the invoicing app cannot handle tax calculation, invoicing, or payment collection.

## Patterns

**Embed NoOpSubscriptionCommandHook** — Validator embeds subscription.NoOpSubscriptionCommandHook so only AfterCreate and AfterUpdate need to be implemented; all other hook methods default to no-op. (`type Validator struct { subscription.NoOpSubscriptionCommandHook; billingService billing.Service }`)
**Early-exit on non-billable subscription** — hasBillableItems checks whether any rate card in the subscription has a non-nil Price before calling the billing service, avoiding unnecessary network calls for free-tier subscriptions. (`if !v.hasBillableItems(view) { return nil }`)
**models.NewGenericConflictError wrapping** — Billing validation failures are wrapped in models.NewGenericConflictError, which maps to HTTP 409 in the error encoder chain. (`return models.NewGenericConflictError(fmt.Errorf("invalid billing setup: %w", err))`)
**Nil-guard constructors returning hook interface** — NewValidator returns subscription.SubscriptionCommandHook (interface), not *Validator, so callers depend on the interface not the concrete type. (`func NewValidator(billingService billing.Service) (subscription.SubscriptionCommandHook, error)`)
**Capability slice validation** — App capability requirements are passed as a slice to ValidateCustomer, making it easy to add or remove required capabilities without changing the validation logic. (`customerApp.ValidateCustomer(ctx, customerProfile.Customer, []app.CapabilityType{app.CapabilityTypeCalculateTax, app.CapabilityTypeInvoiceCustomers, app.CapabilityTypeCollectPayments})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `validator.go` | Single-file package: Validator struct, NewValidator, AfterCreate, AfterUpdate, validateBillingSetup, hasBillableItems. Both AfterCreate and AfterUpdate delegate to the same validateBillingSetup, keeping the logic DRY. | customerapp.AsCustomerApp type-asserts the app base to a customer-app interface — if the app type changes, this assertion will fail at runtime. The Expand: billing.CustomerOverrideExpand{Apps: true, Customer: true} field must include both Apps and Customer for the validation to work; missing either causes a nil-dereference or incomplete check. |

## Anti-Patterns

- Overriding BeforeCreate/BeforeDelete without strong justification — the pattern here is post-create/post-update validation so billing state is checked after subscription state is persisted.
- Returning a plain error instead of models.NewGenericConflictError for billing setup failures — the HTTP layer expects a specific error type for 409 responses.
- Calling subscription service or Ent adapters directly — all billing state reads must go through billing.Service.
- Adding app-type-specific branching (e.g., 'if Stripe do X') — capability validation is intentionally app-agnostic via the CapabilityType slice.
- Using context.Background() instead of propagating the incoming ctx.

## Decisions

- **Validator is registered with subscription.Service.RegisterHook() at wiring time in app/common, not via direct import.** — Billing imports subscription; subscription cannot import billing. The hook registration at wire time inverts the dependency.
- **Validation runs after create/update (AfterCreate, AfterUpdate) rather than before.** — The billing service needs the persisted subscription view (with all phases and items resolved) to evaluate billability; a before-hook would not have this complete view.

<!-- archie:ai-end -->
