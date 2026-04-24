# features

<!-- archie:ai-start -->

> Full CRUD v3 HTTP handler package for product-catalog features (list, get, create, update, delete/archive). Includes LLM pricing enrichment on get/update responses and domain-error-specific HTTP status mapping via a local errorEncoder.

## Patterns

**Per-operation file split** — Each CRUD verb lives in its own file (create.go, get.go, list.go, update.go, delete.go). handler.go declares the Handler interface and New() constructor. convert.go holds all domain<->API type conversions. error_encoder.go holds domain-error HTTP status mapping. (`func (h *handler) CreateFeature() CreateFeatureHandler { return httptransport.NewHandler(...) }`)
**Goverter-free manual conversion with unit tests** — This package uses hand-written convert.go functions (not Goverter) and covers them with convert_test.go. All convert functions return (T, error) to propagate parse failures. (`func convertFeatureToAPI(f feature.Feature) (api.Feature, error)`)
**Domain-specific errorEncoder chained after GenericErrorEncoder** — Append httptransport.WithErrorEncoder(errorEncoder()) after apierrors.GenericErrorEncoder() in AppendOptions. errorEncoder handles feature-domain typed errors (FeatureInvalidFiltersError → 400, ForbiddenError → 403, FeatureWithNameAlreadyExistsError → 409). (`httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()), httptransport.WithErrorEncoder(errorEncoder())`)
**Conditional LLM pricing enrichment on read responses** — After converting a feature to API, check UnitCost.Type == UnitCostTypeLLM and call resolveLLMPricing; if pricing returned is non-nil call enrichFeatureResponseWithPricing. This is done in both get.go and update.go. (`if feat.UnitCost != nil && feat.UnitCost.Type == feature.UnitCostTypeLLM && h.llmcostService != nil { pricing := resolveLLMPricing(ctx, h.llmcostService, feat); if pricing != nil { enrichFeatureResponseWithPricing(&resp, pricing) } }`)
**Meter validation at decode time** — Meter ID resolution and filter key validation happen in the request decoder (first func of NewHandler), not in the operation func. Use models.NewGenericValidationError for filter key mismatches. (`func validateMeterFilters(filters map[string]api.QueryFilterStringMapItem, m meter.Meter) error { for k := range filters { if _, ok := m.GroupBy[k]; !ok { return models.NewGenericValidationError(...) } } }`)
**nullable.Nullable for optional-null PATCH fields** — Update inputs use nullable.Nullable[T]; check IsNull() to clear, IsSpecified()+Get() to set, neither to leave unchanged. Test all three branches. (`if body.UnitCost.IsNull() { input.UnitCost = nullable.NewNullNullable[feature.UnitCost]() } else if body.UnitCost.IsSpecified() { ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Handler interface with 5 methods; handler struct holds resolveNamespace, connector (feature.FeatureConnector), meterService, llmcostService, options. | llmcostService may be nil (optional dependency); guard with != nil before calling. |
| `convert.go` | All domain<->API conversions: convertFeatureToAPI, convertCreateRequestToDomain, convertUpdateRequestToDomain, convertUnitCostToAPI/FromAPI, enrichFeatureResponseWithPricing, resolveLLMPricing, filter conversions. | UnitCost discriminator switch must cover all UnitCostType values; unknown type returns error. |
| `error_encoder.go` | Maps feature-domain typed errors to HTTP status codes using commonhttp.HandleErrorIfTypeMatches. | Add new feature-domain errors here; do not let them fall through to 500. |
| `list.go` | List with pagination (default page 1/size 20), filter[meter_id], filter[key], filter[name], sort. Uses apierrors.NewBadRequestError for invalid filter/sort params. | Page validation must be called after construction; missing it allows page=0 through. |
| `create.go` | Validates meter reference exists and filter keys match meter GroupBy dimensions before calling CreateFeature. | validateMeterFilters only checks key existence, not operator validity; operator validation is done downstream in MeterGroupByFilters.Validate. |

## Anti-Patterns

- Calling h.connector directly from handler.go methods instead of delegating to per-operation files
- Skipping errorEncoder() in AppendOptions — feature domain errors will become 500
- Returning pricing enrichment errors instead of silently skipping — pricing is best-effort
- Accepting llmcostService as non-optional if it can be nil in some wirings

## Decisions

- **Hand-written conversions instead of Goverter** — UnitCost is a discriminated union (manual/llm) requiring switch-based dispatch that Goverter cannot generate cleanly; hand-written code with tests is more explicit.
- **LLM pricing resolved at read time, not stored** — Pricing data changes independently of feature configuration; resolving at read time from llmcost.Service ensures freshness without coupling the feature write path to pricing.

## Example: Add a new CRUD operation (e.g. PatchFeature)

```
// patch.go
package features

import (
	"context"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

// ...
```

<!-- archie:ai-end -->
