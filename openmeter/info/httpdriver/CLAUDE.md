# httpdriver

<!-- archie:ai-start -->

> HTTP driver for the 'info' domain — exposes static reference/metadata endpoints (currently only the currency list) to the server router. Stateless: it has no service/adapter layer behind it and derives data directly from the gobl `currency` library.

## Patterns

**Handler interface + private struct + New constructor** — driver.go declares a public `Handler` interface listing one method per endpoint, an unexported `handler` struct holding shared `[]httptransport.HandlerOption`, and a `New(options ...httptransport.HandlerOption) Handler` constructor returning the interface. New endpoints add a method to the interface and a method on `*handler`. (`type Handler interface { ListCurrencies() ListCurrenciesHandler }`)
**Request/Response/Handler type triple per endpoint** — Each endpoint defines three named types in a `type (...)` block: a `XxxRequest` struct (input, may be empty), a `XxxResponse` (output, here an `[]api.Currency` alias), and `XxxHandler = httptransport.Handler[XxxRequest, XxxResponse]`. (`ListCurrenciesRequest struct{}; ListCurrenciesResponse []api.Currency; ListCurrenciesHandler httptransport.Handler[ListCurrenciesRequest, ListCurrenciesResponse]`)
**httptransport.NewHandler with decode/business/encode closures** — Methods return `httptransport.NewHandler(decode, handle, encode, opts...)`. The decode func builds the Request from `*http.Request`, the handle func produces the Response, encode is `commonhttp.JSONResponseEncoderWithStatus[Resp](http.StatusOK)`, and options come from `httptransport.AppendOptions(h.options, httptransport.WithOperationName("..."))`. (`httptransport.NewHandler(func(ctx, r){...}, func(ctx, req){...}, commonhttp.JSONResponseEncoderWithStatus[ListCurrenciesResponse](http.StatusOK), httptransport.AppendOptions(h.options, httptransport.WithOperationName("listCurrencies"))...)`)
**Operation name matches OpenAPI operationId** — `WithOperationName` must use the camelCase operationId from the generated API spec (e.g. "listCurrencies") so telemetry/routing line up with `api/`. (`httptransport.WithOperationName("listCurrencies")`)
**Map domain data to api.* types with samber/lo** — Business closures translate library types into generated `api.*` structs using `lo.Map`/`lo.Filter`; ISO-only currencies are kept by filtering `def.ISONumeric != ""` to exclude crypto. (`lo.Map(lo.Filter(currency.Definitions(), func(d *currency.Def, _ int) bool { return d.ISONumeric != "" }), func(d *currency.Def, _ int) api.Currency { ... })`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `driver.go` | Declares the `Handler` interface, the `handler` struct, and `New(...)`. The single place to register a new endpoint method on the interface. | Keep `handler` unexported and return the interface from `New`; the router depends on `Handler`, not the concrete struct. |
| `currencies.go` | Implements `ListCurrencies()` — returns ISO currency definitions from gobl as `[]api.Currency`. | Preserve the `def.ISONumeric != ""` filter (drops crypto/non-ISO entries); changing it leaks non-ISO currencies into the API response. |

## Anti-Patterns

- Putting business/data-access logic in handlers that warrants a service+adapter layer — this folder is intentionally driver-only and stateless; if state or DB access is needed, introduce a service package instead.
- Returning the concrete `*handler` instead of the `Handler` interface from `New`.
- Hand-writing response structs instead of mapping into generated `api.*` types.
- Hardcoding HTTP status or bypassing `commonhttp.JSONResponseEncoderWithStatus` for encoding.
- Omitting `WithOperationName` or using a name that doesn't match the OpenAPI operationId.

## Decisions

- **No service/adapter layer for the info domain.** — Data is static reference metadata sourced from the gobl `currency` library, so the handler computes it inline with no persistence or tenancy concerns.

## Example: Adding a new info endpoint following the existing pattern

```
type (
	ListCurrenciesRequest  struct{}
	ListCurrenciesResponse []api.Currency
	ListCurrenciesHandler  httptransport.Handler[ListCurrenciesRequest, ListCurrenciesResponse]
)

func (h *handler) ListCurrencies() ListCurrenciesHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (ListCurrenciesRequest, error) {
			return ListCurrenciesRequest{}, nil
		},
		func(ctx context.Context, request ListCurrenciesRequest) (ListCurrenciesResponse, error) {
			return lo.Map(lo.Filter(currency.Definitions(),
				func(def *currency.Def, _ int) bool { return def.ISONumeric != "" }),
				func(def *currency.Def, _ int) api.Currency {
// ...
```

<!-- archie:ai-end -->
