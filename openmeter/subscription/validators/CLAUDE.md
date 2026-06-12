# validators

<!-- archie:ai-start -->

> Structural folder for subscription-invariant validators injected into other services. customer/ gates customer mutations (subject-key change, delete) on active subscriptions; subscription/ enforces the per-feature vs per-subscription uniqueness/overlap rule as a command hook.

## Patterns

**Interface via Noop embedding** — customer validators embed customer.NoopRequestValidator; subscription validators embed subscription.NoOpSubscriptionCommandHook — override only the relevant methods so the contract stays satisfied. (`type validator struct { customer.NoopRequestValidator; ... }`)
**clock.Now() for active-subscription checks** — Active-subscription presence is checked via List with ActiveAt=clock.Now() (never time.Now()) so frozen-time tests stay deterministic. (`svc.List(ctx, customer.ListSubscriptionsInput{ActiveAt: clock.Now()})`)
**Feature-flag-driven uniqueness scope** — validateUniqueConstraint routes to ByFeatures or BySubscriptions based on the MultiSubscriptionEnabled feature flag; callers must never call the branch functions directly. (`if ff.MultiSubscriptionEnabled { ...BySubscriptions } else { ...ByFeatures }`)
**Typed precondition/not-found errors** — State violations return models.NewGeneric*Error (precondition/not-found/conflict) so the HTTP layer maps them to status codes. (`return models.NewGenericPreConditionFailedError(...)`)

## Anti-Patterns

- Adding validation methods without embedding the relevant Noop base, breaking the validator/hook interface assertion.
- Using time.Now() instead of clock.Now() for ActiveAt checks.
- Loading candidate subscriptions with a fixed page instead of pagination.CollectAll.
- Forgetting to exclude the current subscription on update/continue, falsely flagging a self-overlap.
- Returning plain errors for deleted/not-found/conflict states instead of models.NewGeneric*Error.

## Decisions

- **Subject-key mutation and customer delete are gated on active subscriptions.** — Keeps usage attribution stable for billing while a subscription is live.
- **Uniqueness/overlap is enforced as a SubscriptionCommandHook with separate Before and After passes.** — Lets the invariant guard every mutation path uniformly instead of being duplicated inline in the service.

<!-- archie:ai-end -->
