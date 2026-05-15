# plans

<!-- archie:ai-start -->

> v3 HTTP handler package for all plan CRUD and lifecycle operations (create, get, list, update, delete, publish, archive); bridges generated api/v3 types to plan.Service with rich bidirectional conversion for the Plan > Phase > RateCard > Price hierarchy.

## Patterns

**Type-alias triplets per operation file** — Each operation file declares <Op>Request, <Op>Response, <Op>Handler type aliases. Request/Response alias domain input/output types directly. (`type CreatePlanRequest = plan.CreatePlanInput; type CreatePlanResponse = api.BillingPlan; type CreatePlanHandler httptransport.Handler[CreatePlanRequest, CreatePlanResponse]`)
**NewHandlerWithArgs for path-param endpoints; NewHandler for no-param** — Endpoints with a planID path param use NewHandlerWithArgs; CreatePlan (no path param) uses NewHandler. (`httptransport.NewHandlerWithArgs(decoderFunc, operationFunc, commonhttp.JSONResponseEncoderWithStatus[Response](http.StatusOK), opts...)`)
**Namespace resolved first in decoder** — h.resolveNamespace(ctx) is always the first call in the decoder closure; return immediately on error. (`ns, err := h.resolveNamespace(ctx); if err != nil { return Request{}, err }`)
**IgnoreNonCriticalIssues=true on create and update** — After FromAPICreatePlanRequest / FromAPIUpsertPlanRequest, set req.IgnoreNonCriticalIssues = true before returning from the decoder. (`req, err := FromAPICreatePlanRequest(ns, body); ...; req.IgnoreNonCriticalIssues = true; return req, nil`)
**clock.Now() for time-sensitive fields in decoder** — PublishPlan sets EffectiveFrom = clock.Now(); ArchivePlan sets EffectiveTo = clock.Now(). Never use time.Now() — tests mock clock via clock.SetTime. (`EffectiveFrom: lo.ToPtr(clock.Now())`)
**Nil-check on service response for mutating lifecycle operations** — CreatePlan, UpdatePlan, PublishPlan, ArchivePlan check for p == nil after the service call and return a descriptive error. Get/Delete do not need this check. (`if p == nil { return CreatePlanResponse{}, fmt.Errorf("failed to create plan") }`)
**Exhaustive price type switch with unsupported-type guard** — ToAPIBillingPrice uses an exhaustive switch on p.Type(). DynamicPriceType and PackagePriceType return models.NewGenericConflictError. ListPlans silently skips plans with unsupported prices via hasUnsupportedV3Price. (`case productcatalog.DynamicPriceType: return result, models.NewGenericConflictError(fmt.Errorf("dynamic price is not supported in v3 API"))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Defines Handler interface (7 method signatures) and handler struct with resolveNamespace, service (plan.Service), and options. New() constructor. Only plan.Service is injected — no streaming or customer service dependency. | Do not inject additional services here; if you need customer lookups, delegate to plan.Service or add a dedicated helper. |
| `convert.go` | All bidirectional conversion: ToAPIBillingPlan, ToAPIBillingPlanPhase, ToAPIBillingRateCard, ToAPIBillingPrice, ToAPIBillingPriceTiers, FromAPICreatePlanRequest, FromAPIUpsertPlanRequest, FromAPIBillingPlanPhase, FromAPIBillingRateCard, FromAPIBillingPrice, ToProRatingConfig. | DynamicPriceType and PackagePriceType return GenericConflictError (not 500). Adding a new price type requires updating ToAPIBillingPrice, FromAPIBillingPrice, and the unsupportedV3PriceTypes map. |
| `convert_test.go` | Table-driven unit tests for all convert.go functions. Uses clock.SetTime for status tests, newTestPlan helper, and decimal/gobl/currency literals. | New conversion paths must have corresponding table-driven tests here; use assert.Equal with typed API constants (api.BillingPlanStatus*, api.ISO8601Duration, api.Numeric) rather than plain strings. |
| `list.go` | ListPlans with page-based pagination (default page 1, size 20). Silently skips plans with unsupported v3 prices via hasUnsupportedV3Price. | The hasUnsupportedV3Price skip is a temporary workaround (FIXME). Do not add further silent-skip logic — return errors or expose all items. |
| `publish.go` | PublishPlan sets EffectiveFrom = clock.Now() in the decoder. | Must use clock.Now() not time.Now() — tests control clock via clock.SetTime. |
| `archive.go` | ArchivePlan sets EffectiveTo = clock.Now() in the decoder. | Same clock.Now() requirement as publish.go. |

## Anti-Patterns

- Using time.Now() instead of clock.Now() in decoder closures — breaks time-controlled tests.
- Omitting req.IgnoreNonCriticalIssues = true in Create/Update decoders — rejects plans with non-critical validation issues.
- Adding a new price type in ToAPIBillingPrice without also updating FromAPIBillingPrice, hasUnsupportedV3Price, and the round-trip tests in convert_test.go.
- Omitting httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()) from any handler options block.
- Calling domain service methods from the encoder closure (third argument) — all service calls must happen in the operation closure (second argument).

## Decisions

- **DynamicPriceType and PackagePriceType return GenericConflictError (409) rather than a 500** — The v3 API does not yet have a wire format for these price types; a conflict error surfaces the limitation clearly to callers instead of silently dropping data or returning a generic server error.
- **Labels round-trip via labels.ToMetadata / labels.FromMetadata, not direct map assignment** — The labels package enforces key/value format rules and provides consistent nil-vs-empty handling across all handlers that touch label metadata.
- **IgnoreNonCriticalIssues set in the HTTP decoder, not in the domain service** — HTTP create/update callers expect lenient validation; the service layer retains the ability to surface non-critical issues as warnings on the returned object without blocking persistence.

## Example: Add a new plan lifecycle action (e.g. ClonePlan) with clock.Now() and nil-check

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
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
// ...
```

<!-- archie:ai-end -->
