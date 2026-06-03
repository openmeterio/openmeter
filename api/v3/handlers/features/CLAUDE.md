# features

<!-- archie:ai-start -->

> Full-CRUD v3 HTTP handler package for product-catalog features (list, get, create, update, delete/archive), with best-effort LLM pricing enrichment on read responses and feature-domain-specific HTTP status mapping via a local errorEncoder.

## Patterns

**Per-operation file split** — Each verb has its own file (create.go, get.go, list.go, update.go, delete.go). handler.go declares the Handler interface + New(); convert.go holds all conversions; error_encoder.go holds domain-error mapping. (`func (h *handler) CreateFeature() CreateFeatureHandler { return httptransport.NewHandler(...) }`)
**Hand-written conversions with unit tests** — convert.go is hand-coded (no goverter) and covered by convert_test.go; conversion functions return (T, error) to propagate parse failures. (`func convertFeatureToAPI(f feature.Feature) (api.Feature, error)`)
**Local errorEncoder chained after GenericErrorEncoder** — AppendOptions includes WithErrorEncoder(apierrors.GenericErrorEncoder()) then WithErrorEncoder(errorEncoder()); errorEncoder maps feature errors (FeatureInvalidFiltersError->400, ForbiddenError->403, FeatureWithNameAlreadyExistsError->409) via commonhttp.HandleErrorIfTypeMatches. (`httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()), httptransport.WithErrorEncoder(errorEncoder())`)
**Best-effort LLM pricing enrichment on reads** — After converting, if UnitCost.Type == UnitCostTypeLLM and h.llmcostService != nil, call resolveLLMPricing and enrichFeatureResponseWithPricing. Pricing errors are silently swallowed (resolveLLMPricing returns nil). Applied in get.go and update.go. (`if feat.UnitCost != nil && feat.UnitCost.Type == feature.UnitCostTypeLLM && h.llmcostService != nil { if p := resolveLLMPricing(ctx, h.llmcostService, feat); p != nil { enrichFeatureResponseWithPricing(&resp, p) } }`)
**Meter validation at decode time** — Meter ID resolution and GroupBy filter-key validation happen in the request decoder (first NewHandler func) using models.NewGenericValidationError, not in the operation func. (`for k := range filters { if _, ok := m.GroupBy[k]; !ok { return models.NewGenericValidationError(...) } }`)
**nullable.Nullable tri-state for PATCH fields** — Update inputs use nullable.Nullable[T]: IsNull() clears, IsSpecified()+Get() sets, neither leaves unchanged. All three branches are tested in convert_test.go. (`if body.UnitCost.IsNull() { input.UnitCost = nullable.NewNullNullable[feature.UnitCost]() } else if body.UnitCost.IsSpecified() { ... }`)
**UnitCost discriminated union via switch with error default** — convertUnitCostToAPI/FromAPI switch on UnitCostType / Discriminator(); the default case returns an error to catch unhandled future types. (`switch u.Type { case feature.UnitCostTypeManual: ...; case feature.UnitCostTypeLLM: ...; default: return out, fmt.Errorf("unknown unit cost type: %s", u.Type) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Handler interface (5 methods) + handler struct: resolveNamespace, connector (feature.FeatureConnector), meterService, llmcostService, options. | llmcostService is optional and may be nil (e.g. credits disabled); guard with != nil before use in get.go/update.go. |
| `convert.go` | All domain<->API conversions plus resolveLLMPricing/enrichFeatureResponseWithPricing and feature/filter conversions (convertFiltersTo/FromAPI). | UnitCost discriminator switch must cover all UnitCostType values; resolveLLMPricing silently returns nil on error — callers must not treat nil as a hard failure. Use FromBillingFeatureManualUnitCost/FromBillingFeatureLLMUnitCost setters, never struct literals. |
| `error_encoder.go` | Maps feature-domain typed errors to HTTP status codes via commonhttp.HandleErrorIfTypeMatches. | Add new feature-domain errors here; unhandled types fall through to 500. |
| `list.go` | List with pagination (default page 1/size 20), filter[meter_id]/[key]/[name], and sort; invalid params -> apierrors.NewBadRequestError. | page.Validate() must run after construction or page=0 slips through. |
| `create.go` | Validates meter reference exists and filter keys match meter GroupBy dimensions (in the decoder) before CreateFeature. | validateMeterFilters checks key existence only; operator validity is validated downstream by MeterGroupByFilters.Validate. |

## Anti-Patterns

- Calling h.connector directly from handler.go instead of delegating to per-operation files
- Skipping errorEncoder() in AppendOptions — feature domain errors become 500
- Returning pricing enrichment errors instead of silently skipping (enrichment is best-effort)
- Accepting llmcostService as a required (non-nil) dependency
- Constructing api.BillingFeatureUnitCost{} literally instead of using the From* setters

## Decisions

- **Hand-written conversions instead of goverter** — UnitCost is a manual/llm discriminated union needing switch-based dispatch goverter cannot generate cleanly; hand-written code is explicit and covered by convert_test.go.
- **LLM pricing resolved at read time, not stored** — Pricing changes independently of feature config; resolving from llmcost.Service at read time keeps freshness without coupling the feature write path to pricing.

## Example: Add a new CRUD operation following the per-operation file pattern

```
// archive.go
package features
import (
  "context"; "net/http"
  "github.com/openmeterio/openmeter/api/v3/apierrors"
  "github.com/openmeterio/openmeter/pkg/framework/commonhttp"
  "github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
  "github.com/openmeterio/openmeter/pkg/models"
)
type (
  ArchiveFeatureRequest = models.NamespacedID
  ArchiveFeatureResponse = any
  ArchiveFeatureHandler  httptransport.HandlerWithArgs[ArchiveFeatureRequest, ArchiveFeatureResponse, string]
)
func (h *handler) ArchiveFeature() ArchiveFeatureHandler {
// ...
```

<!-- archie:ai-end -->
