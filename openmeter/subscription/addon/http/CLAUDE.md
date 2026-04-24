# http

<!-- archie:ai-start -->

> HTTP handler layer for subscription addon CRUD, bridging the v1 API (package httpdriver) to subscriptionaddon.Service and subscriptionworkflow.Service. Follows the standard httptransport.HandlerWithArgs[Request, Response, Params] pattern used throughout openmeter.

## Patterns

**HandlerWithArgs three-stage pipeline** — Each operation uses httptransport.NewHandlerWithArgs(decoder, operation, encoder, ...options). The decoder resolves namespace + decodes body, the operation calls domain services, the encoder writes JSON with a fixed status code. (`httptransport.NewHandlerWithArgs(func(ctx, r, params) (Req, error) {...}, func(ctx, req) (Resp, error) {...}, commonhttp.JSONResponseEncoderWithStatus[Resp](http.StatusOK), opts...)`)
**Upsert-on-create convenience** — CreateSubscriptionAddon checks if the addon already exists on the subscription and routes to ChangeAddonQuantity instead of AddAddon to provide idempotent create semantics. (`if sAdd, ok := lo.Find(subsAdds.Items, ...); ok { view, add, err = h.WorkflowService.ChangeAddonQuantity(...) } else { view, add, err = h.WorkflowService.AddAddon(...) }`)
**Namespace resolved via NamespaceDecoder** — All decoders call h.resolveNamespace(ctx) which delegates to h.NamespaceDecoder.GetNamespace(ctx). Missing namespace returns HTTP 500. (`ns, err := h.resolveNamespace(ctx); if err != nil { return ..., err }`)
**Response requires SubscriptionView** — MapSubscriptionAddonToResponse needs both the SubscriptionAddon and its parent SubscriptionView to compute AffectedSubscriptionItemIds via addondiff.GetAffectedItemIDs. Get and List endpoints each call SubscriptionService.GetView. (`view, err := h.SubscriptionService.GetView(ctx, req.SubscriptionID); return MapSubscriptionAddonToResponse(view, *res)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Defines Handler interface, HandlerConfig (three services + NamespaceDecoder + Logger), NewHandler constructor, and resolveNamespace helper. | HandlerConfig embeds all three services (SubscriptionAddonService, SubscriptionWorkflowService, SubscriptionService) — all are required. |
| `mapping.go` | MapSubscriptionAddonToResponse builds api.SubscriptionAddon including AffectedSubscriptionItemIds. MapCreateSubscriptionAddonRequestToInput translates API body to workflow input. | Returns error if addon has no instances (empty union of periods). Calls addondiff.GetAffectedItemIDs from the diff sub-package. |
| `create.go` | Upsert-on-create: if addon already attached, calls ChangeAddonQuantity; otherwise AddAddon. Both paths return via MapSubscriptionAddonToResponse. | Returns HTTP 200 for both create and update paths — no 201 distinction. |
| `update.go` | PATCH-style quantity update. Requires both Timing and Quantity fields in body; returns 400 validation error otherwise. | Timing and Quantity nil-checks are manual in the decoder, not via model.Validate. |

## Anti-Patterns

- Adding business logic to decoder functions — decoders only parse and validate shape, not semantics
- Calling domain service methods directly without going through the Handler methods registered on the router
- Forgetting to fetch SubscriptionView when constructing the response — MapSubscriptionAddonToResponse panics without it
- Using httptransport.NewHandler instead of NewHandlerWithArgs when URL path params are needed

## Decisions

- **Upsert semantics on POST create** — API consumers often want idempotent addon attachment; routing to ChangeAddonQuantity when the addon is already present avoids a 409 conflict and simplifies client logic.

<!-- archie:ai-end -->
