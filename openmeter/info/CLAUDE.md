# info

<!-- archie:ai-start -->

> Organisational namespace for the info domain. Currently contains only openmeter/info/httpdriver, which exposes stateless endpoints (e.g. currency list) via the httptransport pattern with no domain service dependency — data is sourced directly from external libraries (gobl/currency) or static definitions.

## Patterns

**Dependency-free stateless handlers** — When an endpoint's data comes from a static library rather than a DB-backed service, the handler struct holds no injected service. Avoid adding service fields to keep these handlers trivially testable. (`type handler struct{} // no fields; operation func calls gobl/currency.All() directly`)
**httptransport.NewHandler triple** — Every endpoint follows the decoder/operation/encoder triple: decoder extracts HTTP params, operation is pure business logic, encoder serialises the response. (`httptransport.NewHandler(decode, operate, encode, opts...)`)
**No-op request decoder for parameterless endpoints** — Endpoints with no parameters use a decoder that returns an empty struct and nil error rather than reading from the request. (`func(r *http.Request) (struct{}, error) { return struct{}{}, nil }`)
**Handler interface + private handler struct** — driver.go declares the public Handler interface; per-resource files (like currencies.go) implement a private handler struct with a New constructor. (`type Handler interface { ListCurrencies() http.Handler }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/info/httpdriver/driver.go` | Public Handler interface declaration and New constructor — the only symbols exported to callers (router.go). | New endpoints must be added to the Handler interface here before being implemented in a per-resource file. |
| `openmeter/info/httpdriver/currencies.go` | ListCurrencies endpoint implementation — canonical example of a no-service stateless handler sourcing data from gobl/currency. | Do not inject a service or adapter; if new data requires DB access it belongs in a new domain package, not here. |

## Anti-Patterns

- Injecting a domain service or adapter into a handler struct when data comes from a static library.
- Returning a raw error from the operation func instead of a typed domain error (models.GenericValidationError, etc.) — breaks the generic error encoder chain.
- Hand-editing generated API types in api/ instead of regenerating via make gen-api.
- Adding request validation logic inside the decoder func instead of returning a validation error from the operation func.

## Decisions

- **No domain service or Wire provider needed for info/httpdriver** — All current endpoints are backed by static library calls; adding a Wire provider for purely static data would add unnecessary DI boilerplate.

<!-- archie:ai-end -->
