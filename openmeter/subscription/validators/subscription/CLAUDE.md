# subscription

<!-- archie:ai-start -->

> Implements the subscription uniqueness/overlap invariant as a subscription.SubscriptionCommandHook: before and after every create/update/cancel/continue/delete it recomputes the customer's overlapping subscription set and rejects mutations that would violate uniqueness (either per-feature or per-subscription depending on the multi-subscription feature flag).

## Patterns

**SubscriptionCommandHook via NoOpSubscriptionCommandHook embedding** — SubscriptionUniqueConstraintValidator embeds subscription.NoOpSubscriptionCommandHook and overrides Before*/After* hooks; NewSubscriptionUniqueConstraintValidator returns subscription.SubscriptionCommandHook. (`type SubscriptionUniqueConstraintValidator struct { subscription.NoOpSubscriptionCommandHook; Config SubscriptionUniqueConstraintValidatorConfig }`)
**Config struct with Validate() and constructor guard** — Dependencies (FeatureFlags ffx.Service, QueryService subscription.QueryService, CustomerService customer.Service) live in a Config struct with a Validate() method; the constructor returns an error if Config.Validate() fails. (`if err := config.Validate(); err != nil { return nil, fmt.Errorf("invalid subscription unique constraint validator config: %w", err) }`)
**Feature-flag-driven uniqueness mode** — validateUniqueConstraint checks subscription.MultiSubscriptionEnabledFF; if enabled it calls subscription.ValidateUniqueConstraintByFeatures, otherwise ValidateUniqueConstraintBySubscriptions. BeforeUpdate is skipped entirely when multi-subscription is disabled. (`if multiSubscriptionEnabled { return specs, subscription.ValidateUniqueConstraintByFeatures(specs) } return specs, subscription.ValidateUniqueConstraintBySubscriptions(specs)`)
**Sub -> View -> Spec pipeline of small step functions** — components.go provides composable steps (collectCustomerSubscriptionsStarting, mapSubsToViews via QueryService.ExpandViews, mapViewsToSpecs via AsSpec, filterSubViews, includeSubSpec, includeSubViewUnique) that the Before/After hooks chain together. (`views, _ := v.mapSubsToViews(ctx, subs); specs, _ := v.mapViewsToSpecs(views); specs, _ = v.validateUniqueConstraint(ctx, specs)`)
**Two-phase Before validation (existing-set then candidate-included)** — Before* hooks first validate the already-scheduled set alone (wrapping failures as an inconsistency error), then include the candidate spec and validate again; the candidate is excluded from the loaded set first (filterSubViews on Subscription.ID) when updating/continuing. (`specs, err = v.validateUniqueConstraint(ctx, specs); if err != nil { return fmt.Errorf("inconsistency error: already scheduled subscriptions are overlapping: %w", err) }`)
**Paginated full collection of candidate subscriptions** — collectCustomerSubscriptionsStarting uses pagination.CollectAll over QueryService.List with ActiveInPeriod=StartBoundedPeriod{From: starting}, page size 1000, filtered by customer + namespace. (`pagination.CollectAll(ctx, pagination.NewPaginator(func(...) { return v.Config.QueryService.List(ctx, subscription.ListSubscriptionsInput{ActiveInPeriod: &timeutil.StartBoundedPeriod{From: starting}, ...}) }), 1000)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `validator.go` | Holds the Config struct + Validate(), the validator type, NewSubscriptionUniqueConstraintValidator, all hook methods (BeforeCreate/Update/Continue, AfterCreate/Update/Cancel/Continue, BeforeDelete) and collectCustomerSubscriptionsStarting. | BeforeUpdate is a no-op unless MultiSubscriptionEnabledFF is on. All After* and BeforeDelete delegate to pipelineAfter, which re-validates the post-mutation set (using includeSubViewUnique to dedupe by Subscription.ID). BeforeContinue clears spec.ActiveTo (continue = indefinite) before validating. |
| `components.go` | Method-receiver helper steps for the validation pipeline (validateUniqueConstraint, mapSubsToViews, mapViewsToSpecs, includeSubSpec, includeSubViewUnique, filterSubViews). | mapSubsToViews delegates to Config.QueryService.ExpandViews — an extra DB round trip per hook invocation. includeSubViewUnique dedupes via lo.UniqBy on Subscription.ID; do not assume input views are already unique. |

## Anti-Patterns

- Calling ValidateUniqueConstraintByFeatures/BySubscriptions directly instead of routing through validateUniqueConstraint (bypasses the MultiSubscriptionEnabledFF branch).
- Forgetting to exclude the current subscription (filterSubViews on Subscription.ID) in update/continue paths, which would falsely flag a self-overlap.
- Adding a hook that loads candidate subscriptions with a fixed page instead of pagination.CollectAll — misses customers with >1000 subscriptions.
- Returning the inconsistency-wrapped error for the candidate-included validation; only the pre-existing-set validation should be wrapped as 'already scheduled subscriptions are overlapping'.
- Implementing hooks without embedding NoOpSubscriptionCommandHook, breaking the subscription.SubscriptionCommandHook contract.

## Decisions

- **Uniqueness is enforced as a command hook rather than inline in the service, with separate Before and After passes.** — Before passes reject invalid mutations pre-commit; After passes re-validate the materialized state to catch inconsistencies the spec-level check could miss, keeping the invariant centralized and reusable across create/update/cancel/continue/delete.
- **Uniqueness scope (per-feature vs per-subscription) is selected by feature flag.** — Single-subscription deployments enforce one active subscription per customer; multi-subscription deployments instead enforce non-overlap at feature granularity.

## Example: A Before* hook chaining the pipeline: load overlapping subs, validate existing set, then validate with candidate included.

```
func (v SubscriptionUniqueConstraintValidator) BeforeCreate(ctx context.Context, namespace string, spec subscription.SubscriptionSpec) error {
	subs, err := v.collectCustomerSubscriptionsStarting(ctx, namespace, spec.CustomerId, spec.ActiveFrom)
	if err != nil { return err }
	views, err := v.mapSubsToViews(ctx, subs)
	if err != nil { return err }
	specs, err := v.mapViewsToSpecs(views)
	if err != nil { return err }
	specs, err = v.validateUniqueConstraint(ctx, specs)
	if err != nil { return fmt.Errorf("inconsistency error: already scheduled subscriptions are overlapping: %w", err) }
	specs, err = v.includeSubSpec(spec, specs)
	if err != nil { return err }
	_, err = v.validateUniqueConstraint(ctx, specs)
	return err
}
```

<!-- archie:ai-end -->
