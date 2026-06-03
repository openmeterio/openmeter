# addons

<!-- archie:ai-start -->

> v3 HTTP handlers for addon CRUD plus lifecycle (publish/archive), translating apiv3.Addon wire types to addon.Service domain calls through the httptransport pipeline. Each operation lives in its own file.

## Patterns

**One operation per file** — Each CRUD/lifecycle operation has its own file (create.go, delete.go, get.go, list.go, update.go, archive.go, publish.go). handler.go only declares the Handler interface and New() constructor. (`func (h *handler) ArchiveAddon() ArchiveAddonHandler { return httptransport.NewHandlerWithArgs(...) }`)
**Type-alias triad per operation** — Every operation file declares <Op>Request = addon.<Op>Input, <Op>Response = apiv3.Type, and <Op>Handler = httptransport.Handler[Req,Resp] (or HandlerWithArgs for path-param ops). (`type (ArchiveAddonRequest = addon.ArchiveAddonInput; ArchiveAddonResponse = apiv3.Addon; ArchiveAddonHandler httptransport.HandlerWithArgs[ArchiveAddonRequest, ArchiveAddonResponse, string])`)
**resolveNamespace first in every decoder** — Decoders call ns, err := h.resolveNamespace(ctx) first and propagate the error before building the domain input. (`ns, err := h.resolveNamespace(ctx); if err != nil { return ArchiveAddonRequest{}, err }`)
**apierrors.GenericErrorEncoder as base error handler** — Every httptransport.AppendOptions includes WithErrorEncoder(apierrors.GenericErrorEncoder()) and WithOperationName(<kebab-name>). (`httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder())`)
**convert.go owns all domain<->API mapping** — All ToAPI*/FromAPI* functions live in convert.go; operation files call them and never inline-convert. BillingPrice/RateCard discriminated unions are encoded manually via Discriminator()/From* calls. (`return ToAPIAddon(*a)`)
**Nil-check pointer results before conversion** — Handlers receiving *addon.Addon guard against nil before ToAPIAddon, returning a descriptive error. (`if a == nil { return ArchiveAddonResponse{}, fmt.Errorf("failed to archive add-on") }`)
**IgnoreNonCriticalIssues on mutating inputs** — Create and update decoders set req.IgnoreNonCriticalIssues = true after FromAPI conversion to suppress non-fatal validation issues. (`req.IgnoreNonCriticalIssues = true`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Handler interface listing all endpoint methods + handler struct (resolveNamespace func, service addon.Service, options []httptransport.HandlerOption) + New() constructor. | New endpoint = add to the Handler interface here first, then create a dedicated file. |
| `convert.go` | All bidirectional apiv3<->productcatalog/addon mapping: rate cards, prices (flat/unit/tiered graduated+volume), tax config, discounts, labels, status. | Rate-card and price type switches are exhaustive — a new RateCardType or PriceType needs a case in both ToAPIBillingRateCard/ToAPIBillingPrice and FromAPIBillingRateCard. FlatFeeRateCard leaves BillingCadence nil; UsageBased sets it. Missing Price => free price. |
| `archive.go` | ArchiveAddon sets EffectiveTo: clock.Now() in the decoder — the HTTP layer owns wall-clock time. | Lifecycle ops (publish, archive) must set clock.Now() in the decoder, never inside the service. |
| `list.go` | ListAddons with page pagination (pagination.NewPage defaults 1,20), validating page before building ListAddonsInput. | page.Validate errors must surface as apierrors.NewBadRequestError with Field='page'. |

## Anti-Patterns

- Inline type conversion in operation files instead of calling ToAPIAddon/FromAPICreate*
- Returning domain errors without the httptransport GenericErrorEncoder chain
- Skipping h.resolveNamespace(ctx) in a new operation decoder
- Using httptransport.NewHandler instead of NewHandlerWithArgs for path-param operations
- Adding validation/defaults logic outside the domain service or convert.go

## Decisions

- **Lifecycle operations set clock.Now() in the decoder, not the service** — HTTP layer owns request-scoped wall-clock time; the domain service receives explicit timestamps, making it deterministic and testable.
- **convert.go hand-codes BillingPrice/RateCard union encoding instead of goverter** — BillingPrice is a oneOf requiring Discriminator()+From* calls goverter cannot generate; a manual switch is the only safe option.

## Example: Add a new addon path-param lifecycle endpoint

```
// restore.go
package addons
import (
    "context"; "fmt"; "net/http"
    apiv3 "github.com/openmeterio/openmeter/api/v3"
    "github.com/openmeterio/openmeter/api/v3/apierrors"
    "github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
    "github.com/openmeterio/openmeter/pkg/clock"
    "github.com/openmeterio/openmeter/pkg/framework/commonhttp"
    "github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
    "github.com/openmeterio/openmeter/pkg/models"
)
type (
    RestoreAddonRequest  = addon.RestoreAddonInput
    RestoreAddonResponse = apiv3.Addon
// ...
```

<!-- archie:ai-end -->
