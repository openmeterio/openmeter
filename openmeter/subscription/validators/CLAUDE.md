# validators

<!-- archie:ai-start -->

> Organisational folder for cross-domain validators that enforce subscription pre-conditions on other domains' mutations: validators/customer blocks customer delete/subject-key change when active subscriptions exist (customer.RequestValidator), and validators/subscription enforces the unique-subscription-per-customer invariant (SubscriptionCommandHook). Both are registered via their service's RegisterRequestValidator/RegisterHook in app/common wiring to avoid circular imports.

## Patterns

**Embed noop base types** — Both validators embed their noop base (NoopRequestValidator / NoOpSubscriptionCommandHook) so only the relevant methods are overridden and interface additions don't break compilation. (`type customerValidator struct { customer.NoopRequestValidator; subSvc subscription.Service }`)
**Register in app/common, not the domain package** — Validators register via customer.Service.RegisterRequestValidator() and subscription.Service.RegisterHook() in app/common wiring, keeping circular-import boundaries clean. (`customerSvc.RegisterRequestValidator(subscriptionvalidators.NewCustomerValidator(subSvc))`)
**models.NewGeneric* errors for HTTP mapping** — Validator errors use models.NewGenericPreConditionFailedError or NewGenericNotFoundError so the HTTP layer maps them to 412 / 404. (`return models.NewGenericPreConditionFailedError(fmt.Errorf("customer has active subscriptions"))`)
**ActiveAt/ActiveInPeriod filter + pagination.CollectAll** — Subscription queries must include ActiveAt/ActiveInPeriod (to exclude cancelled subs) and use pagination.CollectAll to avoid silently missing subscriptions beyond the first page. (`subs, err := pagination.CollectAll(func(p pagination.Page) (pagination.Result[subscription.Subscription], error) { return svc.List(ctx, subscription.ListSubscriptionsInput{CustomerID: id, ActiveAt: &now}) })`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `customer/validator.go` | Blocks customer delete/subject-key update when active subscriptions exist; uses clock.Now() for the ActiveAt filter. | Must filter by ActiveAt — without it cancelled subscriptions are returned and block valid deletes. |
| `subscription/validator.go` | Enforces the unique-subscription invariant before Create/Update; feature-flag-gated per-subscription vs per-feature uniqueness. | Must exclude the current subscription ID on Update; must use pagination.CollectAll to avoid missing subscriptions beyond the first page. |
| `subscription/components.go` | Pure utility functions for subscription period and feature overlap checks. | Business-logic decisions belong in validator.go, not here. |

## Anti-Patterns

- Calling subscription.Service write methods from within a SubscriptionCommandHook — creates re-entrant hook calls.
- Using context.Background() instead of the caller-supplied ctx.
- Returning plain fmt.Errorf instead of models.NewGenericPreConditionFailedError — breaks HTTP status mapping.
- Querying subscriptions without the ActiveAt/ActiveInPeriod filter — returns all-time subscriptions and breaks the check.
- Registering the validator inside the subscription or customer package constructors — must be done in app/common.

## Decisions

- **Validators placed in a dedicated validators/ sub-folder.** — Avoids circular imports: subscription depends on customer, so subscription validators of customer cannot live in the subscription package root.
- **Feature flag (MultiSubscriptionEnabledFF) selects uniqueness semantics.** — Allows per-feature rather than per-subscription uniqueness for tenants needing multiple concurrent subscriptions.

<!-- archie:ai-end -->
