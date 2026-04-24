# subscriptions

<!-- archie:ai-start -->

> v3 HTTP handler package for billing subscription lifecycle: list, get, create (from plan), cancel, unschedule cancellation, and change-plan. Delegates to plansubscription.PlanSubscriptionService for create/change and subscription.Service for get/list/cancel/continue.

## Patterns

**Pre-validation of referenced entities in decoder** — Resolve and validate customer and plan references inside the request decoder (first func). Use apierrors.NewBadRequestError with InvalidParameters for missing required fields (plan.id/plan.key, customer.id/customer.key). (`if body.Plan.Id == nil && body.Plan.Key == nil { return ..., apierrors.NewBadRequestError(ctx, errors.New(reason), []apierrors.InvalidParameter{{Field: "plan.id", ...}, {Field: "plan.key", ...}}) }`)
**Helper methods for ID-or-key entity lookup** — getCustomerByIDOrKey and getPlanByIDOrKey are private handler methods. They accept namespace + optional ID + optional Key pointers and return the entity. TODO comments note these should eventually move to the service layer. (`customerEntity, err := h.getCustomerByIDOrKey(ctx, ns, body.Customer.Id, body.Customer.Key)`)
**Timing union type decoding: datetime before enum** — FromAPIBillingSubscriptionEditTiming tries AsDateTime() first (RFC3339) before AsBillingSubscriptionEditTimingEnum(). This order is critical — datetime strings also match as strings, so enum fallback must be second. (`if custom, err := t.AsDateTime(); err == nil { return subscription.Timing{Custom: &custom}, nil }`)
**PlanInput via FromRef** — Always construct plansubscription.PlanInput via planInput.FromRef(&PlanRefInput{Key: ..., Version: ...}) after resolving the concrete plan. Never set PlanInput fields directly. (`planInput.FromRef(&plansubscription.PlanRefInput{Key: planEntity.Key, Version: &planEntity.Version})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Handler interface (6 methods) and handler struct: resolveNamespace, customerService, planService, planSubscriptionService, subscriptionService. | Two distinct subscription services: planSubscriptionService (create/change) vs subscriptionService (get/list/cancel/continue). Using the wrong one silently changes workflow semantics. |
| `create.go` | Creates subscription from plan; validates customer and plan exist, builds PlanInput via FromRef, calls planSubscriptionService.Create. | getCustomerByIDOrKey and getPlanByIDOrKey are defined here as private handler methods. |
| `change.go` | Changes running subscription to a new plan; fetches current subscription for name/desc/metadata defaults, builds ChangeSubscriptionWorkflowInput, calls planSubscriptionService.Change. | Response is BillingSubscriptionChangeResponse{Current, Next} — not a single subscription. |
| `convert.go` | ToAPIBillingSubscription, FromAPIBillingSubscriptionEditTiming, FromAPIBillingSubscriptionCreate. Timing decode order (datetime before enum) is load-bearing. | clock.Now() is used for GetStatusAt — tests must mock clock if status assertions are needed. |

## Anti-Patterns

- Swapping planSubscriptionService and subscriptionService calls — Create/Change must go through plan service for billing sync hooks
- Checking plan.Key/Version after planSubscriptionService.Create instead of before — plan must be validated and resolved pre-call
- Reversing timing decode order (enum before datetime) — datetime strings will be misidentified as enum values

## Decisions

- **Two separate subscription services (plan-aware and base)** — plansubscription.PlanSubscriptionService orchestrates billing sync and plan versioning on top of the core subscription.Service; the split keeps billing-coupling out of the generic subscription lifecycle.

<!-- archie:ai-end -->
