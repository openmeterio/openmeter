# charges

<!-- archie:ai-start -->

> v3 HTTP handler for listing a customer's billing charges (flat-fee and usage-based). Primary constraint: credit-purchase charges are intentionally excluded at the query layer and must never appear on this API surface.

## Patterns

**HandlerWithArgs with inline pagination validation** — ListCustomerCharges decoder builds a pagination.Page (default 1,20) from query params and calls page.Validate(), returning apierrors.NewBadRequestError on failure. (`page := pagination.NewPage(1, 20); if err := page.Validate(); err != nil { return ..., apierrors.NewBadRequestError(ctx, err, ...) }`)
**Credit-purchase exclusion via ChargeTypes filter** — The request always sets ChargeTypes to {ChargeTypeFlatFee, ChargeTypeUsageBased}; convertChargeToAPI returns an explicit error if ChargeTypeCreditPurchase ever reaches it as defense in depth. (`req := ListCustomerChargesRequest{..., ChargeTypes: []meta.ChargeType{meta.ChargeTypeFlatFee, meta.ChargeTypeUsageBased}}`)
**Realizations always expanded for booked totals** — expands always includes meta.ExpandRealizations (booked totals need realization-run data); RealtimeUsage is added only when the caller requests it. (`expands := meta.Expands{meta.ExpandRealizations}; if slices.Contains(*args.Params.Expand, api.BillingChargesExpandRealTimeUsage) { expands = expands.With(meta.ExpandRealtimeUsage) }`)
**Sort field whitelist validation** — validChargesSortField only accepts id, created_at, service_period.from, billing_period.from; anything else returns a 400 listing the allowed values. (`if !validChargesSortField(sort.Field) { return ..., apierrors.NewBadRequestError(ctx, fmt.Errorf("unsupported sort field: %s"), ...) }`)
**Union-type dispatch in convertChargeToAPI** — Switch on charge.Type(), call AsFlatFeeCharge/AsUsageBasedCharge, map to the API struct, then out.FromBillingFlatFeeCharge/FromBillingUsageBasedCharge — both steps' errors must be checked. (`if err := out.FromBillingFlatFeeCharge(apiFF); err != nil { return out, fmt.Errorf("setting flat fee charge union: %w", err) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Defines Handler interface and struct; New() takes resolveNamespace + billingcharges.ChargeService (no full billing.Service dependency). | Methods needing billing.Service require extending the struct and constructor. |
| `list.go` | Only handler: decodes pagination, sort, status filter, expand; calls service.ListCharges; maps via convertChargeToAPI. | Status filter parsing uses a switch in parseChargeStatusFilterSlice — add new meta.ChargeStatus variants there. |
| `convert.go` | All domain-to-API mappers; many exported (ConvertClosedPeriodToAPI, ConvertCurrencyCodeToAPI, ConvertChargeStatusToAPI...) and reused by sibling packages like customerscredits. | Exported converters are shared — signature changes break siblings. DynamicPrice/PackagePrice are unsupported in toAPIBillingPrice and return an error. |

## Anti-Patterns

- Including meta.ChargeTypeCreditPurchase in the ChargeTypes filter (credit purchases belong to the credits API)
- Omitting meta.ExpandRealizations from expands (breaks booked totals)
- Using a generic string sort field without validating against validChargesSortField
- Calling out.FromBillingFlatFeeCharge without checking the returned error

## Decisions

- **Credit-purchase charges are excluded here and served by the credits API** — Credit purchases have a distinct lifecycle (settlement, ledger funding) belonging to the credits domain; mixing them would conflate two billing concepts.
- **Realization runs are always fetched even when not requested** — Booked totals are computed from persisted realization runs; returning a charge without them yields incorrect zero totals.

## Example: Convert a domain Charge to the API union type with error handling

```
func convertChargeToAPI(charge billingcharges.Charge) (api.BillingCharge, error) {
    var out api.BillingCharge
    switch charge.Type() {
    case meta.ChargeTypeFlatFee:
        ff, err := charge.AsFlatFeeCharge()
        if err != nil { return out, fmt.Errorf("converting flat fee charge: %w", err) }
        apiFF, err := convertFlatFeeChargeToAPI(ff)
        if err != nil { return out, err }
        if err := out.FromBillingFlatFeeCharge(apiFF); err != nil { return out, fmt.Errorf("setting flat fee charge union: %w", err) }
    case meta.ChargeTypeCreditPurchase:
        return out, fmt.Errorf("credit purchase charges are not supported in the charges API")
    default:
        return out, fmt.Errorf("unsupported charge type: %s", charge.Type())
    }
    return out, nil
// ...
```

<!-- archie:ai-end -->
