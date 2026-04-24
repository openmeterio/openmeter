# addons

<!-- archie:ai-start -->

> HTTP handlers for the v3 addon CRUD + lifecycle (publish/archive) endpoints, translating between apiv3.Addon wire types and addon.Service domain calls via the httptransport pattern.

## Patterns

**One operation per file** — Each CRUD or lifecycle operation lives in its own file (create.go, delete.go, get.go, list.go, update.go, archive.go, publish.go). handler.go only declares the Handler interface and constructor. (`func (h *handler) ArchiveAddon() ArchiveAddonHandler { return httptransport.NewHandlerWithArgs(...) }`)
**Type alias triad per operation** — Every operation file declares three type aliases: <Op>Request = domain.Input, <Op>Response = apiv3.Type, <Op>Handler = httptransport.Handler[Req,Resp] (or HandlerWithArgs for path-param operations). (`type ( ArchiveAddonRequest = addon.ArchiveAddonInput; ArchiveAddonResponse = apiv3.Addon; ArchiveAddonHandler httptransport.HandlerWithArgs[...] )`)
**resolveNamespace in every decoder** — All decoders call h.resolveNamespace(ctx) as the first step and propagate the error before building the domain input struct. (`ns, err := h.resolveNamespace(ctx); if err != nil { return ArchiveAddonRequest{}, err }`)
**apierrors.GenericErrorEncoder as base error handler** — Every httptransport.AppendOptions call includes httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()) as the error handler. (`httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder())`)
**convert.go owns all domain<->API mapping** — All ToAPI* and FromAPI* conversion functions are in convert.go; individual operation files call these helpers, never inline-converting types. (`return ToAPIAddon(*a) // in operation handler body`)
**Nil-check after service call** — Handlers that receive a pointer result (*addon.Addon) guard against nil before calling ToAPIAddon, returning a descriptive error instead. (`if a == nil { return ArchiveAddonResponse{}, fmt.Errorf("failed to archive add-on") }`)
**IgnoreNonCriticalIssues on mutating inputs** — Create and update decoders set req.IgnoreNonCriticalIssues = true after FromAPI conversion to suppress non-fatal validation issues. (`req.IgnoreNonCriticalIssues = true`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Declares the Handler interface listing all endpoint methods and the handler struct with resolveNamespace, service addon.Service, and options []httptransport.HandlerOption. New() constructor. | Adding a new endpoint: add it to the Handler interface here first, then create a dedicated file. |
| `convert.go` | All bidirectional conversion between apiv3 and productcatalog/addon domain types including rate cards, prices (flat/unit/tiered), tax config, discounts, and labels. | Rate card type switch is exhaustive — adding a new RateCardType requires a new case in both ToAPIBillingRateCard and FromAPIBillingRateCard. |
| `list.go` | ListAddons with page-pagination using pagination.NewPage defaults (1, 20); validates page before building ListAddonsInput. | Default page size is 20; page validation errors must use apierrors.NewBadRequestError with field='page'. |

## Anti-Patterns

- Inline type conversion in operation files instead of calling ToAPIAddon/FromAPICreate*
- Returning domain errors directly without httptransport error encoder chain
- Skipping h.resolveNamespace(ctx) call in a new operation decoder
- Using httptransport.NewHandler instead of NewHandlerWithArgs for path-param operations
- Adding business logic (validation, defaults) outside of the domain service or convert.go

## Decisions

- **Lifecycle operations (publish, archive) set clock.Now() in the decoder, not the service.** — HTTP layer owns wall-clock time for request-scoped operations; domain service receives explicit timestamps making it easier to test.
- **convert.go handles all price discriminated union encoding/decoding manually (not goverter).** — BillingPrice is a oneOf union requiring Discriminator() + From* calls that goverter cannot generate; manual switch-case is the only safe option.

## Example: Add a new addon lifecycle endpoint (e.g., RestoreAddon)

```
// handler.go: add RestoreAddon() RestoreAddonHandler to Handler interface
// restore.go:
package addons
import (
	"context"
	"net/http"
	apiv3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)
type (
	RestoreAddonRequest  = addon.RestoreAddonInput
// ...
```

<!-- archie:ai-end -->
