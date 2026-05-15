# httpdriver

<!-- archie:ai-start -->

> HTTP handler layer for the info domain, exposing stateless endpoints (e.g. currency list) via the httptransport pattern. No service dependency — data comes from external library calls (gobl/currency) or static definitions, so no adapter or Wire provider is needed.

## Patterns

**httptransport.NewHandler triple** — Every endpoint is a dedicated named handler type (e.g. ListCurrenciesHandler) defined as httptransport.Handler[Req, Resp] and constructed via httptransport.NewHandler with a decoder func, an operation func, and a response encoder. (`type ListCurrenciesHandler httptransport.Handler[ListCurrenciesRequest, ListCurrenciesResponse]
return httptransport.NewHandler(decoderFn, operationFn, commonhttp.JSONResponseEncoderWithStatus[...](http.StatusOK), opts...)`)
**Handler interface + private handler struct** — driver.go defines a public Handler interface listing all endpoint methods, and a private handler struct carrying shared options. New handlers are added as methods on *handler and listed in the Handler interface. (`type Handler interface { ListCurrencies() ListCurrenciesHandler }
type handler struct { options []httptransport.HandlerOption }
func New(options ...httptransport.HandlerOption) Handler { return &handler{options: options} }`)
**Options forwarded via httptransport.AppendOptions** — Handler-level options (e.g. WithOperationName) are appended to h.options using httptransport.AppendOptions so cross-cutting concerns (tracing, error handling) from the caller propagate correctly. (`httptransport.AppendOptions(h.options, httptransport.WithOperationName("listCurrencies"))...`)
**No-op request decoder for parameterless endpoints** — When an endpoint has no path/query parameters, the decoder simply returns an empty request struct and nil error. (`func(ctx context.Context, r *http.Request) (ListCurrenciesRequest, error) { return ListCurrenciesRequest{}, nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `driver.go` | Defines the Handler interface and New constructor — the only public API of this package. Add new handler method signatures here when adding endpoints. | Do not add business logic here; driver.go is pure wiring. |
| `currencies.go` | Implements ListCurrencies using gobl/currency.Definitions() filtered to ISO currencies. Representative template for all future handlers in this package. | The ISONumeric != '' filter intentionally excludes crypto/non-ISO currencies — do not remove it without understanding the API contract. |

## Anti-Patterns

- Injecting a domain service or adapter into handler struct when data comes from a static library — keep stateless handlers dependency-free
- Returning raw error from operation func without using domain error types — breaks the generic error encoder chain
- Defining handler types outside this package or in driver.go — each handler belongs in its own file
- Hand-editing generated API types in api/ instead of regenerating via make gen-api
- Adding request validation logic inside the decoder instead of returning a models.GenericValidationError from the operation func

## Decisions

- **No domain service dependency — data sourced directly from gobl/currency library** — Currency definitions are static ISO data; introducing a service/adapter layer would add unnecessary indirection and wiring cost.
- **httptransport.Handler[Req, Resp] generic over plain http.HandlerFunc** — Keeps decoder, operation, and encoder as separate typed functions, enabling independent testing of each phase and reuse of shared error encoders.

## Example: Adding a new stateless info endpoint (e.g. ListTimezones)

```
// timezones.go
package httpdriver

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	ListTimezonesRequest  struct{}
	ListTimezonesResponse []api.Timezone
// ...
```

<!-- archie:ai-end -->
