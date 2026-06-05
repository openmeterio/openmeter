# customer

<!-- archie:ai-start -->

> Enforces subscription-aware invariants on customer mutations: a customer's subject keys cannot change while they have an active subscription, and a customer cannot be deleted while active subscriptions exist. Implemented as a customer.RequestValidator that is injected into the customer service.

## Patterns

**Implements customer.RequestValidator via NoopRequestValidator embedding** — Validator embeds customer.NoopRequestValidator and overrides only the hooks it cares about; the compile-time assertion `var _ customer.RequestValidator = (*Validator)(nil)` must hold. (`type Validator struct { customer.NoopRequestValidator; subscriptionService subscription.Service; customerService customer.Service }`)
**Constructor validates required dependencies** — NewValidator returns an error (not panic) when subscriptionService or customerService is nil, returning *Validator and error. (`if subscriptionService == nil { return nil, fmt.Errorf("subscription service is required") }`)
**Active-subscription check via List + ActiveAt=clock.Now()** — Existence of an active subscription is determined by subscriptionService.List with CustomerID filter and ActiveAt set to clock.Now(); presence is `len(subscriptions.Items) > 0`. (`v.subscriptionService.List(ctx, subscription.ListSubscriptionsInput{Namespaces: []string{ns}, CustomerID: &filter.FilterULID{FilterString: filter.FilterString{Eq: &id}}, ActiveAt: lo.ToPtr(clock.Now())})`)
**Delegate to input.Validate() first** — Each Validate*Customer method calls input.Validate() before performing cross-aggregate checks. (`if err := input.Validate(); err != nil { return err }`)
**Typed precondition/not-found errors for customer state** — Deleted/missing customer states return models.NewGenericPreConditionFailedError / models.NewGenericNotFoundError rather than bare errors. (`return models.NewGenericPreConditionFailedError(fmt.Errorf("customer is deleted [...]"))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `validator.go` | Single-file package `customer` holding the Validator type, NewValidator constructor, ValidateUpdateCustomer and ValidateDeleteCustomer. | ValidateUpdateCustomer only blocks subject-key changes when input.CustomerMutate.UsageAttribution.SubjectKeys is non-nil AND a subscription exists; it compares the old vs new SubjectKeys index-by-index (length then per-element). Do not relax this to a set comparison — order is significant here. |

## Anti-Patterns

- Adding new validation methods without embedding NoopRequestValidator (breaks the customer.RequestValidator interface assertion).
- Using time.Now() instead of clock.Now() for ActiveAt — breaks deterministic tests that freeze time.
- Returning plain errors for deleted/not-found customers instead of the models.NewGeneric*Error constructors the HTTP layer maps to status codes.
- Performing the customer GetCustomer lookup unconditionally rather than only when SubjectKeys are being mutated.

## Decisions

- **Subject-key mutation is gated on active subscriptions to keep usage attribution stable for billing.** — Changing which subjects feed a customer mid-subscription would silently change metered usage attribution for an active billing relationship.

<!-- archie:ai-end -->
