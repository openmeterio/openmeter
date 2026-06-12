# driver

<!-- archie:ai-start -->

> HTTP transport layer (package creditdriver) for credit grants: builds httptransport handlers for ListGrants (v1, array-or-paginated union), VoidGrant, and ListGrantsV2. Maps credit/grant domain objects into entitlement-grant API shapes.

## Patterns

**Three-stage httptransport handler** — Every endpoint is httptransport.NewHandlerWithArgs[Req,Resp,Params] with a decode fn (resolve namespace, parse params into grant.ListParams), a business fn (call grantRepo/grantConnector), and a response encoder. (`httptransport.NewHandlerWithArgs[ListGrantsHandlerRequest, ListGrantsHandlerResponse, ListGrantsHandlerParams](decode, handle, commonhttp.JSONResponseEncoder, opts...)`)
**Namespace resolution gate** — Each decode fn calls h.resolveNamespace(ctx) (wrapping namespaceDecoder.GetNamespace) and returns 500 if absent before constructing the request. (`ns, err := h.resolveNamespace(ctx); if err != nil { return Req{}, err }`)
**Per-handler error encoder** — Handlers append httptransport.WithErrorEncoder to translate domain errors: models.IsGenericValidationError -> 400, *credit.GrantNotFoundError -> 404, *pagination.InvalidError -> 400. (`if _, ok := err.(*credit.GrantNotFoundError); ok { commonhttp.NewHTTPError(http.StatusNotFound, err).EncodeError(ctx, w); return true }`)
**Domain grant mapped through entitlement grant** — credit grants are converted via meteredentitlement.GrantFromCreditGrant then entitlement_httpdriver.MapEntitlementGrantToAPI (v1) / ...v2.MapEntitlementGrantToAPIV2 — there is no direct credit-grant API mapper. (`eg, _ := meteredentitlement.GrantFromCreditGrant(grant, clock.Now()); apiGrant := entitlement_httpdriver.MapEntitlementGrantToAPI(eg)`)
**Backward-compatible union response** — ListGrants returns commonhttp.Union[[]Grant, Result[Grant]]: when Page.IsZero() it emits a bare array (Option1), otherwise a paginated object (Option2). (`if request.params.Page.IsZero() { response.Option1 = &apiGrants } else { response.Option2 = &pagination.Result[...]{...} }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `grant.go` | GrantHandler interface + grantHandler with ListGrants/VoidGrant/ListGrantsV2; holds namespaceDecoder, grantConnector, grantRepo, customerService. | Listing goes through grantRepo.ListGrants but voiding goes through grantConnector.VoidGrant; OrderBy strings are CamelToSnake-converted and validated against grant.OrderBy().StrValues(); V2 resolves Customer IDs-or-keys via customerService and skips deleted customers. |

## Anti-Patterns

- Reading the namespace from the request body/path instead of namespaceDecoder.GetNamespace via resolveNamespace.
- Returning domain errors unmapped — every handler must register WithErrorEncoder for validation/not-found/pagination errors.
- Building API grant DTOs directly instead of routing through GrantFromCreditGrant + entitlement MapEntitlementGrantToAPI(V2).
- Dropping the v1 array-vs-paginated union behavior keyed on Page.IsZero() (breaks backward compatibility).

## Decisions

- **Credit grants are exposed under the entitlement-grant API shape, not their own.** — Grants are always owned by metered entitlements; the comment in code notes 'entitlement grants are all we have', avoiding a duplicate public surface.
- **ListGrants keeps a polymorphic array-or-paginated response.** — Older clients depend on the bare-array form when no pagination params are supplied.

<!-- archie:ai-end -->
