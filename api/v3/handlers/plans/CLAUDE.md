# plans

<!-- archie:ai-start -->

> v3 HTTP handler package for all plan CRUD and lifecycle operations (create, get, list, update, delete, publish, archive). Bridges generated api/v3 types to plan.Service and productcatalog domain types. Owns a rich convert.go with hand-written bidirectional transformations for the multi-level Plan → Phase → RateCard → Price hierarchy. The planaddons/ child handles the plan-addons sub-resource.

## Patterns

**Type-alias triplets per operation** — Each operation file declares <Op>Request, <Op>Response, <Op>Handler. Request/Response alias domain input/output types directly. (`type CreatePlanRequest = plan.CreatePlanInput; type CreatePlanResponse = api.BillingPlan; type CreatePlanHandler httptransport.Handler[CreatePlanRequest, CreatePlanResponse]`)
**httptransport.NewHandlerWithArgs for path-param endpoints; NewHandler for no-param** — Endpoints with a planID path param use NewHandlerWithArgs. CreatePlan (no path param) uses NewHandler. (`httptransport.NewHandlerWithArgs(decoderFunc, operationFunc, commonhttp.JSONResponseEncoderWithStatus[Response](http.StatusOK), opts...)`)
**Namespace resolved in decoder, never operation** — h.resolveNamespace(ctx) is always the first call in the decoder closure. (`ns, err := h.resolveNamespace(ctx); if err != nil { return Request{}, err }`)
**apierrors.GenericErrorEncoder on every handler** — Every handler options block must include httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()) via httptransport.AppendOptions(h.options, ...). (`httptransport.AppendOptions(h.options, httptransport.WithOperationName("create-plan"), httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()))...`)
**IgnoreNonCriticalIssues=true on create and update** — After FromAPICreatePlanRequest / FromAPIUpsertPlanRequest, set req.IgnoreNonCriticalIssues = true before returning from the decoder. This allows plans with non-critical validation issues to be saved. (`req, err := FromAPICreatePlanRequest(ns, body); ...; req.IgnoreNonCriticalIssues = true; return req, nil`)
**Nil-check on service response for mutating operations** — CreatePlan, UpdatePlan, PublishPlan, ArchivePlan check for p == nil after the service call and return a descriptive error. Get/Delete do not need this check. (`if p == nil { return CreatePlanResponse{}, fmt.Errorf("failed to create plan") }`)
**Price type exhaustive switch with unsupported-type guard** — ToAPIBillingPrice uses an exhaustive switch on p.Type(). DynamicPriceType and PackagePriceType return models.NewGenericConflictError. ListPlans filters out plans containing unsupported prices via hasUnsupportedV3Price. (`case productcatalog.DynamicPriceType: return result, models.NewGenericConflictError(fmt.Errorf("dynamic price is not supported in v3 API"))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Defines Handler interface (7 method signatures) and handler struct with resolveNamespace, service (plan.Service), and options. New() constructor. | plan.Service is the only domain service injected here — unlike meters, there is no streaming or customer service dependency at this level. |
| `convert.go` | All bidirectional conversion: ToAPIBillingPlan, ToAPIBillingPlanPhase, ToAPIBillingRateCard, ToAPIBillingPrice, ToAPIBillingPriceTiers, ToAPIBillingRateCardTaxConfig, ToAPIBillingRateCardDiscount, ToAPIBillingSpendCommitments, ToAPIProductCatalogValidationErrors, FromAPICreatePlanRequest, FromAPIUpsertPlanRequest, FromAPIBillingPlanPhase, FromAPIBillingRateCard, FromAPIBillingPrice, ToProRatingConfig. | DynamicPriceType and PackagePriceType are intentionally unsupported in v3 — return GenericConflictError, not a 500. Adding a new price type requires updating both ToAPIBillingPrice and FromAPIBillingPrice and the unsupportedV3PriceTypes map. |
| `convert_test.go` | Unit tests for all convert.go functions using table-driven subtests. Uses clock.SetTime for status tests, newTestPlan helper, and decimal/gobl/currency literals. | New conversion paths must have corresponding table-driven tests here. Use assert.Equal with typed API constants (api.BillingPlanStatus*, api.ISO8601Duration, api.Numeric) rather than plain strings. |
| `list.go` | ListPlans with page-based pagination (default page 1, size 20). Calls hasUnsupportedV3Price to silently skip plans with dynamic/package prices. | The hasUnsupportedV3Price skip is temporary (FIXME comment). Do not add other silent-skip logic; return errors or expose the items. |
| `publish.go` | PublishPlan sets EffectiveFrom = clock.Now() in the decoder, not in the service layer. | Using clock.Now() (not time.Now()) is required so tests can control the clock via clock.SetTime. |
| `archive.go` | ArchivePlan sets EffectiveTo = clock.Now() in the decoder. | Same clock.Now() requirement as publish.go. |

## Anti-Patterns

- Using time.Now() instead of clock.Now() in decoder closures — breaks time-controlled tests.
- Omitting req.IgnoreNonCriticalIssues = true in Create/Update decoders — rejects plans with non-critical validation issues.
- Adding new price type support in ToAPIBillingPrice without also updating FromAPIBillingPrice, hasUnsupportedV3Price, and the round-trip tests in convert_test.go.
- Omitting httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()) from any handler's options block.
- Calling domain service methods from the encoder closure (third argument) — all service calls must happen in the operation closure (second argument).

## Decisions

- **DynamicPriceType and PackagePriceType return GenericConflictError (409) in ToAPIBillingPrice** — The v3 API does not yet have a wire format for these price types; returning a conflict error surfaces the limitation clearly to callers rather than silently dropping data.
- **Labels round-trip via labels.ToMetadata / labels.FromMetadata, not direct map assignment** — The labels package enforces key/value format rules and provides consistent nil-vs-empty handling across all handlers that touch label metadata.
- **IgnoreNonCriticalIssues set in decoder, not in domain service** — HTTP create/update callers expect lenient validation (non-critical issues allowed); the service layer retains the ability to surface those issues as warnings on the returned object without blocking persistence.

## Example: Add a new lifecycle action (e.g. ClonePlan) following the established handler pattern with clock.Now() and nil-check

```
// clone.go
package plans

import (
	"context"
	"fmt"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)
// ...
```

<!-- archie:ai-end -->
