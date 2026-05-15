# customer

<!-- archie:ai-start -->

> Implements customer.RequestValidator to enforce subscription-aware pre-conditions on customer mutations — blocks deletes when active subscriptions exist and prevents subject-key changes when a customer has active subscriptions. Registered at wiring time via app/common to avoid circular imports.

## Patterns

**NoopRequestValidator embedding** — Embed customer.NoopRequestValidator so only the overridden methods need implementation; unimplemented lifecycle methods default to no-op. (`type Validator struct { customer.NoopRequestValidator; subscriptionService subscription.Service; customerService customer.Service }`)
**Interface assertion at package level** — Declare var _ customer.RequestValidator = (*Validator)(nil) at package top for compile-time interface satisfaction proof. (`var _ customer.RequestValidator = (*Validator)(nil)`)
**Constructor nil-guard** — NewValidator returns (*Validator, error) and validates every dependency is non-nil before construction. (`if subscriptionService == nil { return nil, fmt.Errorf("subscription service is required") }`)
**Use clock.Now() for time-sensitive queries** — Pass lo.ToPtr(clock.Now()) to subscription.ListSubscriptionsInput.ActiveAt so tests can control time via the clock package. (`ActiveAt: lo.ToPtr(clock.Now())`)
**Delegate input validation to input type** — Call input.Validate() first inside each handler before any domain queries to keep validation self-contained. (`if err := input.Validate(); err != nil { return err }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `validator.go` | Single file implementing the customer.RequestValidator interface with ValidateUpdateCustomer and ValidateDeleteCustomer logic. | ValidateUpdateCustomer fetches the current customer only when UsageAttribution.SubjectKeys is non-nil — skipping this branch for other mutations is intentional. Error types must use models.NewGenericNotFoundError / models.NewGenericPreConditionFailedError for correct HTTP status mapping. |

## Anti-Patterns

- Returning plain fmt.Errorf for not-found or pre-condition errors — must use models.NewGenericNotFoundError / models.NewGenericPreConditionFailedError so HTTP mapping works correctly
- Using context.Background() instead of the caller-supplied ctx
- Calling billing or other non-subscription services from this validator — would create import cycles
- Checking active subscriptions without the ActiveAt filter — would return already-cancelled subscriptions
- Registering the validator inside the subscription or customer package constructors — must be done in app/common to avoid circular imports

## Decisions

- **Validator is registered via customer.Service.RegisterRequestValidator() in app/common wiring, not hardcoded in the customer package.** — Avoids circular import between subscription and customer packages; subscription registers the constraint at startup through the registry pattern.
- **Only ValidateUpdateCustomer and ValidateDeleteCustomer are overridden; ValidateCreateCustomer remains a no-op via embedding.** — Customer creation has no subscription pre-conditions; embedding NoopRequestValidator avoids boilerplate.

## Example: Register the validator during app wiring (app/common)

```
import customer "github.com/openmeterio/openmeter/openmeter/subscription/validators/customer"

v, err := customer.NewValidator(subscriptionService, customerService)
if err != nil { return err }
customerService.RegisterRequestValidator(v)
```

<!-- archie:ai-end -->
