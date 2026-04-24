# router

<!-- archie:ai-start -->

> Thin v1 HTTP routing layer that implements api.ServerInterface by delegating every endpoint to a typed domain handler retrieved from router.Config. It owns no business logic — its sole constraint is that every method body must terminate with a single handler call ending in .ServeHTTP(w, r).

## Patterns

**Handler delegation via .With().ServeHTTP()** — Every Router method unpacks path/query params into a typed Params struct, calls the handler's named method (e.g. a.billingHandler.GetInvoice()), chains .With(params) when params exist, and ends with .ServeHTTP(w, r). No business logic lives here. (`a.billingHandler.GetInvoice().With(httpdriver.GetInvoiceParams{InvoiceID: invoiceId, Expand: lo.FromPtr(params.Expand)}).ServeHTTP(w, r)`)
**Config aggregates all domain service interfaces** — router.Config holds every domain service (billing.Service, customer.Service, etc.) and config flags. NewRouter constructs all domain handlers from Config fields, injecting staticNamespaceDecoder and httptransport.WithErrorHandler(config.ErrorHandler) into every handler constructor. (`router.customerHandler = customerhttpdriver.New(staticNamespaceDecoder, config.Customer, config.SubscriptionService, config.EntitlementConnector, httptransport.WithErrorHandler(config.ErrorHandler))`)
**StaticNamespaceDecoder injected universally** — All multi-tenant handlers receive namespacedriver.StaticNamespaceDecoder(config.NamespaceManager.GetDefaultNamespace()) as their first argument so namespace is resolved once at construction time for self-hosted deployments. (`staticNamespaceDecoder := namespacedriver.StaticNamespaceDecoder(config.NamespaceManager.GetDefaultNamespace())`)
**One file per domain resource** — Each resource area has its own file (billing.go, customer.go, meter.go, etc.) containing only the Router methods for that domain. router.go contains Config, Router struct, NewRouter, and init(). (`billing.go holds all /api/v1/billing/* methods; meter.go holds all /api/v1/meters/* methods`)
**Stub unimplemented endpoints with 501** — Methods that are not yet implemented return w.WriteHeader(http.StatusNotImplemented) directly rather than delegating. noop.go validates the Router still satisfies api.ServerInterface via var unimplemented api.ServerInterface = api.Unimplemented{}. (`func (a *Router) VoidInvoiceAction(w http.ResponseWriter, r *http.Request, invoiceId string) { w.WriteHeader(http.StatusNotImplemented) }`)
**Pre-handler body reads for non-standard content** — When a handler needs raw bytes before delegation (e.g. Stripe webhook payload, CloudEvents), the Router method reads/limits the body itself and passes it via a Params struct. Only do this for content that cannot be decoded by the httpdriver. (`r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes); payload, err := io.ReadAll(r.Body); ... .With(AppStripeWebhookParams{Payload: payload}).ServeHTTP(w, r)`)
**Media-type dispatch within one endpoint** — QueryMeter and QueryMeterPost use commonhttp.GetMediaType to choose between JSON and CSV handler variants on the same route. All other routes use a single handler. (`if mediatype == "text/csv" { a.meterHandler.QueryMeterCSV().With(handlerParams).ServeHTTP(w, r); return }; a.meterHandler.QueryMeter().With(handlerParams).ServeHTTP(w, r)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `router.go` | Defines Config (all injected services), Router struct (all handler fields), NewRouter constructor that wires handlers, and init() that registers CloudEvents body decoders for kin-openapi. | Config.Validate() panics if any required service is nil — always add new fields here AND add a nil check in Validate. The compile-time assertion var _ api.ServerInterface = (*Router)(nil) will catch missing method implementations. |
| `noop.go` | Declares var unimplemented api.ServerInterface = api.Unimplemented{} to ensure the generated Unimplemented type is valid; does not contain real stubs. | Do not remove; it anchors the contract check for the generated fallback type. |
| `portal.go` | QueryPortalMeter is a legacy handler that manually extracts the authenticated subject and re-dispatches to QueryMeter. It does not use the httpdriver pattern. | This is the only Router method that contains conditional auth logic and context manipulation — do not replicate this pattern for new endpoints. |
| `meter.go` | Contains the only media-type dispatch (JSON vs CSV) via commonhttp.GetMediaType. All other domain files are pure delegation. | Logging the media-type parse error at Debug level is intentional — it is non-fatal and falls through to JSON default. |
| `billing.go` | Invoice progress actions (advance, approve, retry, snapshot-quantities) all use the same ProgressInvoice handler with an action discriminator argument rather than separate handlers. | httpdriver.InvoiceProgressActionAdvance etc. are typed constants — pass the correct constant, not a string literal. |

## Anti-Patterns

- Adding business logic, error handling, or database calls directly in a Router method — all logic belongs in the domain httpdriver package.
- Reading r.Body in a Router method unless the httpdriver cannot decode it (e.g. raw webhook bytes) — let the handler's RequestDecoder own body parsing.
- Skipping Config.Validate() nil check when adding a new service field — will cause a nil-pointer panic at handler construction time.
- Creating a new file in this package for non-HTTP concerns (e.g. middleware, auth) — middleware belongs in openmeter/server/server.go, not the router package.
- Bypassing the .With().ServeHTTP() chain by writing an inline http.Handler or calling w.WriteHeader/w.Write for anything other than stubs — breaks the uniform error encoding contract.

## Decisions

- **Router is a pure delegation layer with no business logic** — Keeps the generated api.ServerInterface boundary thin so the TypeSpec → OpenAPI → oapi-codegen pipeline can be re-run without touching any business logic. All logic lives in testable httpdriver packages.
- **Config aggregates every domain service and config flag into a single validated struct** — NewRouter is the single wiring point for all ~25 domain handlers; a nil check at construction time surfaces missing dependencies before the server starts rather than at first request.
- **One domain file per resource group** — Mirrors the TypeSpec package split (billing, customer, entitlement, etc.) so contributors can find the delegation for any endpoint without grepping the entire file.

## Example: Adding a new v1 endpoint for a domain that already has a handler in Config

```
// In the appropriate domain file (e.g. billing.go):
func (a *Router) GetMyNewResource(w http.ResponseWriter, r *http.Request, id string, params api.GetMyNewResourceParams) {
	a.billingHandler.GetMyNewResource().With(httpdriver.GetMyNewResourceParams{
		ID:     id,
		Expand: lo.FromPtr(params.Expand),
	}).ServeHTTP(w, r)
}
// No other changes needed in this package — the httpdriver method and TypeSpec → gen-api step are the real work.
```

<!-- archie:ai-end -->
