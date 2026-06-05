# httpdriver

<!-- archie:ai-start -->

> HTTP transport layer for the progressmanager domain: wraps progressmanager.Service in httptransport handlers, decoding the namespace from context and mapping entity.Progress to api.Progress. Read-only (GetProgress) endpoint surface.

## Patterns

**Nested Handler interface + handler struct** — Handler embeds ProgressHandler which exposes GetProgress() GetProgressHandler; the unexported handler{service, namespaceDecoder, options} implements it, asserted via `var _ Handler = (*handler)(nil)`. (`type ProgressHandler interface { GetProgress() GetProgressHandler }`)
**httptransport.NewHandlerWithArgs three-func shape** — Each endpoint = (decode req from *http.Request + path arg) + (logic calling h.service) + JSONResponseEncoderWithStatus, with WithOperationName appended via httptransport.AppendOptions(h.options, ...). (`httptransport.NewHandlerWithArgs(decodeFn, logicFn, commonhttp.JSONResponseEncoderWithStatus[GetProgressResponse](http.StatusOK), httptransport.AppendOptions(h.options, httptransport.WithOperationName("getProgress"))...)`)
**Namespace from decoder, not request body** — resolveNamespace pulls the namespace via namespaceDecoder.GetNamespace(ctx); a missing namespace is a 500 commonhttp.NewHTTPError, never trusted from the client. (`ns, ok := h.namespaceDecoder.GetNamespace(ctx); if !ok { return "", commonhttp.NewHTTPError(http.StatusInternalServerError, errors.New("internal server error")) }`)
**Type-alias request/response/handler triplet** — Requests reuse entity types (GetProgressRequest = entity.GetProgressInput), responses reuse api types (GetProgressResponse = api.Progress), handler aliases httptransport.HandlerWithArgs[...]. Domain->API mapping via local progressToAPI. (`type GetProgressResponse = api.Progress`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Handler/ProgressHandler interfaces, handler struct, New(namespaceDecoder, service, options...), resolveNamespace helper. | Missing namespace returns 500 (internal), not 400 — it's expected to be set by upstream middleware. |
| `progress.go` | GetProgress() handler implementation and progressToAPI mapper. | service.GetProgress returning a nil *Progress with no error is treated as an internal error (fmt.Errorf), not 404 — 404 must come from the adapter's NotFound. progressToAPI drops namespace/ID (api.Progress only carries counters + UpdatedAt). |

## Anti-Patterns

- Reading the namespace from the request body/query instead of namespaceDecoder.
- Hand-rolling http.ResponseWriter encoding instead of commonhttp JSON encoders.
- Omitting WithOperationName, which OpenAPI/operation routing relies on.
- Mapping entity->API outside a dedicated *ToAPI function.

## Decisions

- **Only GetProgress is exposed over HTTP; UpsertProgress is internal.** — Progress is written by background streaming/clickhouse jobs, not clients; the API is read-only polling of operation status.

## Example: Map a domain Progress to the API type

```
import (
  "github.com/openmeterio/openmeter/api"
  progressmanagerentity "github.com/openmeterio/openmeter/openmeter/progressmanager/entity"
)

func progressToAPI(p progressmanagerentity.Progress) api.Progress {
  return api.Progress{Success: p.Success, Failed: p.Failed, Total: p.Total, UpdatedAt: p.UpdatedAt}
}
```

<!-- archie:ai-end -->
