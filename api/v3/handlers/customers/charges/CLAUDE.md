# charges

<!-- archie:ai-start -->

> HTTP handler for listing a customer's billing charges (flat-fee and usage-based) in the v3 API. Primary constraint: credit-purchase charges are intentionally excluded at the query layer and must never appear in this API surface.

## Patterns

**HandlerWithArgs with inline pagination validation** — ListCustomerCharges decoder builds a pagination.Page from query params with explicit page.Validate() that returns apierrors.NewBadRequestError on invalid page. Default page is (1, 20). (`page := pagination.NewPage(1, 20)
if args.Params.Page != nil { page = pagination.NewPage(lo.FromPtrOr(args.Params.Page.Number, 1), lo.FromPtrOr(args.Params.Page.Size, 20)) }
if err := page.Validate(); err != nil { return ..., apierrors.NewBadRequestError(ctx, err, ...) }`)
**Credit-purchase exclusion via ChargeTypes filter** — The request always sets ChargeTypes: []meta.ChargeType{meta.ChargeTypeFlatFee, meta.ChargeTypeUsageBased}. convertChargeToAPI returns an explicit error if ChargeTypeCreditPurchase ever reaches it as a defensive measure. (`req := ListCustomerChargesRequest{..., ChargeTypes: []meta.ChargeType{meta.ChargeTypeFlatFee, meta.ChargeTypeUsageBased}}`)
**Realizations always expanded for booked totals** — expands always includes meta.ExpandRealizations because booked totals require realization run data. RealtimeUsage is only added when the caller requests BillingChargesExpandRealTimeUsage. (`expands := meta.Expands{meta.ExpandRealizations}
if slices.Contains(*args.Params.Expand, api.BillingChargesExpandRealTimeUsage) { expands = expands.With(meta.ExpandRealtimeUsage) }`)
**Sort field whitelist validation** — Acceptable sort fields are "id", "created_at", "service_period.from", "billing_period.from". validChargesSortField rejects anything else with an explicit 400 listing allowed values. (`if !validChargesSortField(sort.Field) { return ..., apierrors.NewBadRequestError(ctx, fmt.Errorf("unsupported sort field: %s"), ...) }`)
**Union-type dispatch in convertChargeToAPI** — convertChargeToAPI switches on charge.Type(), calls AsFlatFeeCharge or AsUsageBasedCharge, maps to the API type, then calls out.FromBillingFlatFeeCharge or out.FromBillingUsageBasedCharge to set the discriminated union. Both the domain→struct and struct→union steps must succeed. (`if err := out.FromBillingFlatFeeCharge(apiFF); err != nil { return out, fmt.Errorf("setting flat fee charge union: %w", err) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Defines Handler interface and handler struct. Constructor New() takes resolveNamespace + billingcharges.ChargeService. No billing.Service dependency — only the charges sub-service. | Adding handler methods that need billing.Service will require extending the struct and constructor. |
| `list.go` | Only handler in this package. Decodes pagination, sort, status filter, and expand from query params; calls service.ListCharges; maps results via convertChargeToAPI. | Status filter parsing uses a type-safe switch in parseChargeStatusFilterSlice — add new status values there when meta.ChargeStatus gains new variants. |
| `convert.go` | All domain→API mapping functions. Many are exported (ConvertClosedPeriodToAPI, ConvertCurrencyCodeToAPI, etc.) and reused by other handler packages in the customers subtree. | Exported converters are shared; changing their signatures affects callers in sibling packages (e.g., customerscredits). |

## Anti-Patterns

- Including meta.ChargeTypeCreditPurchase in the ChargeTypes filter (credit purchases belong to the credits API)
- Omitting meta.ExpandRealizations from expands (breaks booked totals computation)
- Using a generic string sort field without validating against the whitelist in validChargesSortField
- Hand-writing the union set call (out.FromBillingFlatFeeCharge) without checking the error it returns

## Decisions

- **Credit-purchase charges are excluded from this endpoint and served by the credits API** — Credit purchases have a distinct lifecycle (settlement, ledger funding) that belongs to the credits domain; mixing them into the charges list would conflate two different billing concepts.
- **Realization runs are always fetched even when not requested by the caller** — Booked totals are computed from persisted realization runs; returning a charge without them would produce zero totals, which is incorrect for active charges.

<!-- archie:ai-end -->
