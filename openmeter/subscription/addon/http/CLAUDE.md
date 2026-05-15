# http

<!-- archie:ai-start -->

> HTTP handler layer for subscription addon CRUD, bridging the v1 API to subscriptionaddon.Service and subscriptionworkflow.Service using the standard httptransport.HandlerWithArgs[Request,Response,Params] pipeline.

## Patterns

**HandlerWithArgs three-stage pipeline** — Each operation uses httptransport.NewHandlerWithArgs(decoder, operation, encoder, ...options). The decoder resolves namespace and decodes body, the operation calls domain services, the encoder writes JSON with a fixed status code. (`httptransport.NewHandlerWithArgs(
	func(ctx context.Context, r *http.Request, params P) (Req, error) { ... },
	func(ctx context.Context, req Req) (Resp, error) { ... },
	commonhttp.JSONResponseEncoderWithStatus[Resp](http.StatusOK),
	httptransport.AppendOptions(h.Options, httptransport.WithOperationName("op"))...,
)`)
**Upsert-on-create convenience** — CreateSubscriptionAddon checks if the addon is already attached to the subscription and routes to ChangeAddonQuantity instead of AddAddon, providing idempotent create semantics. Both paths return HTTP 200. (`if sAdd, ok := lo.Find(subsAdds.Items, func(s subscriptionaddon.SubscriptionAddon) bool {
	return s.Addon.ID == req.AddonInput.AddonID
}); ok {
	view, add, err = h.WorkflowService.ChangeAddonQuantity(ctx, ...)
} else {
	view, add, err = h.WorkflowService.AddAddon(ctx, ...)
}`)
**Namespace resolved via NamespaceDecoder** — All decoders call h.resolveNamespace(ctx) which delegates to h.NamespaceDecoder.GetNamespace(ctx). Missing namespace returns HTTP 500 via commonhttp.NewHTTPError. (`ns, err := h.resolveNamespace(ctx)
if err != nil { return ..., err }`)
**Response always requires SubscriptionView** — MapSubscriptionAddonToResponse needs both the SubscriptionAddon and its parent SubscriptionView to compute AffectedSubscriptionItemIds via addondiff.GetAffectedItemIDs. Every Get/List endpoint fetches the view separately. (`view, err := h.SubscriptionService.GetView(ctx, req.SubscriptionID)
if err != nil { return ..., err }
return MapSubscriptionAddonToResponse(view, *res)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Defines Handler interface, HandlerConfig (three services + NamespaceDecoder + Logger), NewHandler constructor, and resolveNamespace helper. | HandlerConfig embeds all three services (SubscriptionAddonService, SubscriptionWorkflowService, SubscriptionService) — all are required; missing any causes panics at call time. |
| `mapping.go` | MapSubscriptionAddonToResponse builds api.SubscriptionAddon including AffectedSubscriptionItemIds. MapCreateSubscriptionAddonRequestToInput translates API body to workflow input. | Returns error if addon has no instances (empty union of periods). Calls addondiff.GetAffectedItemIDs which needs the full SubscriptionView. |
| `create.go` | Upsert-on-create: if addon already attached, calls ChangeAddonQuantity; otherwise AddAddon. Both paths return HTTP 200. | No 201 distinction between create and update paths — intentional for idempotency. |
| `update.go` | PATCH-style quantity update. Requires both Timing and Quantity fields; returns 400 validation error otherwise. | Timing and Quantity nil-checks are manual in the decoder, not via model.Validate — keep them in sync with the API schema. |

## Anti-Patterns

- Adding business logic to decoder functions — decoders only parse and validate shape, not semantics
- Forgetting to fetch SubscriptionView before calling MapSubscriptionAddonToResponse — the function requires it to compute AffectedSubscriptionItemIds
- Using httptransport.NewHandler instead of NewHandlerWithArgs when URL path params (SubscriptionID, SubscriptionAddonID) are needed
- Returning nil SubscriptionView to MapSubscriptionAddonToResponse — it will error on instance union computation

## Decisions

- **Upsert semantics on POST create** — API consumers often want idempotent addon attachment; routing to ChangeAddonQuantity when the addon is already present avoids a 409 conflict and simplifies client logic.

## Example: Standard handler wiring with URL params and namespace resolution

```
func (h *handler) GetSubscriptionAddon() GetSubscriptionAddonHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params GetSubscriptionAddonParams) (GetSubscriptionAddonRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil { return GetSubscriptionAddonRequest{}, err }
			return GetSubscriptionAddonRequest{
				SubscriptionID:      models.NamespacedID{Namespace: ns, ID: params.SubscriptionID},
				SubscriptionAddonID: models.NamespacedID{Namespace: ns, ID: params.SubscriptionAddonID},
			}, nil
		},
		func(ctx context.Context, req GetSubscriptionAddonRequest) (GetSubscriptionAddonResponse, error) {
			res, err := h.SubscriptionAddonService.Get(ctx, req.SubscriptionAddonID)
			if err != nil { return GetSubscriptionAddonResponse{}, err }
			view, err := h.SubscriptionService.GetView(ctx, req.SubscriptionID)
			if err != nil { return GetSubscriptionAddonResponse{}, err }
// ...
```

<!-- archie:ai-end -->
