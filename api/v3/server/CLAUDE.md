# server

<!-- archie:ai-start -->

> The sole assembly point for the v3 API surface: validates Config, constructs all domain handler structs, registers OAS request-validation middleware on a Chi router, and delegates every generated ServerInterface method to a typed handler. Credits feature flag is enforced both at constructor time (noop wiring) and at route dispatch.

## Patterns

**Config.Validate() before NewServer** — Config.Validate() runs at the top of NewServer and returns a wrapped error listing all missing required services. Never skip it. (`if err := config.Validate(); err != nil { return nil, fmt.Errorf("invalid v3 server config: %w", err) }`)
**credits.enabled gated at route AND constructor level** — Credits routes in routes.go check both s.Credits.Enabled and handler != nil before dispatching; in NewServer, disabled credits replace real services with noops. Both guards are required. (`if !s.Credits.Enabled || s.customersCreditsHandler == nil { unimplemented.GetCustomerCreditBalance(w, r, ...); return }`)
**Handler delegation via .With(params).ServeHTTP(w, r)** — Each ServerInterface method calls the typed handler method, optionally passes path/query params via .With(...), and calls ServeHTTP. No business logic in routes.go. (`s.metersHandler.GetMeter().With(meterId).ServeHTTP(w, r)`)
**resolveNamespace closure injected into every handler** — Namespace is not a URL path segment; a resolveNamespace func(ctx) (string, error) closure is built once in NewServer and injected into all handler constructors. (`resolveNamespace := func(ctx context.Context) (string, error) { ns, ok := config.NamespaceDecoder.GetNamespace(ctx); ... }`)
**Optional handlers guarded by nil checks** — Handlers for optional services (ChargeService, LLMCostService, CostService) are constructed only when the service is non-nil; the field stays nil otherwise. Routes nil-check or delegate to api.Unimplemented{}. (`var chargesH chargeshandler.Handler
if config.ChargeService != nil { chargesH = chargeshandler.New(...) }`)
**Content negotiation at route level** — QueryMeter inspects the Accept header via commonhttp.GetMediaType; text/csv routes to the CSV handler, all others to JSON. Negotiation happens in routes.go, not in the handler. (`if mediaType, _ := commonhttp.GetMediaType(r); mediaType == "text/csv" { s.metersHandler.QueryMeterCSV()... }`)
**var _ api.ServerInterface compile-time assertion** — A blank-identifier assertion ensures Server implements the generated ServerInterface; removing it hides missing route implementations. (`var _ api.ServerInterface = (*Server)(nil)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `server.go` | Config struct + Config.Validate, Server struct with all handler fields, NewServer constructor, RegisterRoutes with OAS validation middleware. | Adding a domain service requires: field in Config, nil-check in Validate (if required), handler construction in NewServer, handler field in Server struct. Missing any step causes a panic or silent noop. |
| `routes.go` | One method per generated ServerInterface operation; pure delegation to handlers; contains all credits feature-flag guards. | Credits endpoints must check both s.Credits.Enabled and handler != nil. Multi-param routes pack args into a local struct before .With(...). Never add business logic here. |

## Anti-Patterns

- Adding business logic or error handling inside routes.go — it must be pure delegation
- Forgetting credits.Enabled guard on new credit-related routes (both route and constructor levels required)
- Passing a nil handler to ServeHTTP — always guard optional handlers with nil checks or delegate to api.Unimplemented{}
- Adding a service to Config without adding it to Config.Validate() — silently accepts broken wiring
- Removing the var _ api.ServerInterface compile-time assertion

## Decisions

- **credits.enabled enforced at both NewServer (noop wiring) and route dispatch (explicit guard)** — Credits is cross-cutting; a single guard point is insufficient. The route-level check is a safety net for when noop wiring is bypassed.
- **resolveNamespace is a closure injected into handlers, not middleware** — Namespace decoding is not a URL path parameter in v3; injecting a closure keeps the transport layer clean without Chi context injection.
- **Handler fields are unexported; Config services are exported** — Handlers are implementation details assembled in NewServer; Config is the public dependency surface that app/common populates via Wire.

## Example: Adding a new domain handler to the v3 server

```
// 1. server.go Config: FooService foo.Service
// 2. Config.Validate(): if c.FooService == nil { errs = append(errs, errors.New("foo service is required")) }
// 3. Server struct: fooHandler foohandler.Handler
// 4. NewServer: fooH := foohandler.New(resolveNamespace, config.FooService, httptransport.WithErrorHandler(config.ErrorHandler)); return &Server{ ..., fooHandler: fooH }, nil
// 5. routes.go: func (s *Server) ListFoos(w http.ResponseWriter, r *http.Request, params api.ListFoosParams) { s.fooHandler.ListFoos().With(params).ServeHTTP(w, r) }
```

<!-- archie:ai-end -->
