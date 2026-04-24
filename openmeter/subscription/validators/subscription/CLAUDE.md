# subscription

<!-- archie:ai-start -->

> Implements subscription.SubscriptionCommandHook to enforce the unique-constraint invariant: no two active subscriptions for the same customer may overlap in time or cover the same features (depending on feature-flag state).

## Patterns

**NoOpSubscriptionCommandHook embedding** — Embed subscription.NoOpSubscriptionCommandHook so only the overridden Before*/After* methods need implementation. (`type SubscriptionUniqueConstraintValidator struct { subscription.NoOpSubscriptionCommandHook; Config SubscriptionUniqueConstraintValidatorConfig }`)
**Config struct with Validate()** — Group all dependencies in a *ValidatorConfig struct with its own Validate() method; the constructor calls config.Validate() and returns an error before construction if any dependency is missing. (`func (c SubscriptionUniqueConstraintValidatorConfig) Validate() error { if c.FeatureFlags == nil { return fmt.Errorf("feature flags is required") } ... }`)
**Pipeline composition through small component methods** — Split validation into named steps (collectCustomerSubscriptionsStarting, mapSubsToViews, mapViewsToSpecs, validateUniqueConstraint, includeSubSpec) in components.go and compose them in validator.go — each step returns (result, error). (`subs, _ := v.collectCustomerSubscriptionsStarting(...); views, _ := v.mapSubsToViews(...); specs, _ := v.mapViewsToSpecs(views); _, err = v.validateUniqueConstraint(ctx, specs)`)
**Feature-flag-gated logic** — Check ffx.Service.IsFeatureEnabled for subscription.MultiSubscriptionEnabledFF before choosing which validation path to run; BeforeUpdate skips validation entirely when multi-subscription is disabled. (`multiSubscriptionEnabled, err := v.Config.FeatureFlags.IsFeatureEnabled(ctx, subscription.MultiSubscriptionEnabledFF)`)
**pagination.CollectAll for unbounded list queries** — Use pagination.CollectAll + pagination.NewPaginator when fetching all customer subscriptions to avoid partial result bugs. (`return pagination.CollectAll(ctx, pagination.NewPaginator(func(ctx context.Context, page pagination.Page) (pagination.Result[subscription.Subscription], error) { return v.Config.QueryService.List(ctx, ...) }), 1000)`)
**Exclude current subscription by ID before re-inserting updated spec** — In BeforeUpdate, filter out the subscription being updated (by ID) from the existing set before appending the target spec, so the old version does not conflict with the new one. (`views, err = v.filterSubViews(func(v subscription.SubscriptionView) bool { return v.Subscription.ID != currentId.ID }, views)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `validator.go` | Implements all Before*/After* hook methods; each method builds a pipeline that collects existing subscriptions, maps them to specs, validates uniqueness, then re-validates with the incoming spec included. | pipelineAfter is reused for AfterCreate, AfterUpdate, AfterCancel, AfterContinue, and BeforeDelete — any change to post-mutation validation affects all five hooks. |
| `components.go` | Low-level pipeline building blocks (collect, map, filter, include) used by validator.go; no business rules live here. | includeSubViewUnique uses lo.UniqBy by subscription ID to prevent double-counting the same subscription; losing that dedup causes false positive uniqueness errors. |

## Anti-Patterns

- Querying subscriptions without the ActiveInPeriod filter — returns all-time subscriptions and breaks the uniqueness check
- Calling subscription.Service (write methods) from within a hook — creates re-entrant hook calls
- Using context.Background() instead of the caller-supplied ctx
- Bypassing pagination.CollectAll for large customer subscription lists — silently misses subscriptions beyond first page
- Adding business logic directly into components.go — it is a pure utility file; keep domain decisions in validator.go

## Decisions

- **Components are separated into components.go to keep validator.go focused on the validation pipeline logic.** — Improves readability and testability of each pipeline step in isolation.
- **Feature flag subscription.MultiSubscriptionEnabledFF selects between per-subscription and per-feature uniqueness checks.** — Allows gradual rollout of multi-subscription support without breaking existing single-subscription customers.
- **Hook is registered via subscription.Service.RegisterHook() at wiring time (app/common), not in the subscription package itself.** — Keeps the constraint out of the core subscription package to avoid tight coupling and import cycles.

## Example: Register the validator during app wiring

```
import subval "github.com/openmeterio/openmeter/openmeter/subscription/validators/subscription"

hook, err := subval.NewSubscriptionUniqueConstraintValidator(subval.SubscriptionUniqueConstraintValidatorConfig{
    FeatureFlags:    featureFlagService,
    QueryService:    subscriptionService,
    CustomerService: customerService,
})
if err != nil { return err }
subscriptionService.RegisterHook(hook)
```

<!-- archie:ai-end -->
