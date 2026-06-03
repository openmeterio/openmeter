# subscriptions

<!-- archie:ai-start -->

> v3 HTTP handler package for billing subscription lifecycle (list, get, create-from-plan, cancel, unschedule cancellation, change-plan); uses two distinct domain services — planSubscriptionService for create/change workflows and subscriptionService for get/list/cancel/continue. The subscriptionaddons/ sub-package lists subscription addons.

## Patterns

**Two subscription services with split responsibilities** — The handler struct holds planSubscriptionService (create/change, drives billing sync + plan versioning) and subscriptionService (get/list/cancel/continue). Using the wrong one silently changes workflow semantics. (`h.planSubscriptionService.Create(ctx, request) // not h.subscriptionService.Create`)
**Pre-validation of referenced entities in the decoder** — Resolve and validate customer and plan refs inside the decoder via getCustomerByIDOrKey/getPlanByIDOrKey, returning apierrors.NewBadRequestError with InvalidParameters for missing required fields. (`if body.Plan.Id == nil && body.Plan.Key == nil { return ..., apierrors.NewBadRequestError(ctx, errors.New(reason), []apierrors.InvalidParameter{{Field: "plan.id", Rule: "required", ...}}) }`)
**PlanInput constructed via FromRef** — After resolving a concrete plan, build plansubscription.PlanInput via planInput.FromRef(&PlanRefInput{Key, Version}) — never direct field assignment. (`planInput := plansubscription.PlanInput{}; planInput.FromRef(&plansubscription.PlanRefInput{Key: planEntity.Key, Version: &planEntity.Version})`)
**Timing union: AsDateTime() before enum** — FromAPIBillingSubscriptionEditTiming tries AsDateTime() first, enum fallback second; order is load-bearing because RFC3339 strings also satisfy the enum string union. (`if custom, err := t.AsDateTime(); err == nil { return subscription.Timing{Custom: &custom}, nil }`)
**Cancel defaults timing to immediate** — cancel.go sets timing.Enum = TimingImmediate when body.Timing is nil rather than requiring a timing value. (`if body.Timing == nil { timing.Enum = lo.ToPtr(subscription.TimingImmediate) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Handler interface (6 methods) and struct: resolveNamespace, customerService, planService, planSubscriptionService, subscriptionService. | Mixing the two subscription services changes workflow semantics and billing sync. |
| `create.go` | Creates subscription from plan; validates customer+plan, builds PlanInput via FromRef, calls planSubscriptionService.Create. Defines private getCustomerByIDOrKey/getPlanByIDOrKey helpers. | Helpers are intentionally private to the handler (TODO to move to service layer). |
| `change.go` | Change-to-plan; fetches current subscription for name/desc/metadata defaults, builds ChangeSubscriptionWorkflowInput, calls planSubscriptionService.Change. | Response is BillingSubscriptionChangeResponse{Current, Next}; metadata defaults from existing subscription unless body.Labels is non-nil. |
| `convert.go` | ToAPIBillingSubscription (clock.Now() for GetStatusAt), FromAPIBillingSubscriptionEditTiming (datetime-before-enum), FromAPIBillingSubscriptionCreate. | Status tests must mock clock via clock.SetTime; reversing the timing decode order breaks custom RFC3339 timing. |
| `cancel.go` | CancelSubscription; defaults nil timing to immediate. | Nil timing must default to immediate, not error. |
| `subscriptionaddons/` | Sub-package listing subscription addons; toAPISubscriptionAddon unions instance periods into one ActiveFrom/ActiveTo via clock.Now(). | Sort validation uses subscriptionaddon.OrderBy.Validate() after field mapping. |

## Anti-Patterns

- Swapping planSubscriptionService and subscriptionService — Create/Change must use the plan service for billing sync hooks
- Reversing timing decode order (enum before datetime) — datetime strings get misidentified as enum
- Calling planSubscriptionService without first resolving customer+plan via getByIDOrKey helpers
- Using commonhttp.GenericErrorEncoder() instead of apierrors.GenericErrorEncoder() in v3 handlers
- Setting PlanInput fields directly instead of planInput.FromRef

## Decisions

- **Two separate subscription services** — plansubscription.PlanSubscriptionService orchestrates billing sync and plan versioning atop core subscription.Service; the split keeps billing-coupling out of generic lifecycle and allows non-plan subscriptions later.
- **datetime decoded before enum in timing decode** — Both satisfy the string union; trying datetime first is the only way to correctly route RFC3339 custom timestamps without misidentifying them as enum values.
- **Entity pre-validation happens in the decoder** — Failing fast with apierrors.NewBadRequestError and typed InvalidParameters gives callers actionable field-level errors before any service call.

<!-- archie:ai-end -->
