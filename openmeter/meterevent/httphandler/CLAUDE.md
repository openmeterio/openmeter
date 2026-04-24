# httphandler

<!-- archie:ai-start -->

> HTTP handler layer for the meterevent domain, adapting v1 (ListEvents) and v2 (ListEventsV2) REST endpoints to meterevent.Service calls. Uses the generic httptransport.HandlerWithArgs pattern and is mounted by openmeter/server/router.

## Patterns

**HandlerWithArgs triple: decoder / operation / encoder** — Every endpoint is created with httptransport.NewHandlerWithArgs(decoderFn, operationFn, commonhttp.JSONResponseEncoderWithStatus[Response](http.StatusOK), options...). (`return httptransport.NewHandlerWithArgs(func(ctx, r, params) (Req, error) {...}, func(ctx, req) (Resp, error) {...}, commonhttp.JSONResponseEncoderWithStatus[ListEventsResponse](http.StatusOK), opts...)`)
**Type alias block per endpoint** — Each handler file declares a type block aliasing Params, Request, Response, and Handler types before the method. Params and Response alias api.* types; Request aliases meterevent.* params. (`type (
	ListEventsParams   = api.ListEventsParams
	ListEventsResponse = []api.IngestedEvent
	ListEventsHandler  httptransport.HandlerWithArgs[ListEventsRequest, ListEventsResponse, ListEventsParams]
)
type ListEventsRequest = meterevent.ListEventsParams`)
**Namespace resolved via namespaceDecoder in decoder fn** — The decoder function always calls h.resolveNamespace(ctx) first. Never pass namespace as a query param from outside. (`ns, err := h.resolveNamespace(ctx); if err != nil { return ListEventsRequest{}, err }`)
**Validation errors wrapped with models.NewGenericValidationError in decoder** — If convertListEventsV2Params returns an error, wrap it in models.NewGenericValidationError before returning from the decoder — not in the operation. (`if err != nil { return ListEventsV2Request{}, models.NewGenericValidationError(err) }`)
**OperationName option appended to handler options** — Each handler appends httptransport.WithOperationName("<camelCaseName>") to h.options via httptransport.AppendOptions. (`httptransport.AppendOptions(h.options, httptransport.WithOperationName("listEventsV2"))...`)
**Mapping functions in mapping.go, not inline** — All API↔domain type conversions (convertEvent, convertListEventsV2Params, convertListEventsV2Response) live in mapping.go. Handler files call them but do not inline mapping logic. (`result[i], err = convertEvent(event)`)
**Interface composition for Handler** — The exported Handler interface embeds fine-grained sub-interfaces (EventHandler). Add new groups as separate embedded interfaces, not as direct methods on Handler. (`type Handler interface { EventHandler }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Defines Handler/EventHandler interfaces, the private handler struct, New() constructor, and resolveNamespace helper. | resolveNamespace returns http.StatusInternalServerError on missing namespace — never return 400 here; missing namespace is a server misconfiguration. |
| `event.go` | ListEvents handler (v1). Applies MaximumFromDuration default floor and MaximumLimit default. | minimumFrom adds one second to avoid edge-case validation failures — preserve this when changing time window logic. |
| `event_v2.go` | ListEventsV2 handler (v2). Delegates param conversion to convertListEventsV2Params in mapping.go. | The StoredAt filter is NOT forwarded through ListEventsV2 from event_v2.go (missing in convertListEventsV2Params). Adding it requires both mapping.go and the domain params struct. |
| `mapping.go` | All API↔domain conversions. convertEvent marshals meterevent.Event to api.IngestedEvent including ValidationErrors join. convertListEventsV2Response builds the cursor-paginated response. | ValidationErrors are joined with errors.Join and placed in api.IngestedEvent.ValidationError as *string — nil when no errors. |

## Anti-Patterns

- Calling meterevent.Service methods directly in the decoder function — decoder maps params only; the operation function calls the service.
- Inlining type conversion logic in handler files instead of mapping.go.
- Returning a non-200 status from JSONResponseEncoderWithStatus for successful responses (errors go through the error encoder chain).
- Adding business logic (e.g. time window enforcement) in the operation function instead of the decoder.
- Omitting WithOperationName from handler options — breaks tracing and metrics labeling.

## Decisions

- **v1 and v2 list handlers are in separate files (event.go vs event_v2.go).** — v1 uses a simple slice response with a hard default limit; v2 uses cursor pagination and richer filters. Keeping them separate avoids branching inside a single handler.
- **mapping.go centralizes all API↔domain conversions.** — Keeps handler files focused on the request lifecycle (decode→operate→encode) and makes conversion logic independently testable and auditable.

## Example: Adding a new v2-style list endpoint to this handler

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
