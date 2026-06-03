# plans

<!-- archie:ai-start -->

> v3 HTTP handler package for all plan CRUD and lifecycle operations (create, get, list, update, delete, publish, archive); bridges generated api/v3 types to plan.Service with rich bidirectional conversion of the Plan > Phase > RateCard > Price hierarchy. Its planaddons/ sub-package handles the plan-addons sub-resource.

## Patterns

**Type-alias triplet per operation file** — Each file declares <Op>Request/<Op>Response/<Op>Handler aliasing domain input/output directly. (`type ArchivePlanRequest = plan.ArchivePlanInput; type ArchivePlanResponse = api.BillingPlan; type ArchivePlanHandler httptransport.HandlerWithArgs[ArchivePlanRequest, ArchivePlanResponse, ArchivePlanParams]`)
**Namespace resolved first in decoder** — h.resolveNamespace(ctx) is the first decoder call; return immediately on error. (`ns, err := h.resolveNamespace(ctx); if err != nil { return ArchivePlanRequest{}, err }`)
**IgnoreNonCriticalIssues=true on create and update** — After FromAPICreatePlanRequest/FromAPIUpsertPlanRequest, set req.IgnoreNonCriticalIssues = true before returning from the decoder. (`req.IgnoreNonCriticalIssues = true; return req, nil`)
**clock.Now() for time-sensitive fields** — PublishPlan sets EffectiveFrom = clock.Now(); ArchivePlan sets EffectiveTo = clock.Now(). Never time.Now() — tests use clock.SetTime. (`EffectiveTo: clock.Now()`)
**Nil-check service response for mutating lifecycle ops** — Create/Update/Publish/Archive check p == nil after the service call and return a descriptive error; Get/Delete do not. (`if p == nil { return ArchivePlanResponse{}, fmt.Errorf("failed to archive plan") }; return ToAPIBillingPlan(*p)`)
**Exhaustive price-type switch with unsupported guard** — ToAPIBillingPrice switches on p.Type(); DynamicPriceType/PackagePriceType return models.NewGenericConflictError. ListPlans skips plans with unsupported prices via hasUnsupportedV3Price. (`case productcatalog.DynamicPriceType: return result, models.NewGenericConflictError(fmt.Errorf("dynamic price is not supported in v3 API"))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Handler interface (7 methods) and struct holding only resolveNamespace, plan.Service, and options. | No streaming/customer service is injected; do not add deps — delegate to plan.Service. |
| `convert.go` | All bidirectional Plan>Phase>RateCard>Price conversion plus unsupportedV3PriceTypes map and hasUnsupportedV3Price. | Dynamic/Package prices return GenericConflictError (409), not 500; a new price type requires updating ToAPIBillingPrice, FromAPIBillingPrice, and unsupportedV3PriceTypes together. |
| `convert_test.go` | Table-driven tests for convert.go; uses clock.SetTime, newTestPlan, decimal/gobl literals. | New conversion paths need round-trip tests using typed API constants, not plain strings. |
| `list.go` | ListPlans with page-based pagination; silently skips unsupported-v3-price plans (FIXME). | Do not add further silent-skip logic — return errors or expose all items. |
| `publish.go` | PublishPlan sets EffectiveFrom = clock.Now(). | Must use clock.Now(), not time.Now(). |
| `archive.go` | ArchivePlan sets EffectiveTo = clock.Now(). | Same clock.Now() requirement. |
| `planaddons/` | Sub-package for the plan-addons sub-resource bridging to planaddon.Service (injected separately from plan.Service). | planaddon.Service is a distinct injection from plan.Service. |

## Anti-Patterns

- Using time.Now() instead of clock.Now() in decoder closures
- Omitting req.IgnoreNonCriticalIssues = true in Create/Update decoders
- Adding a price type in ToAPIBillingPrice without updating FromAPIBillingPrice, hasUnsupportedV3Price, and round-trip tests
- Omitting apierrors.GenericErrorEncoder() from a handler options block
- Calling domain service methods from the encoder (third) closure instead of the operation (second) closure

## Decisions

- **Dynamic/Package price types return GenericConflictError (409) not 500** — The v3 API has no wire format for these yet; a conflict error surfaces the limitation clearly instead of dropping data or returning a generic 500.
- **IgnoreNonCriticalIssues set in the HTTP decoder, not the domain service** — HTTP create/update callers expect lenient validation while the service still surfaces non-critical issues as warnings on the returned object.

<!-- archie:ai-end -->
