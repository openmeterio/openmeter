# validators

<!-- archie:ai-start -->

> Organisational folder for cross-domain validators that enforce subscription pre-conditions on other domain mutations. Contains two sub-packages: validators/customer (blocks customer delete/update when active subscriptions exist) and validators/subscription (unique-subscription-per-customer invariant as a SubscriptionCommandHook). Both are registered via their respective service's RegisterHook/RegisterRequestValidator in app/common wiring.

## Patterns

**Embed noop base types** — Both validators embed their respective noop base (NoopRequestValidator / NoOpSubscriptionCommandHook) so only the relevant methods are overridden, future interface additions don't break compilation. (`type customerValidator struct { customer.NoopRequestValidator; subSvc subscription.Service }`)
**Register in app/common, not in domain package** — Validators are registered via customer.Service.RegisterRequestValidator() and subscription.Service.RegisterHook() in app/common wiring, keeping circular-import boundaries clean. (`customerSvc.RegisterRequestValidator(subscriptionvalidators.NewCustomerValidator(subSvc))`)
**Use models.NewGeneric* errors for HTTP mapping** — All validator errors use models.NewGenericPreConditionFailedError or models.NewGenericNotFoundError so the HTTP layer maps them to 412 or 404 correctly. (`return models.NewGenericPreConditionFailedError(fmt.Errorf("customer has active subscriptions"))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `validators/customer/validator.go` | Blocks customer delete/subject-key update when active subscriptions exist. Uses clock.Now() for ActiveAt filter. | Must filter by ActiveAt — without it, cancelled subscriptions are returned and block valid deletes. |
| `validators/subscription/validator.go` | Enforces unique-subscription invariant before Create/Update. Feature-flag-gated between per-subscription and per-feature uniqueness. | Must exclude the current subscription ID when checking uniqueness on Update; must use pagination.CollectAll to avoid silently missing subscriptions beyond first page. |
| `validators/subscription/components.go` | Pure utility functions for subscription period and feature overlap checks. | Business logic decisions belong in validator.go, not components.go. |

## Anti-Patterns

- Calling subscription.Service write methods from within a SubscriptionCommandHook — creates re-entrant hook calls.
- Using context.Background() instead of the caller-supplied ctx.
- Returning plain fmt.Errorf instead of models.NewGenericPreConditionFailedError — breaks HTTP status code mapping.
- Querying subscriptions without the ActiveAt / ActiveInPeriod filter — returns all-time subscriptions and breaks uniqueness check.

## Decisions

- **Validators placed in a dedicated validators/ sub-folder** — Avoids circular imports: subscription depends on customer, so subscription validators of customer cannot live in the subscription package root.
- **Feature flag selects uniqueness semantics** — MultiSubscriptionEnabledFF allows per-feature rather than per-subscription uniqueness checking for tenants that need multiple concurrent subscriptions.

<!-- archie:ai-end -->
