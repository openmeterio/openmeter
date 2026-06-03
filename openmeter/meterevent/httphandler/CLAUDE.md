# httphandler

<!-- archie:ai-start -->

> HTTP handler layer for the meterevent domain, adapting v1 (ListEvents) and v2 (ListEventsV2) REST endpoints to meterevent.Service calls via the generic httptransport.HandlerWithArgs pipeline. Mounted by openmeter/server/router.

## Patterns

**HandlerWithArgs triple: decoder / operation / encoder** — Every endpoint is built with httptransport.NewHandlerWithArgs(decoder, operation, commonhttp.JSONResponseEncoderWithStatus[Response](http.StatusOK), options...); the decoder maps params only, the operation calls the service. (`return httptransport.NewHandlerWithArgs(func(ctx, r, params) (Req, error) {...}, func(ctx, req) (Resp, error) {...}, commonhttp.JSONResponseEncoderWithStatus[ListEventsResponse](http.StatusOK), opts...)`)
**Type-alias block per endpoint file** — Each handler file opens with a type block aliasing Params/Response to api.* and Request to meterevent.* params, plus the Handler alias. (`type ( ListEventsParams = api.ListEventsParams; ListEventsResponse = []api.IngestedEvent; ListEventsHandler httptransport.HandlerWithArgs[...] ); type ListEventsRequest = meterevent.ListEventsParams`)
**Namespace resolved via resolveNamespace in decoder** — The decoder calls h.resolveNamespace(ctx) first; namespace is never a query param and a missing namespace returns http.StatusInternalServerError (server misconfiguration, not 400). (`ns, err := h.resolveNamespace(ctx); if err != nil { return ListEventsRequest{}, err }`)
**Validation errors wrapped in decoder, not operation** — Param-conversion errors (e.g. from convertListEventsV2Params) are wrapped in models.NewGenericValidationError in the decoder before returning. (`if err != nil { return ListEventsV2Request{}, models.NewGenericValidationError(err) }`)
**WithOperationName appended to handler options** — Every handler appends httptransport.WithOperationName("<camelCaseName>") via httptransport.AppendOptions for tracing/metrics labeling. (`httptransport.AppendOptions(h.options, httptransport.WithOperationName("listEventsV2"))...`)
**Mapping functions in mapping.go, not inline** — All API↔domain conversions (convertEvent, convertListEventsV2Params, convertListEventsV2Response) live in mapping.go; handler files call them with no inline conversion. (`result[i], err = convertEvent(event)`)
**Handler interface embeds fine-grained sub-interfaces** — The exported Handler interface embeds EventHandler (and future groups); new endpoint groups become new embedded interfaces, not direct methods on Handler. (`type Handler interface { EventHandler }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Defines Handler/EventHandler interfaces, the private handler struct, New() constructor, and resolveNamespace helper. | resolveNamespace returns http.StatusInternalServerError for a missing namespace — this is correct (server misconfiguration); do not change to 400. |
| `event.go` | ListEvents handler (v1); applies MaximumFromDuration and MaximumLimit defaults in the decoder. | minimumFrom uses MaximumFromDuration to avoid edge-case validation failures — preserve when changing time window logic. |
| `event_v2.go` | ListEventsV2 handler (v2); delegates param conversion to convertListEventsV2Params in mapping.go. | StoredAt filter is not forwarded through convertListEventsV2Params — adding it requires changes in both mapping.go and the domain params struct. |
| `mapping.go` | All API↔domain conversions: convertEvent, convertListEventsV2Params, convertListEventsV2Response. | ValidationErrors are joined with errors.Join into api.IngestedEvent.ValidationError as *string (nil when none) — never returned as top-level errors. |

## Anti-Patterns

- Calling meterevent.Service methods in the decoder — decoder maps params only; service calls belong in the operation.
- Inlining type conversion logic in handler files instead of mapping.go.
- Omitting WithOperationName from handler options — breaks tracing and metrics labeling.
- Adding business logic (e.g. time window enforcement) in the operation instead of the decoder.
- Returning a non-200 status from JSONResponseEncoderWithStatus for successful responses — errors go through the error encoder chain.

## Decisions

- **v1 and v2 list handlers live in separate files (event.go vs event_v2.go)** — v1 returns a flat slice with a hard default limit; v2 uses cursor pagination and richer filters — separate files avoid branching inside one handler and keep versioned behavior auditable.
- **mapping.go centralizes all API↔domain conversions** — Keeps handler files focused on the decode→operate→encode lifecycle and makes conversions independently testable without reading handler code.

## Example: Adding a new v2-style list endpoint to this handler package

```
// subject.go
package httphandler

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/meterevent"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	ListXParams   = api.ListXParams
// ...
```

<!-- archie:ai-end -->
