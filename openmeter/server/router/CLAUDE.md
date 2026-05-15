# router

<!-- archie:ai-start -->

> Pure v1 HTTP delegation layer implementing api.ServerInterface: every Router method unpacks path/query params and ends with a single typed handler call via .With(params).ServeHTTP(w, r). Owns no business logic — its only constraint is that every method body terminates with one handler dispatch.

## Patterns

**Handler delegation via .With().ServeHTTP()** — Every Router method unpacks path/query params into a typed Params struct, calls the domain handler's named method (e.g. a.billingHandler.GetInvoice()), optionally chains .With(params), and ends with .ServeHTTP(w, r). No business logic lives here. (`a.billingHandler.GetInvoice().With(httpdriver.GetInvoiceParams{InvoiceID: invoiceId, Expand: lo.FromPtr(params.Expand)}).ServeHTTP(w, r)`)
**Config aggregates all domain service interfaces with nil validation** — router.Config holds every domain service (~40 interface fields). NewRouter calls config.Validate() before constructing handlers; each required field has an explicit nil check that returns an error at startup, not at first request. (`if c.NamespaceManager == nil { return errors.New("namespace manager is required") }`)
**StaticNamespaceDecoder injected universally into every handler constructor** — All multi-tenant handlers receive namespacedriver.StaticNamespaceDecoder(config.NamespaceManager.GetDefaultNamespace()) as their first argument so namespace resolution happens once at construction time for self-hosted single-namespace deployments. (`staticNamespaceDecoder := namespacedriver.StaticNamespaceDecoder(config.NamespaceManager.GetDefaultNamespace())
router.customerHandler = customerhttpdriver.New(staticNamespaceDecoder, config.Customer, ...)`)
**One domain file per resource group** — Each resource area has its own file (billing.go, customer.go, meter.go, entitlement.go, etc.) containing only the Router methods for that domain. router.go contains Config, Router struct, NewRouter, and init(). (`billing.go holds all /api/v1/billing/* methods; meter.go holds all /api/v1/meters/* methods`)
**Stub unimplemented endpoints with w.WriteHeader(501)** — Methods not yet implemented return w.WriteHeader(http.StatusNotImplemented) directly. noop.go validates the Router still satisfies api.ServerInterface via var unimplemented api.ServerInterface = api.Unimplemented{}. (`func (a *Router) VoidInvoiceAction(w http.ResponseWriter, r *http.Request, invoiceId string) { w.WriteHeader(http.StatusNotImplemented) }`)
**Pre-handler body read for non-standard content only** — Router methods read r.Body themselves only when the httpdriver cannot decode the content (e.g. raw Stripe webhook bytes). The body is limited via http.MaxBytesReader then passed via a Params struct. (`r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes); payload, err := io.ReadAll(r.Body); ... .With(AppStripeWebhookParams{AppID: appID, Payload: payload}).ServeHTTP(w, r)`)
**Media-type dispatch on same route (QueryMeter only)** — QueryMeter and QueryMeterPost use commonhttp.GetMediaType to choose between JSON and CSV handler variants on the same route. The media-type parse error is logged at Debug level (non-fatal, falls through to JSON default). All other routes use a single handler. (`mediatype, _ := commonhttp.GetMediaType(r); if mediatype == "text/csv" { a.meterHandler.QueryMeterCSV().With(handlerParams).ServeHTTP(w, r); return }; a.meterHandler.QueryMeter().With(handlerParams).ServeHTTP(w, r)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `router.go` | Defines Config (all injected services + feature flags), Router struct (all handler fields), NewRouter constructor that validates and wires ~25 handlers, and init() that registers CloudEvents body decoders (application/cloudevents+json, application/cloudevents-batch+json) for kin-openapi request validation. | Adding a new service to Config requires: (1) a field in Config, (2) a nil check in Config.Validate(), (3) a handler field in Router struct, and (4) handler construction in NewRouter. Missing any step causes a nil-pointer panic at first request. The compile-time assertion var _ api.ServerInterface = (*Router)(nil) catches missing method implementations. |
| `noop.go` | Declares var unimplemented api.ServerInterface = api.Unimplemented{} to ensure the generated Unimplemented type remains valid. Contains no real stubs. | Do not remove — it anchors the compile-time check for the generated fallback type. If the generated api.Unimplemented{} diverges from api.ServerInterface after gen-api, this line fails to compile. |
| `portal.go` | QueryPortalMeter is the only Router method with conditional auth logic: it extracts the authenticated subject from ctx via authenticator.GetAuthenticatedSubject, injects it as a filter, and re-dispatches to QueryMeter. All other portal methods are pure delegation. | This is the only Router method that manipulates ctx and contains conditional logic. Do not replicate this pattern for new endpoints — it should eventually be migrated to an httpdriver handler. |
| `billing.go` | Invoice progress actions (advance, approve, retry, snapshot-quantities) all use the same ProgressInvoice handler with a typed action constant argument instead of separate handlers. | Pass httpdriver.InvoiceProgressActionAdvance / InvoiceProgressActionApprove / InvoiceProgressActionRetry / InvoiceProgressActionSnapshotQuantities constants — do not pass string literals or magic values. |
| `appstripe.go` | Contains the canonical example of pre-handler body reading: reads raw Stripe webhook payload with MaxBytesReader(65536) before passing to appStripeHandler. | If a new webhook endpoint needs raw bytes, follow this pattern (MaxBytesReader + io.ReadAll + pass bytes in Params). Do not call io.ReadAll in any other Router method. |

## Anti-Patterns

- Adding business logic, error handling, or database calls directly in a Router method — all logic belongs in the domain httpdriver package.
- Reading r.Body in a Router method unless the httpdriver cannot decode it (e.g. raw webhook bytes) — let the handler's RequestDecoder own body parsing.
- Skipping the Config.Validate() nil check when adding a new required service field — causes nil-pointer panic at handler construction or first request.
- Creating a new file in this package for non-HTTP concerns (middleware, auth, telemetry) — middleware belongs in openmeter/server/server.go, not the router package.
- Bypassing the .With().ServeHTTP() chain by writing an inline http.Handler or calling w.WriteHeader/w.Write for anything other than 501 stubs — breaks the uniform error encoding contract and OTel tracing.

## Decisions

- **Router is a pure delegation layer with no business logic** — Keeps the generated api.ServerInterface boundary thin so the TypeSpec → OpenAPI → oapi-codegen pipeline can be re-run without touching any business logic. All logic lives in testable httpdriver packages that can be unit-tested without an HTTP server.
- **Config aggregates every domain service and config flag into a single validated struct** — NewRouter is the single wiring point for all ~25 domain handlers; nil-checking at construction time surfaces missing dependencies before the server starts rather than at first request, making startup failures deterministic.
- **One domain file per resource group mirrors TypeSpec package split** — Contributors can locate the delegation for any endpoint without grepping the whole file, and the file boundaries align with the TypeSpec aip/legacy package organization (billing, customer, entitlement, meter, etc.).

## Example: Adding a new v1 endpoint for a domain that already has a handler in Config

```
// 1. Edit TypeSpec in api/spec/packages/legacy/ to add the operation
// 2. Run: make gen-api && make generate
// 3. Add the Router method in the appropriate domain file (e.g. billing.go):
func (a *Router) GetMyNewResource(w http.ResponseWriter, r *http.Request, id string, params api.GetMyNewResourceParams) {
    a.billingHandler.GetMyNewResource().With(httpdriver.GetMyNewResourceParams{
        ID:     id,
        Expand: lo.FromPtr(params.Expand),
    }).ServeHTTP(w, r)
}
// No other changes in this package — the httpdriver method is the real work.
```

<!-- archie:ai-end -->
