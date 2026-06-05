# router

<!-- archie:ai-start -->

> Implements the generated legacy v1/v2 api.ServerInterface (api/api.gen.go) as a single *Router type whose methods are thin adapters: each handler method translates oapi-codegen path/query params into a domain httpdriver/httphandler request and calls .With(...).ServeHTTP(w, r). The Router owns no business logic — it only wires per-domain HTTP handlers built in NewRouter from injected services.

## Patterns

**Thin delegating handler methods** — Every (a *Router) Xxx method is one-to-two lines: build the driver's params struct, then a.<domainHandler>.<Operation>().With(params).ServeHTTP(w, r). No business logic, no DB access, no error handling beyond what the driver does. (`func (a *Router) GetInvoice(w, r, invoiceId, params) { a.billingHandler.GetInvoice().With(httpdriver.GetInvoiceParams{InvoiceID: invoiceId, Expand: lo.FromPtr(params.Expand)}).ServeHTTP(w, r) }`)
**One file per domain, methods named after ServerInterface** — Operations are grouped into per-domain files (billing.go, customer.go, entitlement.go, notification.go, etc.). Method names and signatures MUST match the generated api.ServerInterface exactly; the `var _ api.ServerInterface = (*Router)(nil)` assertion in router.go enforces total coverage at compile time. (`func (a *Router) ListAddons(w http.ResponseWriter, r *http.Request, params api.ListAddonsParams)`)
**Param mapping via driver-specific structs** — When a driver needs more than a raw id/params, build the driver's own *Params struct (e.g. httpdriver.GetCustomerOverrideParams, entitlementdriver.GetEntitlementValueHandlerParams) at the call site. Use lo.FromPtr / lo.FromPtrOr to flatten optional api params, never hand-rolled nil checks. (`a.appHandler.ListCustomerData().With(apphttpdriver.ListCustomerDataParams{ListCustomerAppDataParams: params, CustomerIdOrKey: customerIdOrKey})`)
**Handler construction concentrated in NewRouter** — All domain handlers are stored as unexported fields on Router and built once in NewRouter using each driver's New/NewXxxHandler constructor, passing the StaticNamespaceDecoder (default namespace) and httptransport.WithErrorHandler(config.ErrorHandler). No handler is constructed inside a request method. (`router.billingHandler = ... ; staticNamespaceDecoder := namespacedriver.StaticNamespaceDecoder(config.NamespaceManager.GetDefaultNamespace())`)
**Config dependency validation** — Config carries every domain service/connector; Config.Validate() rejects nil required dependencies and NewRouter returns an error if validation fails. Adding a new dependency means adding the field, a nil-check in Validate(), and wiring in NewRouter. (`if c.Billing == nil { return errors.New("billing service is required") }`)
**Unimplemented endpoints explicit** — Routes not yet built return http.StatusNotImplemented inline (VoidInvoiceAction, RecalculateInvoiceTaxAction, MarketplaceOAuth2Install*) rather than relying on api.Unimplemented; noop.go keeps `var unimplemented api.ServerInterface = api.Unimplemented{}` available for future codegen gaps. (`func (a *Router) VoidInvoiceAction(...) { w.WriteHeader(http.StatusNotImplemented) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `router.go` | Defines Config (all injected services), Config.Validate(), the Router struct with all handler fields, NewRouter() which builds every handler, and the api.ServerInterface compile-time assertion. Also registers cloudevents JSON body decoders for openapi3filter in init(). | New endpoints need a corresponding handler field + constructor call here; forgetting the Validate() nil-check or the field breaks DI silently at runtime. Spec validation itself lives in the sibling openmeter/server package, not here. |
| `billing.go` | Invoice lifecycle and billing-profile/customer-override endpoints. Progress actions (advance/approve/retry/snapshot-quantities) all funnel through a.billingHandler.ProgressInvoice(<action>).With(ProgressInvoiceParams{...}). | VoidInvoiceAction and RecalculateInvoiceTaxAction are intentionally 501 Not Implemented — do not assume they work. |
| `entitlement.go` | Largest file: v1 subject-scoped, v1 customer-scoped, v2 entitlement and v2 customer-scoped entitlement endpoints, split across entitlementHandler, meteredEntitlementHandler, entitlementV2Handler, and customerHandler. | Three different handlers serve entitlement routes; pick the one matching v1/v2 and subject/customer scope. Section comment banners delimit the groups. |
| `noop.go` | Holds the api.Unimplemented fallback instance for codegen-defined-but-unbuilt operations. | Prefer explicit StatusNotImplemented inline (per the file comment) over deferring to this for partially-built domains. |
| `portal.go` | Portal token endpoints plus QueryPortalMeter, which authenticates via authenticator.GetAuthenticatedSubject and forwards to a.QueryMeter with the subject filter applied. | QueryPortalMeter is not yet migrated to an httpdriver — it builds api.QueryMeterParams and re-enters the router method directly; keep the subject scoping when editing. |
| `appstripe.go` | Stripe webhook + API key + checkout endpoints. AppStripeWebhook reads the raw body (capped via http.MaxBytesReader) before delegating. | Webhook reads r.Body manually; this is one of the few methods doing IO before delegation — preserve the MaxBytesReader cap and error->StatusProblem handling. |
| `meter.go` | Meter CRUD + query endpoints; QueryMeter/QueryMeterPost branch to CSV vs JSON handlers based on commonhttp.GetMediaType(r). | Content negotiation is done here, not in the driver — text/csv routes to QueryMeterCSV()/QueryMeterPostCSV(). |

## Anti-Patterns

- Putting business logic, DB queries, or validation inside a Router method instead of the domain httpdriver/httphandler.
- Adding a method whose name/signature does not match the generated api.ServerInterface — breaks the `var _ api.ServerInterface = (*Router)(nil)` assertion.
- Constructing a domain handler inside a request method instead of once in NewRouter and storing it as a Router field.
- Adding a new injected dependency to Config without a nil-check in Validate() and wiring in NewRouter.
- Hand-rolling nil/pointer flattening for optional api params instead of lo.FromPtr / lo.FromPtrOr.

## Decisions

- **Router is a pure adapter over oapi-codegen ServerInterface; each method just maps params and delegates to a domain driver.** — Keeps the legacy v1 transport layer mechanical and lets domain packages own request decoding/validation/response shaping, so the generated interface contract is satisfied without duplicating logic.
- **All handlers built in NewRouter from a single Config of injected services using StaticNamespaceDecoder for the default namespace.** — Centralizes DI and namespace resolution; the legacy API operates against the single default namespace, so the decoder is static rather than request-derived.
- **Unbuilt endpoints return explicit 501 inline rather than codegen Unimplemented.** — Makes the not-yet-implemented surface visible and greppable in the per-domain file rather than hidden behind a shared fallback.

## Example: Adding a legacy v1 endpoint: map oapi-codegen params to a driver request and delegate.

```
import (
	"net/http"
	"github.com/samber/lo"
	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/billing/httpdriver"
)

func (a *Router) GetInvoice(w http.ResponseWriter, r *http.Request, invoiceId string, params api.GetInvoiceParams) {
	a.billingHandler.GetInvoice().With(httpdriver.GetInvoiceParams{
		InvoiceID:           invoiceId,
		Expand:              lo.FromPtr(params.Expand),
		IncludeDeletedLines: lo.FromPtr(params.IncludeDeletedLines),
	}).ServeHTTP(w, r)
}
```

<!-- archie:ai-end -->
