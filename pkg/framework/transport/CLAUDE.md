# transport

<!-- archie:ai-start -->

> Namespace-only organisational folder owning the HTTP transport layer for all domain httpdriver and api/v3/handlers packages. Its sole child (httptransport) provides the single generic Handler[Request,Response] decode→operation→encode pipeline that every v1 and v3 endpoint in the monorepo is built on — no source files live here directly.

## Patterns

**Child-delegated architecture** — All source code lives in pkg/framework/transport/httptransport; this folder is a namespace wrapper only. New transport sub-packages (e.g. grpctransport) would sit here as siblings. (`import "github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `httptransport/handler.go` | Defines Handler[Request,Response] value type and NewHandler triple constructor (RequestDecoder, Operation, ResponseEncoder). Contains the last-resort 500 fallback ErrorEncoder that must not be replicated elsewhere. | Do not instantiate handler{} struct directly — always use NewHandler; it attaches defaultHandlerOptions including GenericErrorEncoder. |
| `httptransport/argshandler.go` | HandlerWithArgs variant for path-param injection; value type cloned by With()/Chain() so adding pointer fields would race. | Never add mutable state (mutex, cache) to HandlerWithArgs — value semantics mean each With()/Chain() call produces a shallow copy. |
| `httptransport/options.go` | HandlerOption and AppendOptions helpers; controls ErrorEncoder chain construction. | ErrorEncoder chain is first-match-wins; ordering of options matters. |
| `httptransport/encoder/encoder.go` | ResponseEncoder and ErrorEncoder function type definitions shared across all handler packages. | SelfEncodingError escape hatch is for infrastructure types only — do not implement it on domain errors. |

## Anti-Patterns

- Placing source .go files directly in pkg/framework/transport/ — all code belongs in named sub-packages
- Adding a second child package that duplicates the decode/operate/encode pipeline instead of reusing httptransport.Handler
- Importing pkg/framework/transport directly (no Go files here); always import the named sub-package
- Calling context.Background() inside a RequestDecoder — always use the ctx passed from ServeHTTP
- Writing to http.ResponseWriter inside a RequestDecoder on error — return the error and let the ErrorEncoder chain handle it

## Decisions

- **Single child package rather than a flat file set** — Keeps the transport abstraction extensible (future grpc/websocket siblings) while letting all current consumers import a stable sub-package path (httptransport) without churn.
- **Value-receiver handler with copy-on-With/Chain semantics** — Prevents accidental shared state across middleware chains; each Chain() call returns a new handler copy, making the pipeline composable without races.
- **Last-resort 500 fallback lives in handler.go, not in GenericErrorEncoder** — Keeps GenericErrorEncoder focused on typed domain-error matching; the transport layer owns the final safety net so domain packages never need to handle unrecognised error types themselves.

<!-- archie:ai-end -->
