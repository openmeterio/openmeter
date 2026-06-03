# subscriptionaddons

<!-- archie:ai-start -->

> v3 HTTP handlers for subscription addons (list and get); implements the httptransport.HandlerWithArgs pipeline converting between domain subscriptionaddon.SubscriptionAddon and apiv3.SubscriptionAddon types.

## Patterns

**HandlerWithArgs decode/operate/encode pipeline** — Every endpoint uses httptransport.NewHandlerWithArgs with three functions: decode params+request, call the domain service, encode response. Never write a raw http.Handler. (`httptransport.NewHandlerWithArgs(decodeFn, operateFn, commonhttp.JSONResponseEncoderWithStatus[T](http.StatusOK), opts...)`)
**Namespace resolution via injected closure** — Namespace is resolved by calling h.resolveNamespace(ctx) inside the decode function — never parsed from URL params directly. (`ns, err := h.resolveNamespace(ctx)`)
**Pagination via pagination.NewPage with validation** — Page input is built via pagination.NewPage(number, size) with defaults (1, 20), then validated with page.Validate() before building the service input. (`page := pagination.NewPage(lo.FromPtrOr(params.Params.Page.Number, 1), lo.FromPtrOr(params.Params.Page.Size, 20))`)
**Error encoding via apierrors.GenericErrorEncoder** — List appends apierrors.GenericErrorEncoder() as the error encoder. Never use commonhttp.GenericErrorEncoder() in v3 handlers. (`httptransport.AppendOptions(h.options, httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()))`)
**Sort parsing via request.ParseSortBy + OrderBy.Validate** — Sort params parse with request.ParseSortBy, map to subscriptionaddon.OrderBy, set Order via sort.Order.ToSortxOrder(), then validate with OrderBy.Validate(); bad fields return 400. (`input.OrderBy = subscriptionaddon.OrderBy(sort.Field); if err := input.OrderBy.Validate(); err != nil { return ..., apierrors.NewBadRequestError(...) }`)
**Conversion in convert.go only** — All domain-to-API conversion is in convert.go as standalone functions (toAPISubscriptionAddon). Operate functions call them; conversion is never embedded in list.go/get.go. (`converted, err := toAPISubscriptionAddon(item)`)
**Handler interface + struct + per-file endpoints** — handler.go defines the Handler interface (ListSubscriptionAddons, GetSubscriptionAddon), a private handler struct (resolveNamespace, addonService, options), and New(). Each endpoint lives in its own file. (`type Handler interface { ListSubscriptionAddons() ListSubscriptionAddonsHandler; GetSubscriptionAddon() GetSubscriptionAddonHandler }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Handler interface, handler struct (resolveNamespace, addonService, options), New() constructor. | Adding dependencies requires wiring in api/v3/server/server.go. Keep struct fields minimal. |
| `list.go` | ListSubscriptionAddons: typed params, request, response; pagination + sort validation; per-item toAPISubscriptionAddon mapping. | Apply default page (1, 20) before Validate(). Unknown sort fields must return 400 via OrderBy.Validate(), not 500. |
| `get.go` | GetSubscriptionAddon: builds GetSubscriptionAddonInput (NamespacedID + SubscriptionID); returns 404 when the service result is nil. | Note get.go does NOT append apierrors.GenericErrorEncoder in its options block (only WithOperationName) — list.go does. |
| `convert.go` | toAPISubscriptionAddon: reads clock.Now() to pick the active instance, unions all instance periods into one ActiveFrom/ActiveTo. | Returns NewGenericNotFoundError if no instance is active at now or no instances exist — callers must handle it. Union uses lo.Reduce; new fields must stay in sync with apiv3.SubscriptionAddon. |

## Anti-Patterns

- Calling addonService methods from the decode function — decode only parses/validates inputs.
- Using commonhttp.GenericErrorEncoder() in v3 handlers; always use apierrors.GenericErrorEncoder().
- Embedding domain-to-API conversion inline in list.go/get.go — put it in convert.go.
- Skipping page.Validate() before using the page value.
- Resolving namespace from URL path params directly instead of h.resolveNamespace(ctx).

## Decisions

- **toAPISubscriptionAddon reads clock.Now() and unions all instance periods into a single ActiveFrom/ActiveTo with a current quantity snapshot.** — The domain stores multiple timed instances; the API exposes a single flattened addon view with a full effective range.
- **Sort validation uses subscriptionaddon.OrderBy.Validate() after field mapping.** — Keeps supported sort fields authoritative in the domain package; the handler does not hardcode valid field names.

## Example: Converting a domain addon with multiple instances to the flattened API view

```
func toAPISubscriptionAddon(addon subscriptionaddon.SubscriptionAddon) (apiv3.SubscriptionAddon, error) {
  now := clock.Now()
  inst, found := addon.GetInstanceAt(now)
  if !found { return apiv3.SubscriptionAddon{}, models.NewGenericNotFoundError(fmt.Errorf("no instance is active at %s", now.Format(time.RFC3339))) }
  pers := lo.Map(addon.GetInstances(), func(i subscriptionaddon.SubscriptionAddonInstance, _ int) timeutil.OpenPeriod { return i.AsPeriod() })
  union := lo.Reduce(pers, func(agg, item timeutil.OpenPeriod, _ int) timeutil.OpenPeriod { return agg.Union(item) }, pers[0])
  return apiv3.SubscriptionAddon{Id: addon.ID, Quantity: inst.Quantity, QuantityAt: now, ActiveFrom: lo.FromPtrOr(union.From, now), ActiveTo: union.To}, nil
}
```

<!-- archie:ai-end -->
