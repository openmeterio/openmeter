# router

<!-- archie:ai-start -->

> Pure v1 HTTP delegation layer implementing the generated api.ServerInterface: every Router method unpacks path/query params and terminates with a single typed handler call via .With(params).ServeHTTP(w, r). Owns no business logic — its constraint is that each method body ends with exactly one handler dispatch.

## Patterns

**Handler delegation via .With().ServeHTTP()** — Every Router method unpacks path/query params into a typed Params struct, calls the domain handler's named method (e.g. a.billingHandler.GetInvoice()), optionally chains .With(params), and ends with .ServeHTTP(w, r). No business logic lives here. (`a.billingHandler.GetInvoice().With(httpdriver.GetInvoiceParams{InvoiceID: invoiceId, Expand: lo.FromPtr(params.Expand)}).ServeHTTP(w, r)`)
**Config aggregates all domain services with nil validation** — router.Config holds every domain service (~40 interface fields). NewRouter calls config.Validate() before constructing handlers; each required field has an explicit nil check that errors at startup, not at first request. (`if c.NamespaceManager == nil { return errors.New("namespace manager is required") }`)
**StaticNamespaceDecoder injected universally** — All multi-tenant handlers receive namespacedriver.StaticNamespaceDecoder(config.NamespaceManager.GetDefaultNamespace()) as their first constructor argument so namespace resolution happens once at construction for self-hosted single-namespace deployments. (`staticNamespaceDecoder := namespacedriver.StaticNamespaceDecoder(config.NamespaceManager.GetDefaultNamespace())`)
**One domain file per resource group** — Each resource area has its own file (billing.go, customer.go, meter.go, entitlement.go, etc.) holding only that domain's Router methods. router.go holds Config, Router, NewRouter, and init(). (`billing.go holds all /api/v1/billing/* methods; meter.go holds all /api/v1/meters/* methods`)
**Stub unimplemented endpoints with 501** — Methods not yet implemented return w.WriteHeader(http.StatusNotImplemented) directly. noop.go anchors var unimplemented api.ServerInterface = api.Unimplemented{}. (`func (a *Router) VoidInvoiceAction(w http.ResponseWriter, r *http.Request, invoiceId string) { w.WriteHeader(http.StatusNotImplemented) }`)
**Pre-handler body read for non-standard content only** — Router methods read r.Body themselves only when the httpdriver cannot decode the content (e.g. raw Stripe webhook bytes), limiting via http.MaxBytesReader then passing via a Params struct. (`r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes); payload, err := io.ReadAll(r.Body); ... .With(AppStripeWebhookParams{AppID: appID, Payload: payload}).ServeHTTP(w, r)`)
**Media-type dispatch on same route (QueryMeter only)** — QueryMeter/QueryMeterPost use commonhttp.GetMediaType to pick JSON vs CSV handler variants on the same route; the parse error is Debug-logged (non-fatal, falls through to JSON). All other routes use a single handler. (`mediatype, _ := commonhttp.GetMediaType(r); if mediatype == "text/csv" { a.meterHandler.QueryMeterCSV().With(handlerParams).ServeHTTP(w, r); return }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `router.go` | Defines Config (all injected services + feature flags), Router struct (all handler fields), NewRouter (validates and wires ~25 handlers), and init() registering CloudEvents body decoders (application/cloudevents+json, application/cloudevents-batch+json) for kin-openapi validation. | Adding a service to Config requires: (1) a Config field, (2) a nil check in Config.Validate(), (3) a Router handler field, (4) construction in NewRouter. Missing any step causes a nil-pointer panic at first request. var _ api.ServerInterface = (*Router)(nil) catches missing methods. |
| `noop.go` | Declares var unimplemented api.ServerInterface = api.Unimplemented{} so the generated Unimplemented fallback stays valid. No real stubs. | Do not remove — it anchors the compile-time check; if generated api.Unimplemented{} diverges after gen-api this line fails to compile. |
| `portal.go` | QueryPortalMeter is the only Router method with conditional auth: extracts the authenticated subject from ctx via authenticator.GetAuthenticatedSubject, injects it as a filter, and re-dispatches to QueryMeter. Other portal methods are pure delegation. | The only Router method manipulating ctx with conditional logic — do not replicate; it should migrate to an httpdriver handler. |
| `billing.go` | Invoice progress actions (advance, approve, retry, snapshot-quantities) all use the same ProgressInvoice handler with a typed action constant argument instead of separate handlers. | Pass httpdriver.InvoiceProgressActionAdvance/Approve/Retry/SnapshotQuantities constants — never string literals. |
| `appstripe.go` | Canonical pre-handler body reading: reads the raw Stripe webhook payload with MaxBytesReader(65536) before passing to appStripeHandler. | Only follow MaxBytesReader + io.ReadAll + bytes-in-Params for new raw-byte webhooks; do not call io.ReadAll in any other Router method. |
| `addon.go` | Example domain file: each method is a thin .With(...).ServeHTTP delegation to addonHandler (e.g. GetAddon maps params into addonhttpdriver.GetAddonRequestParams). | Keep param mapping declarative — no defaulting logic beyond lo.FromPtrOr. |

## Anti-Patterns

- Adding business logic, error handling, or DB calls in a Router method — logic belongs in the domain httpdriver package
- Reading r.Body unless the httpdriver cannot decode it — let the handler's RequestDecoder own body parsing
- Skipping the Config.Validate() nil check when adding a required service field
- Creating a file here for non-HTTP concerns (middleware, auth, telemetry) — those belong in openmeter/server/server.go
- Bypassing .With().ServeHTTP() with an inline http.Handler or w.WriteHeader/w.Write (except 501 stubs) — breaks uniform error encoding and OTel tracing

## Decisions

- **Router is a pure delegation layer with no business logic** — Keeps the generated api.ServerInterface boundary thin so the TypeSpec -> OpenAPI -> oapi-codegen pipeline can re-run without touching logic; all logic lives in unit-testable httpdriver packages.
- **Config aggregates every domain service and flag into a single validated struct** — NewRouter is the single wiring point for ~25 handlers; nil-checking at construction surfaces missing dependencies before the server starts, making startup failures deterministic.
- **One domain file per resource group mirrors the TypeSpec package split** — Contributors locate any endpoint's delegation without grepping; file boundaries align with the aip/legacy TypeSpec organization.

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
