# httpdriver

<!-- archie:ai-start -->

> v1 HTTP handler layer for plan lifecycle endpoints, translating between api.* types and plan.Service calls via httptransport.Handler. Primary constraint: never call Ent or adapters directly — all persistence goes through plan.Service.

## Patterns

**httptransport.NewHandler / NewHandlerWithArgs** — Each endpoint method returns a typed handler alias (e.g. ListPlansHandler = httptransport.HandlerWithArgs[Req,Resp,Params]). The handler is built with a decoder closure, an operation closure delegating to h.service, a response encoder, and AppendOptions for operationName + ValidationErrorEncoder. (`return httptransport.NewHandlerWithArgs(decoderFn, operationFn, commonhttp.JSONResponseEncoderWithStatus[Resp](http.StatusOK), httptransport.AppendOptions(h.options, httptransport.WithOperationName("listPlans"), httptransport.WithErrorEncoder(productcataloghttp.ValidationErrorEncoder(productcataloghttp.ResourceKindPlan)))...)`)
**Namespace decoded from context** — h.resolveNamespace(ctx) is called in every decoder; failure returns commonhttp.NewHTTPError(500) since the namespace must always be set by middleware. (`ns, err := h.resolveNamespace(ctx); if err != nil { return Req{}, fmt.Errorf("failed to resolve namespace: %w", err) }`)
**credits.Enabled guard in decoder** — CreditOnlySettlementMode is rejected in CreatePlan and UpdatePlan decoders when h.credits.Enabled is false, producing models.NewGenericValidationError. (`if !h.credits.Enabled && req.SettlementMode == productcatalog.CreditOnlySettlementMode { return CreatePlanRequest{}, models.NewGenericValidationError(...) }`)
**Type alias pattern for request/response/handler** — Each endpoint declares three type aliases (XxxRequest, XxxResponse, XxxHandler) at package level so the interface method signature is self-documenting. (`type (ListPlansRequest = plan.ListPlansInput; ListPlansResponse = api.PlanPaginatedResponse; ListPlansHandler httptransport.HandlerWithArgs[...])`)
**ref.ParseIDOrKey for dual ID/Key routing** — GetPlan and NextPlan use ref.ParseIDOrKey to distinguish ULID IDs from string keys without separate endpoints. (`idOrKey := ref.ParseIDOrKey(params.IDOrKey); return GetPlanRequest{NamespacedID: models.NamespacedID{Namespace: ns, ID: idOrKey.ID}, Key: idOrKey.Key, ...}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `driver.go` | Handler/PlanHandler interface declaration, handler struct, New constructor. Credits config injected here for settlement mode guards. | credits appconfig.CreditsConfiguration must be passed to New; omitting it silently allows CreditOnly plans when credits are off. |
| `plan.go` | All endpoint implementations (ListPlans, CreatePlan, UpdatePlan, DeletePlan, GetPlan, PublishPlan, ArchivePlan, NextPlan). | IgnoreNonCriticalIssues=true is set in Create and Update decoders — do not remove, plans may have non-critical validation issues. |
| `mapping.go` | FromPlan, FromPlanPhase, AsCreatePlanRequest, AsUpdatePlanRequest, AsPlanPhase — api↔domain conversion. Calls http.FromRateCard/AsRateCards for rate card mapping. | New plan fields must be mapped in both FromPlan and AsCreatePlanRequest/AsUpdatePlanRequest or the API silently ignores them. |

## Anti-Patterns

- Importing entdb or calling plan.Repository directly from httpdriver — all access must go through plan.Service.
- Hardcoding namespace strings instead of calling h.resolveNamespace(ctx).
- Omitting httptransport.WithErrorEncoder(productcataloghttp.ValidationErrorEncoder(...)) — plan validation errors won't map to proper HTTP status codes.
- Adding credits guard logic outside the decoder closure — guards belong in request decoding, not in the operation.

## Decisions

- **Credits feature guard duplicated in both CreatePlan and UpdatePlan decoders.** — credits.enabled must be enforced at the HTTP boundary; service layer does not know about deployment config.
- **EffectivePeriod is zeroed in UpdatePlan service call; Publish/Archive are the only paths that change it.** — Prevents callers from setting plan status directly via update; status transitions are gated through explicit PublishPlan/ArchivePlan operations.

<!-- archie:ai-end -->
