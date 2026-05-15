# subscriptionaddons

<!-- archie:ai-start -->

> v3 HTTP handler for listing subscription addons; implements the httptransport.HandlerWithArgs pipeline for the ListSubscriptionAddons endpoint, converting between domain subscriptionaddon.SubscriptionAddon and API apiv3.SubscriptionAddon types.

## Patterns

**HandlerWithArgs decode/operate/encode pipeline** — Every endpoint uses httptransport.NewHandlerWithArgs with three functions: (1) decode params+request into a typed request struct, (2) call the domain service, (3) encode response. Never write raw http.Handler. (`httptransport.NewHandlerWithArgs(decodeFn, operateFn, commonhttp.JSONResponseEncoderWithStatus[T](http.StatusOK), opts...)`)
**Namespace resolution via injected closure** — Namespace is resolved by calling h.resolveNamespace(ctx) inside the decode function. Never hardcode or parse namespace from URL params directly. (`ns, err := h.resolveNamespace(ctx)`)
**Pagination via pagination.NewPage with validation** — Page input is always constructed via pagination.NewPage(number, size) with defaults (1, 20), then validated with page.Validate() before building the service input. (`page := pagination.NewPage(lo.FromPtrOr(params.Params.Page.Number, 1), lo.FromPtrOr(params.Params.Page.Size, 20))`)
**Error encoding via apierrors.GenericErrorEncoder** — All handlers append apierrors.GenericErrorEncoder() as the error encoder option. Never use commonhttp.GenericErrorEncoder() in v3 handlers. (`httptransport.AppendOptions(h.options, httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()))`)
**Sort parsing via request.ParseSortBy** — Sort query params are parsed with request.ParseSortBy, then mapped to domain OrderBy type and validated with OrderBy.Validate(). Bad sort fields return apierrors.NewBadRequestError. (`sort, err := request.ParseSortBy(*params.Params.Sort); input.OrderBy = subscriptionaddon.OrderBy(sort.Field)`)
**toAPI conversion in convert.go only** — All domain-to-API type conversion lives in convert.go as standalone functions (e.g. toAPISubscriptionAddon). The operate function calls these; it never embeds conversion logic inline. (`converted, err := toAPISubscriptionAddon(item)`)
**Handler interface + struct separation** — handler.go defines a Handler interface listing all endpoint methods (e.g. ListSubscriptionAddons()), a private handler struct holding dependencies, and a New() constructor. Each endpoint lives in its own file. (`type Handler interface { ListSubscriptionAddons() ListSubscriptionAddonsHandler }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Defines Handler interface, handler struct with injected deps (resolveNamespace, subscriptionAddonService, options), and New() constructor. | Adding new dependencies here requires wiring in api/v3/server/server.go. Keep handler struct fields minimal. |
| `list.go` | Full implementation of ListSubscriptionAddons endpoint: typed params, request, response, and handler function. | Default pagination values (page 1, size 20) must be applied before Validate(). Sort fields must be validated via OrderBy.Validate() — unknown fields should return 400 not 500. |
| `convert.go` | Domain-to-API type conversion. toAPISubscriptionAddon reads clock.Now() to determine the active instance and computes the union period across all instances. | Returns error if addon has no instances — callers must handle this. The union period calculation uses lo.Reduce; adding new fields here must stay in sync with apiv3.SubscriptionAddon struct. |

## Anti-Patterns

- Never call subscriptionAddonService methods directly from the decode function — decode must only parse/validate inputs.
- Never use commonhttp.GenericErrorEncoder() in v3 handlers; always use apierrors.GenericErrorEncoder().
- Never embed domain-to-API conversion logic inline in list.go — put it in convert.go.
- Never skip page.Validate() before using the page value — invalid page params must return 400.
- Never resolve namespace from URL path params directly — always call h.resolveNamespace(ctx).

## Decisions

- **toAPISubscriptionAddon reads clock.Now() and unions all instance periods to produce a single ActiveFrom/ActiveTo range** — The domain model stores multiple timed instances; the API surface exposes a single flattened addon view with a current quantity snapshot and a full effective date range.
- **Sort validation uses subscriptionaddon.OrderBy.Validate() after field mapping** — Keeps supported sort fields authoritative in the domain package — the handler does not hardcode valid sort field names.

## Example: Adding a new list endpoint to this package

```
// In handler.go — add to Handler interface:
GetSubscriptionAddon() GetSubscriptionAddonHandler

// New file get.go:
type GetSubscriptionAddonHandler = httptransport.HandlerWithArgs[GetRequest, apiv3.SubscriptionAddon, GetParams]

func (h *handler) GetSubscriptionAddon() GetSubscriptionAddonHandler {
    return httptransport.NewHandlerWithArgs(
        func(ctx context.Context, r *http.Request, params GetParams) (GetRequest, error) {
            ns, err := h.resolveNamespace(ctx)
            if err != nil { return GetRequest{}, err }
            return GetRequest{ID: models.NamespacedID{Namespace: ns, ID: params.AddonID}}, nil
        },
        func(ctx context.Context, req GetRequest) (apiv3.SubscriptionAddon, error) {
            item, err := h.subscriptionAddonService.Get(ctx, req.ID)
// ...
```

<!-- archie:ai-end -->
