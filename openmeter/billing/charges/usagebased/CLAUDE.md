# usagebased

<!-- archie:ai-start -->

> Domain-model and interface root for the usage-based charge type: defines the Charge aggregate (ChargeBase, Intent, State), RealizationRun history, DetailedLine, the Status state-machine vocabulary, and the Adapter/Handler/Service interfaces that the adapter/ and service/ children implement. This package owns the value types and contracts; it contains no DB or orchestration logic itself.

## Patterns

**Validate() aggregates errors via NewNillableGenericValidationError** — Every domain/input type has Validate() that collects into var errs []error, wraps each field with fmt.Errorf("field: %w", err), and returns models.NewNillableGenericValidationError(errors.Join(errs...)) — never early-return on first failure. (`func (c ChargeBase) Validate() error { var errs []error; if err := c.Intent.Validate(); err != nil { errs = append(errs, fmt.Errorf("intent: %w", err)) }; return models.NewNillableGenericValidationError(errors.Join(errs...)) }`)
**Normalized() canonicalizes timestamps before persist/compare** — Types carrying time fields expose Normalized() calling meta.NormalizeTimestamp / meta.NormalizeOptionalTimestamp (Intent.InvoiceAt, State.AdvanceAfter, RealizationRunBase StoredAtLT/ServicePeriodTo). Persist and compare normalized values, never raw. (`func (i Intent) Normalized() Intent { i.Intent = i.Intent.Normalized(); i.InvoiceAt = meta.NormalizeTimestamp(i.InvoiceAt); return i }`)
**Voided billing-history filtering** — RealizationRun.IsVoidedBillingHistory() is the single gate for excluding runs from future billing/balance math: true when DeletedAt != nil OR Type == RealizationRunTypeInvalidDueToUnsupportedCreditNote. Sum/Bisect/MapToBillingMeteredQuantity must skip these via WithoutVoidedBillingHistory(). (`func (r RealizationRuns) WithoutVoidedBillingHistory() RealizationRuns { return lo.Filter(r, func(run RealizationRun, _ int) bool { return !run.IsVoidedBillingHistory() }) }`)
**Cumulative-to-line quantity mapping** — RealizationRun.MeteredQuantity is cumulative from intent service-period start to ServicePeriodTo; convert to billing.StandardLine semantics only via RealizationRuns.MapToBillingMeteredQuantity, which derives PreLinePeriod from the latest non-voided prior run and errors if LinePeriod goes negative. (`linePeriod := currentRun.MeteredQuantity.Sub(preLinePeriod); if linePeriod.IsNegative() { return BillingMeteredQuantity{}, fmt.Errorf("line period metered quantity is negative...") }`)
**RateableIntent bridges charges into the billing rating engine** — Rating is performed through the billing.rating.StandardLineAccessor interface, implemented by RateableIntent. A charge is hardcoded as never progressively billed, zero previously-billed amount, and no user-defined line discounts — keep these invariants. (`var _ rating.StandardLineAccessor = (*RateableIntent)(nil); func (r RateableIntent) IsProgressivelyBilled() bool { return false }`)
**Status enum mirrors meta.ChargeStatus with dotted substates** — Status is a string enum of dotted substates (e.g. active.final_realization.issuing). Status.Validate() checks membership in Values(); ToMetaChargeStatus() coarsens by splitting on the first '.'. Mutability is classified by membership lists, not string parsing. (`func IsMutableInvoiceBackedRealizationStatus(status Status) bool { return slices.Contains(mutableInvoiceBackedRealizationStatuses, status) }`)
**Handler interface with UnimplementedHandler default** — Side-effecting credit/payment/invoice hooks are declared on the Handler interface; UnimplementedHandler provides not-implemented stubs and is conformance-asserted with var _ Handler = (*UnimplementedHandler)(nil). New hooks go on Handler plus the stub. (`var _ Handler = (*UnimplementedHandler)(nil); func (h UnimplementedHandler) OnInvoiceUsageAccrued(...) (ledgertransaction.GroupReference, error) { return ledgertransaction.GroupReference{}, errors.New("not implemented") }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `charge.go` | Charge/ChargeBase/Intent/State/Expands aggregate; GetFeatureKeyOrID returns a state-aware feature ref (key when StatusCreated, ID otherwise, ID-then-key fallback when StatusDeleted). | GetFeatureKeyOrID branches on Status — adding a status without updating the switch silently falls into the default (ID-only) branch. |
| `realizationrun.go` | RealizationRun lifecycle types, RealizationRunType, BillingMeteredQuantity, BisectByTimestamp, MapToBillingMeteredQuantity, voided-history helpers. | Create input rejects RealizationRunTypeInvalidDueToUnsupportedCreditNote; run service-period start is derived from the prior non-voided run's ServicePeriodTo, not a stored field. |
| `statemachine.go` | Status enum constants and the mutableFinalRealization / mutableInvoiceBackedRealization membership lists used to decide where period changes are allowed. | Issuing/Completed states are intentionally absent from the mutable lists — adding them would let billing mutate immutable invoice/ledger records. |
| `adapter.go` | Adapter interface = composition of Charge/RealizationRun/CreditAllocation/InvoiceUsage/Payment sub-adapters plus entutils.TxCreator; implemented by adapter/. | The sub-interface split is load-bearing; add new persistence methods to the matching sub-interface, not a flat one. |
| `handler.go` | Handler hooks (OnInvoiceUsageAccrued, OnPaymentAuthorized/Settled, OnCreditsOnlyUsageAccrued[+Correction]) and their input structs with Validate / ValidateWith(currencyx.Calculator). | Correction inputs need ValidateWith(currencyCalculator) for amount-matching checks; plain Validate() is insufficient. |
| `rating.go` | RateableIntent implements billing.rating.StandardLineAccessor to feed a charge into the shared rating engine. | Discount CorrelationIDs are placeholders ('usagebased-ratecard-*') because charges do not use splitlinegroups; do not rely on them downstream. |
| `detailedline.go / detailedline_uniqueref.go` | DetailedLine over stddetailedline.Base; NewDetailedLinesFromBilling maps rating output; service-period suffix encode/decode for ChildUniqueReferenceID ('id@[from..to]'). | Validate uses stddetailedline.IgnoreQuantityChecks(); PricerReferenceID must be non-empty; strip the '@[...]' suffix before re-keying. |
| `ratingengine.go / errors.go` | RatingEngine enum (delta vs period_preserving) controlling correction semantics; ValidationIssue constants (negative total, credit-allocation mismatch, active-run-exists). | RatingEngine drives whether prior periods are re-snapshotted; errors are models.ValidationIssue with critical severity + 400 status, not plain errors. |

## Anti-Patterns

- Returning on the first validation failure instead of collecting into errs and returning models.NewNillableGenericValidationError(errors.Join(errs...)).
- Counting deleted or unsupported-credit-note runs in billing/balance math instead of filtering through IsVoidedBillingHistory / WithoutVoidedBillingHistory.
- Comparing or persisting timestamps without meta.NormalizeTimestamp / Normalized(), causing UTC/precision drift against stored rows.
- Adding an issuing/completed Status to the mutable membership lists, letting period changes touch immutable invoice or ledger records.
- Deriving billing.StandardLine quantity straight from RealizationRun.MeteredQuantity (cumulative) instead of MapToBillingMeteredQuantity, producing wrong line totals.

## Decisions

- **This package holds only domain types and interfaces (Adapter, Handler, Service); DB logic lives in adapter/ and orchestration/state machines in service/.** — Keeps the type/contract layer importable by both children and tests without pulling in Ent or orchestration dependencies, and lets the state machine and persistence evolve independently.
- **Voided runs (deleted or unsupported-credit-note) are retained but excluded from future billing via IsVoidedBillingHistory rather than physically removed.** — Unsupported-credit-note invoice lines cannot be deleted without prorating/credit-note support, so runs are kept for audit while being ignored by rating and balance calculations.
- **MeteredQuantity is stored as a cumulative value and converted to per-line quantity at read time.** — Standard invoice lines need pre-line + line-period quantities; storing cumulative lets later runs reconstruct prior billed quantity from the latest non-voided prior run.

## Example: Mapping a cumulative run quantity into billing line quantity, skipping voided history

```
func (r RealizationRuns) MapToBillingMeteredQuantity(currentRun RealizationRun) (BillingMeteredQuantity, error) {
    preLinePeriod := alpacadecimal.Zero
    var latestPriorRun *RealizationRun
    for idx := range r {
        if r[idx].IsVoidedBillingHistory() { continue }
        if !r[idx].ServicePeriodTo.Before(currentRun.ServicePeriodTo) { continue }
        if latestPriorRun == nil || r[idx].ServicePeriodTo.After(latestPriorRun.ServicePeriodTo) {
            latestPriorRun = &r[idx]
        }
    }
    if latestPriorRun != nil { preLinePeriod = latestPriorRun.MeteredQuantity }
    linePeriod := currentRun.MeteredQuantity.Sub(preLinePeriod)
    if linePeriod.IsNegative() { return BillingMeteredQuantity{}, fmt.Errorf("line period metered quantity is negative") }
    return BillingMeteredQuantity{PreLinePeriod: preLinePeriod, LinePeriod: linePeriod}, nil
}
```

<!-- archie:ai-end -->
