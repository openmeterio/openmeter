# http

<!-- archie:ai-start -->

> HTTP driver (package httpdriver) for the subscription-addon REST surface: create, list, get, and update (quantity change) of subscription addons. Thin transport layer that decodes requests, delegates to the subscription-addon Service and subscription Workflow Service, and maps domain objects to api.SubscriptionAddon.

## Patterns

**httptransport.HandlerWithArgs triple** — Each endpoint defines Params/Request/Response/Handler type aliases and a method returning httptransport.NewHandlerWithArgs(decode, handle, encode, opts...). Decode resolves namespace and decodes body; encode uses commonhttp.JSONResponseEncoderWithStatus. (`CreateSubscriptionAddonHandler = httptransport.HandlerWithArgs[CreateSubscriptionAddonRequest, CreateSubscriptionAddonResponse, CreateSubscriptionAddonParams]`)
**Namespace resolution via resolveNamespace** — Every decode step calls h.resolveNamespace(ctx) (wraps NamespaceDecoder.GetNamespace) and builds models.NamespacedID from path params; missing namespace returns a 500 HTTPError. (`ns, err := h.resolveNamespace(ctx); SubscriptionID: models.NamespacedID{Namespace: ns, ID: params.SubscriptionID}`)
**Workflow service for mutations, addon service for reads** — Create/Update route through h.SubscriptionWorkflowService (AddAddon / ChangeAddonQuantity); Get/List read via h.SubscriptionAddonService and fetch the SubscriptionView via h.SubscriptionService.GetView for response mapping. (`view, add, err = h.SubscriptionWorkflowService.AddAddon(ctx, req.SubscriptionID, req.AddonInput)`)
**Create-or-change upsert convenience** — CreateSubscriptionAddon lists existing addons; if the AddonID already exists it calls ChangeAddonQuantity instead of AddAddon, so POST is idempotent-by-addon. (`if sAdd, ok := lo.Find(subsAdds.Items, ...); ok { ChangeAddonQuantity(...) } else { AddAddon(...) }`)
**Response mapping requires both view and addon** — MapSubscriptionAddonToResponse(view, addon) computes the union period of instances, current quantity at clock.Now(), affected item IDs via addondiff.GetAffectedItemIDs, and maps rate cards via productcataloghttp.FromRateCard. (`return MapSubscriptionAddonToResponse(view, add)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Handler interface, HandlerConfig (SubscriptionAddonService, SubscriptionWorkflowService, SubscriptionService, NamespaceDecoder, Logger), NewHandler, resolveNamespace | New endpoints must be added to the Handler interface and wired in router |
| `create.go` | createSubscriptionAddon — decodes SubscriptionAddonCreate, upserts via workflow service | Returns 200 (not 201); contains the find-existing-then-ChangeQuantity branch |
| `update.go` | updateSubscriptionAddon — ChangeAddonQuantity; requires Timing and Quantity in body | Explicitly errors if body.Timing or body.Quantity is nil before mapping |
| `get.go / list.go` | Read paths; both fetch SubscriptionView via SubscriptionService.GetView to enrich the response | List uses slicesx.MapWithErr; both depend on the view being available for affected-item mapping |
| `mapping.go` | MapCreateSubscriptionAddonRequestToInput and MapSubscriptionAddonToResponse | Errors with 'no instances found' if addon has zero instances; timing mapped via subscriptionhttp.MapAPITimingToTiming; depends on addondiff.GetAffectedItemIDs |

## Anti-Patterns

- Putting business validation or persistence in handlers — delegate to Service/WorkflowService
- Building NamespacedID without resolveNamespace (drops tenant scoping)
- Mapping a response without the SubscriptionView (affected item IDs and union period need it)
- Adding an endpoint method without registering it on the Handler interface

## Decisions

- **POST create doubles as quantity-change when addon already present** — Convenience so clients need not check existence first; keeps a single 'add this addon' verb
- **Mutations go through the workflow service, not the addon service directly** — AddAddon/ChangeAddonQuantity must also sync the subscription view/spec, which is the workflow layer's responsibility

<!-- archie:ai-end -->
