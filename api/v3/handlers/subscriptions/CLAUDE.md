# subscriptions

<!-- archie:ai-start -->

> v3 HTTP handler package for billing subscription lifecycle (list, get, create from plan, cancel, unschedule cancellation, change-plan); uses two distinct domain services — planSubscriptionService for create/change workflows and subscriptionService for get/list/cancel/continue.

## Patterns

**Pre-validation of referenced entities inside the decoder** — Resolve and validate customer and plan references inside the request decoder (first func) using apierrors.NewBadRequestError with InvalidParameters for missing required fields. (`if body.Plan.Id == nil && body.Plan.Key == nil { return ..., apierrors.NewBadRequestError(ctx, errors.New(reason), []apierrors.InvalidParameter{{Field: "plan.id", ...}}) }`)
**Private helper methods for ID-or-key entity lookup** — getCustomerByIDOrKey and getPlanByIDOrKey are private handler methods that accept namespace + optional ID/Key pointers and return the entity. They are defined in create.go and reused by change.go. (`customerEntity, err := h.getCustomerByIDOrKey(ctx, ns, body.Customer.Id, body.Customer.Key)`)
**PlanInput constructed via FromRef, never direct field assignment** — Always construct plansubscription.PlanInput via planInput.FromRef(&PlanRefInput{Key: ..., Version: ...}) after resolving the concrete plan. (`planInput := plansubscription.PlanInput{}; planInput.FromRef(&plansubscription.PlanRefInput{Key: planEntity.Key, Version: &planEntity.Version})`)
**Timing union decode: AsDateTime() before AsBillingSubscriptionEditTimingEnum()** — FromAPIBillingSubscriptionEditTiming tries AsDateTime() first; enum fallback is second. This order is critical because datetime strings also satisfy the string union type. (`if custom, err := t.AsDateTime(); err == nil { return subscription.Timing{Custom: &custom}, nil }`)
**planSubscriptionService for create/change; subscriptionService for get/list/cancel/continue** — The handler struct holds two subscription services with different responsibilities. Using the wrong one silently changes workflow semantics and billing sync. (`h.planSubscriptionService.Create(ctx, request) // not h.subscriptionService.Create`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Handler interface (6 methods) and handler struct: resolveNamespace, customerService, planService, planSubscriptionService, subscriptionService. | Two distinct subscription services are injected: planSubscriptionService (create/change) and subscriptionService (get/list/cancel/continue). Mixing them changes workflow semantics. |
| `create.go` | Creates subscription from plan; validates customer and plan exist, builds PlanInput via FromRef, calls planSubscriptionService.Create. Also defines getCustomerByIDOrKey and getPlanByIDOrKey private helpers. | TODO comments note helpers should eventually move to the service layer — keep them private to handler for now. |
| `change.go` | Changes running subscription to a new plan; fetches current subscription for name/desc/metadata defaults, builds ChangeSubscriptionWorkflowInput, calls planSubscriptionService.Change. Response is BillingSubscriptionChangeResponse{Current, Next} not a single subscription. | Metadata defaults come from the existing subscription; only override if body.Labels is non-nil. |
| `convert.go` | ToAPIBillingSubscription (uses clock.Now() for GetStatusAt), FromAPIBillingSubscriptionEditTiming (datetime-before-enum order), FromAPIBillingSubscriptionCreate. | Tests that assert subscription status must mock clock via clock.SetTime. The timing decode order (datetime first) is load-bearing — reversing it breaks custom RFC3339 timing. |
| `cancel.go` | CancelSubscription; defaults timing to TimingImmediate when body.Timing is nil. | Nil timing body must default to immediate — do not require timing to be present. |

## Anti-Patterns

- Swapping planSubscriptionService and subscriptionService calls — Create/Change must go through the plan service for billing sync hooks.
- Reversing timing decode order (trying enum before datetime) — datetime strings will be misidentified as enum values.
- Skipping getCustomerByIDOrKey / getPlanByIDOrKey pre-validation and calling planSubscriptionService directly — plan must be resolved to a concrete version before creating.
- Using commonhttp.GenericErrorEncoder() instead of apierrors.GenericErrorEncoder() in v3 handlers.
- Setting PlanInput fields directly instead of using planInput.FromRef — direct assignment bypasses the FromRef invariants.

## Decisions

- **Two separate subscription services: planSubscriptionService and subscriptionService** — plansubscription.PlanSubscriptionService orchestrates billing sync and plan versioning on top of the core subscription.Service; the split keeps billing-coupling out of the generic subscription lifecycle and allows non-plan subscriptions in the future.
- **datetime decoded before enum in FromAPIBillingSubscriptionEditTiming** — Both datetime strings and enum strings satisfy the string union type; trying datetime first and falling back to enum is the only way to correctly route RFC3339 custom timestamps without misidentifying them as enum values.
- **Entity pre-validation (customer + plan lookup) happens in the decoder, not in the operation closure** — Failing fast in the decoder returns structured 400 errors via apierrors.NewBadRequestError with typed InvalidParameters, giving callers actionable field-level error messages before any service call is made.

## Example: Add a new subscription lifecycle action (e.g. PauseSubscription) following the established handler pattern

```
// pause.go
package subscriptions

import (
	"context"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	models "github.com/openmeterio/openmeter/pkg/models"
)
// ...
```

<!-- archie:ai-end -->
