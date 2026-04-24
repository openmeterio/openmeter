# server

<!-- archie:ai-start -->

> The v3 HTTP server: wires all domain service dependencies into typed handler structs, validates the Config, registers OpenAPI request validation middleware on a Chi router, and delegates every generated ServerInterface method to the appropriate handler. This is the sole assembly point for the v3 API surface.

## Patterns

**Config validation before NewServer** — Config.Validate() is called at the top of NewServer and returns a wrapped error listing all missing required services. Never skip validation. (`if err := config.Validate(); err != nil { return nil, fmt.Errorf("invalid v3 server config: %w", err) }`)
**credits.enabled gated at route level AND constructor level** — Credits routes in routes.go check both s.Credits.Enabled and s.customersCreditsHandler != nil before dispatching. In NewServer, when credits are disabled, noop implementations replace real services. Both guards are required. (`if !s.Credits.Enabled || s.customersCreditsHandler == nil { unimplemented.GetCustomerCreditBalance(w, r, ...); return }`)
**Handler delegation via .With(params).ServeHTTP(w, r)** — Each ServerInterface method calls the typed handler method, optionally passes path/query params via .With(...), and calls ServeHTTP. No business logic lives in routes.go. (`s.metersHandler.GetMeter().With(meterId).ServeHTTP(w, r)`)
**resolveNamespace closure injected into every handler** — Namespace is not a URL path segment; a resolveNamespace func(ctx) (string, error) closure is constructed once in NewServer and injected into all handler constructors. (`resolveNamespace := func(ctx context.Context) (string, error) { ns, ok := config.NamespaceDecoder.GetNamespace(ctx); ... }`)
**Optional handlers guarded by nil checks** — Handlers for optional services (ChargeService, LLMCostService, CostService) are only constructed when the service is non-nil; the server field stays nil otherwise. Routes must nil-check or delegate to api.Unimplemented{}. (`if config.ChargeService != nil { chargesH = chargeshandler.New(...) }`)
**Content negotiation at route level** — QueryMeter inspects Accept header via commonhttp.GetMediaType; text/csv routes to the CSV handler, all other types use JSON handler. Content negotiation is done in routes.go, not in the handler. (`if mediaType, _ := commonhttp.GetMediaType(r); mediaType == "text/csv" { s.metersHandler.QueryMeterCSV()... }`)
**var _ api.ServerInterface = (*Server)(nil) compile-time check** — A blank-identifier compile-time assertion ensures Server implements the generated ServerInterface. Must be kept; removing it hides missing route implementations. (`var _ api.ServerInterface = (*Server)(nil)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `server.go` | Config struct definition, Config.Validate, Server struct with all handler fields, NewServer constructor, RegisterRoutes with OAS validation middleware. | Adding a new domain service requires: field in Config, nil-check in Validate (if required), handler construction in NewServer, handler field in Server struct. Missing any step causes a panic or silent noop. |
| `routes.go` | One method per generated ServerInterface operation; pure delegation to handler. Contains all credits feature-flag guards. | Credits endpoints must check both s.Credits.Enabled and handler != nil. Multi-param routes (e.g. ListCostBases) pack args into a local struct (e.g. currencieshandler.ListCostBasesArgs) before .With(...). |

## Anti-Patterns

- Adding business logic or error handling inside routes.go — it must be pure delegation
- Forgetting credits.Enabled guard on new credit-related routes (both route and constructor levels)
- Passing nil handler to ServeHTTP — always guard optional handlers with nil checks or delegate to api.Unimplemented{}
- Adding a service to Config without adding it to Config.Validate() — silently accepts broken wiring
- Removing the var _ api.ServerInterface compile-time assertion

## Decisions

- **credits.enabled enforced at both NewServer (noop wiring) and route dispatch (explicit guard)** — Credits is cross-cutting; a single guard point is insufficient. The route-level check is a safety net for cases where noop wiring is bypassed.
- **resolveNamespace is a closure, not a middleware** — Namespace decoding is not a URL path parameter in v3; injecting it as a closure into each handler keeps the transport layer clean without requiring Chi context injection.
- **Handler fields are unexported; Config services are exported** — Handlers are implementation details assembled in NewServer; the Config struct is the public dependency surface that callers (app/common) populate via Wire.

## Example: Adding a new domain handler to the v3 server

```
// 1. Add to Config in server.go:
FooService foo.Service

// 2. Add to Config.Validate():
if c.FooService == nil { errs = append(errs, errors.New("foo service is required")) }

// 3. Add handler field to Server struct:
fooHandler foohandler.Handler

// 4. In NewServer, construct and assign:
fooH := foohandler.New(resolveNamespace, config.FooService, httptransport.WithErrorHandler(config.ErrorHandler))
// ...
return &Server{ ..., fooHandler: fooH }, nil

// 5. In routes.go, implement the generated interface method:
// ...
```

<!-- archie:ai-end -->
