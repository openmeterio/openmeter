# info

<!-- archie:ai-start -->

> Organisational namespace for the info domain. It currently contains only openmeter/info/httpdriver, which serves stateless endpoints (e.g. currency list) through the httptransport pattern with no domain service, adapter, or Wire provider — data comes directly from external libraries (gobl/currency) or static definitions.

## Patterns

**Dependency-free stateless handlers** — When an endpoint's data is static/library-sourced, the handler struct holds no injected service so it stays trivially testable. (`type handler struct{} // operation func calls gobl/currency.All() directly`)
**httptransport.NewHandler triple** — Every endpoint follows the decoder/operation/encoder triple: decoder extracts params, operation is pure logic, encoder serialises. (`httptransport.NewHandler(decode, operate, encode, opts...)`)
**No-op decoder for parameterless endpoints** — Endpoints without params use a decoder returning an empty struct and nil error rather than reading the request. (`func(r *http.Request) (struct{}, error) { return struct{}{}, nil }`)
**Handler interface + private struct** — driver.go declares the public Handler interface and New constructor; per-resource files (currencies.go) implement a private handler struct. (`type Handler interface { ListCurrencies() http.Handler }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `httpdriver/driver.go` | Public Handler interface and New constructor — the only symbols exported to router.go. | Add new endpoints to the Handler interface here before implementing them in a per-resource file. |
| `httpdriver/currencies.go` | ListCurrencies — canonical no-service stateless handler sourcing data from gobl/currency. | Do not inject a service/adapter; data needing DB access belongs in a new domain package, not here. |

## Anti-Patterns

- Injecting a domain service or adapter when data comes from a static library.
- Returning a raw error from the operation func instead of a typed domain error — breaks the GenericErrorEncoder chain.
- Hand-editing generated API types in api/ instead of regenerating via make gen-api.
- Putting request validation in the decoder func instead of returning a validation error from the operation func.

## Decisions

- **No domain service or Wire provider for info/httpdriver.** — All endpoints are static library-backed; a Wire provider for static data would add needless DI boilerplate.

<!-- archie:ai-end -->
