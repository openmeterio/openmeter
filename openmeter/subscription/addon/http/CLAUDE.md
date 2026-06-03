# http

<!-- archie:ai-start -->

> HTTP handler layer for subscription addon CRUD, bridging the v1 API to subscriptionaddon.Service and subscriptionworkflow.Service via the standard httptransport.HandlerWithArgs[Request,Response,Params] pipeline.

## Patterns

**HandlerWithArgs three-stage pipeline** — Each operation uses httptransport.NewHandlerWithArgs(decoder, operation, encoder, ...options): decoder resolves namespace and decodes body, operation calls domain services, encoder writes JSON with a fixed status. (`httptransport.NewHandlerWithArgs(decoderFn, operationFn, commonhttp.JSONResponseEncoderWithStatus[Resp](http.StatusOK), httptransport.AppendOptions(h.Options, httptransport.WithOperationName("op"))...)`)
**Upsert-on-create convenience** — CreateSubscriptionAddon checks if the addon is already attached and routes to ChangeAddonQuantity instead of AddAddon, giving idempotent create semantics; both paths return HTTP 200. (`if sAdd, ok := lo.Find(subsAdds.Items, func(s subscriptionaddon.SubscriptionAddon) bool { return s.Addon.ID == req.AddonInput.AddonID }); ok { ... ChangeAddonQuantity } else { ... AddAddon }`)
**Namespace resolved via NamespaceDecoder** — All decoders call h.resolveNamespace(ctx), delegating to h.NamespaceDecoder.GetNamespace(ctx); a missing namespace returns HTTP 500 via commonhttp.NewHTTPError. (`ns, err := h.resolveNamespace(ctx); if err != nil { return ..., err }`)
**Response always requires SubscriptionView** — MapSubscriptionAddonToResponse needs both the SubscriptionAddon and its parent SubscriptionView to compute AffectedSubscriptionItemIds via addondiff.GetAffectedItemIDs; every Get/List endpoint fetches the view separately. (`view, err := h.SubscriptionService.GetView(ctx, req.SubscriptionID); return MapSubscriptionAddonToResponse(view, *res)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Defines the Handler interface, HandlerConfig (three services + NamespaceDecoder + Logger), NewHandler, and resolveNamespace. | HandlerConfig embeds all three services (SubscriptionAddonService, SubscriptionWorkflowService, SubscriptionService) — all required; missing any panics at call time. |
| `mapping.go` | MapSubscriptionAddonToResponse builds api.SubscriptionAddon including AffectedSubscriptionItemIds; MapCreateSubscriptionAddonRequestToInput translates API body to workflow input. | Errors if the addon has no instances (empty union of periods); calls addondiff.GetAffectedItemIDs which needs the full SubscriptionView. |
| `create.go` | Upsert-on-create: if attached, ChangeAddonQuantity; else AddAddon. Both paths return HTTP 200. | No 201/200 distinction between create and update — intentional for idempotency. |
| `update.go` | PATCH-style quantity update; requires both Timing and Quantity, else a 400 validation error. | Timing/Quantity nil-checks are manual in the decoder, not via model.Validate — keep them in sync with the API schema. |

## Anti-Patterns

- Adding business logic to decoder functions — decoders only parse and validate shape
- Forgetting to fetch SubscriptionView before MapSubscriptionAddonToResponse
- Using NewHandler instead of NewHandlerWithArgs when URL path params are needed
- Passing a nil SubscriptionView to MapSubscriptionAddonToResponse

## Decisions

- **Upsert semantics on POST create** — Clients often want idempotent addon attachment; routing to ChangeAddonQuantity when already present avoids a 409 conflict and simplifies client logic.

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
