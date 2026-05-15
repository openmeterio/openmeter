# httphandler

<!-- archie:ai-start -->

> HTTP handler layer for the meterevent domain, adapting v1 (ListEvents) and v2 (ListEventsV2) REST endpoints to meterevent.Service calls via the generic httptransport.HandlerWithArgs pipeline. Mounted by openmeter/server/router.

## Patterns

**HandlerWithArgs triple: decoder / operation / encoder** — Every endpoint is constructed with httptransport.NewHandlerWithArgs(decoderFn, operationFn, commonhttp.JSONResponseEncoderWithStatus[Response](http.StatusOK), options...). The decoder maps params only; the operation calls the service. (`return httptransport.NewHandlerWithArgs(func(ctx, r, params) (Req, error) {...}, func(ctx, req) (Resp, error) {...}, commonhttp.JSONResponseEncoderWithStatus[ListEventsResponse](http.StatusOK), opts...)`)
**Type alias block per endpoint file** — Each handler file opens with a type block aliasing Params, Request, Response, and Handler types before the method. Params/Response alias api.* types; Request aliases meterevent.* params. (`type (
	ListEventsParams   = api.ListEventsParams
	ListEventsResponse = []api.IngestedEvent
	ListEventsHandler  httptransport.HandlerWithArgs[ListEventsRequest, ListEventsResponse, ListEventsParams]
)
type ListEventsRequest = meterevent.ListEventsParams`)
**Namespace resolved via namespaceDecoder in decoder fn** — The decoder function always calls h.resolveNamespace(ctx) first. Namespace is never passed as a query param; missing namespace returns http.StatusInternalServerError (server misconfiguration, not 400). (`ns, err := h.resolveNamespace(ctx); if err != nil { return ListEventsRequest{}, err }`)
**Validation errors wrapped in decoder, not operation** — Param conversion errors (e.g. from convertListEventsV2Params) are wrapped in models.NewGenericValidationError in the decoder function before returning. (`if err != nil { return ListEventsV2Request{}, models.NewGenericValidationError(err) }`)
**WithOperationName appended to handler options** — Every handler appends httptransport.WithOperationName("<camelCaseName>") via httptransport.AppendOptions(h.options, ...) — required for tracing and metrics labeling. (`httptransport.AppendOptions(h.options, httptransport.WithOperationName("listEventsV2"))...`)
**Mapping functions in mapping.go, not inline** — All API↔domain type conversions (convertEvent, convertListEventsV2Params, convertListEventsV2Response) live in mapping.go. Handler files call them but contain no inline conversion logic. (`result[i], err = convertEvent(event)`)
**Handler interface embeds fine-grained sub-interfaces** — The exported Handler interface embeds EventHandler (and future groups). New endpoint groups become new embedded interfaces, not direct methods on Handler. (`type Handler interface { EventHandler }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Defines Handler/EventHandler interfaces, private handler struct, New() constructor, and resolveNamespace helper. | resolveNamespace returns http.StatusInternalServerError for missing namespace — this is correct (server misconfiguration). Do not change to 400. |
| `event.go` | ListEvents handler (v1). Applies MaximumFromDuration default and MaximumLimit default in the decoder. | minimumFrom adds one second via MaximumFromDuration to avoid edge-case validation failures — preserve this when changing time window logic. |
| `event_v2.go` | ListEventsV2 handler (v2). Delegates param conversion to convertListEventsV2Params in mapping.go. | StoredAt filter is not forwarded through convertListEventsV2Params. Adding it requires changes in both mapping.go and the domain params struct. |
| `mapping.go` | All API↔domain type conversions: convertEvent, convertListEventsV2Params, convertListEventsV2Response. | ValidationErrors are joined with errors.Join and placed in api.IngestedEvent.ValidationError as *string — nil when no errors. Do not return them as top-level errors. |

## Anti-Patterns

- Calling meterevent.Service methods in the decoder function — decoder maps params only; service calls belong in the operation function.
- Inlining type conversion logic in handler files instead of mapping.go.
- Omitting WithOperationName from handler options — breaks tracing and metrics labeling.
- Adding business logic (e.g. time window enforcement) in the operation function instead of the decoder.
- Returning a non-200 status from JSONResponseEncoderWithStatus for successful responses — errors go through the error encoder chain.

## Decisions

- **v1 and v2 list handlers are in separate files (event.go vs event_v2.go).** — v1 returns a flat slice with a hard default limit; v2 uses cursor pagination and richer filters. Separate files avoid branching inside a single handler and make versioned behavior auditable.
- **mapping.go centralizes all API↔domain conversions.** — Keeps handler files focused on the decode→operate→encode lifecycle and makes conversion logic independently testable and auditable without reading handler code.

## Example: Adding a new v2-style list endpoint to this handler package

```
// In a new file, e.g. subject.go:
package httphandler

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/meterevent"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
// ...
```

<!-- archie:ai-end -->
