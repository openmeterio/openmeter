# targetstate

<!-- archie:ai-start -->

> Computes the desired (target) billing state for a subscription as of a point in time: iterates each subscription phase to project the invoice lines/charges that SHOULD exist, then exposes each as a StateItem the reconciler compares against persistedstate. Primary constraint: everything is aligned-billing-period aware and truncated to meter (1s / MinimumWindowSizeDuration) resolution.

## Patterns

**Builder.Build orchestrates, PhaseIterator generates** — Builder.Build validates BuildInput, sorts phases by ActiveFrom, calls collectUpcomingLines (which constructs a NewPhaseIterator per phase and calls Generate up to a per-phase generationLimit), then corrects period starts and maps results to StateItem. New target-state logic hangs off Builder; per-phase line projection hangs off PhaseIterator. (`iterator, err := NewPhaseIterator(b.logger, b.tracer, subs, phase.SubscriptionPhase.Key)`)
**Deterministic UniqueID composition** — Every generated item's UniqueID is strings.Join of subscriptionID/phaseKey/itemKey/v[version]/period[index] with '/'. This MUST exactly match the key persistedstate exposes via ChildUniqueReferenceID, or the reconciler treats it as create+delete instead of update. correctPeriodStartForUpcomingLines reconstructs the previous period's key the same way. (`UniqueID: strings.Join([]string{it.sub.Subscription.ID, it.phase.Spec.PhaseKey, item.Spec.ItemKey, fmt.Sprintf("v[%d]", version), fmt.Sprintf("period[%d]", periodIdx)}, "/")`)
**Truncate to meter resolution everywhere** — iterationEnd, ServicePeriod, FullServicePeriod, and BillingPeriod are all Truncate(streaming.MinimumWindowSizeDuration). truncateItemsIfNeeded drops items whose truncated service period is empty UNLESS the price is flat. Any new period math must truncate before comparison to avoid ns rounding artifacts. (`iterationEnd = iterationEnd.Truncate(streaming.MinimumWindowSizeDuration).Add(streaming.MinimumWindowSizeDuration - time.Nanosecond)`)
**Bounded phase iteration loop** — generateForAlignedItemVersion advances periodIdx until invoiceAt > iterationEnd, item deactivation, phase end, or maxSafeIter (1000) is hit; the last triggers an error log with a stack dump rather than a panic. Never write an unbounded period loop here. (`if periodIdx > maxSafeIter { logger.ErrorContext(ctx, "max iterations reached", ...); break }`)
**tracex span wrapping on every method** — Build, collectUpcomingLines, Generate, generateAligned, generateForAlignedItemVersion(+Period) all open a tracex.Start[T] span and return span.Wrap(func(ctx)...). Match the naming scheme 'billing.worker.subscription.sync...' / 'billing.worker.subscription.phaseiterator...' for new spans. (`span := tracex.Start[State](ctx, b.tracer, "billing.worker.subscription.sync.targetstate.Build")\nreturn span.Wrap(func(ctx context.Context) (State, error) { ... })`)
**InvoiceAt depends on payment term** — SubscriptionItemWithPeriods.GetInvoiceAt returns BillingPeriod.From only for flat price with InAdvancePaymentTerm; everything else is lo.Latest(ServicePeriod.To, BillingPeriod.To). StateItem.GetExpectedLine returns (nil,nil) for in-arrears flat fees with a zero-length full service period and for prorated flat amounts that round to zero. (`if flatFee.PaymentTerm == productcatalog.InAdvancePaymentTerm { return r.BillingPeriod.From }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `targetstate.go` | State/BuildInput/Builder types; Build is the entry point. collectUpcomingLines drives phase iterators and computes generationLimit from GetAlignedBillingPeriodAt. correctPeriodStartForUpcomingLines stitches continuous service-period starts for sync-ignored, force-continuous prior lines. | A nil SubscriptionView returns an empty State (signals reconciler to delete everything). CustomerDeletedAt caps the subscription via withActiveTo. correctPeriodStart only fires when the previous persisted line is subscription-managed AND has both AnnotationSubscriptionSyncIgnore and AnnotationSubscriptionSyncForceContinuousLines. |
| `phaseiterator.go` | PhaseIterator + SubscriptionItemWithPeriods. Generate/generateAligned/generateForAlignedItemVersion(+Period) project per-period items using GetAlignedBillingPeriodAt and item.Spec.GetFullServicePeriodAt. PeriodPercentage drives proration; GetMinimumBillableTime / GetInvoiceAt / HasInvoicableItems gate generation. | TimeInfinity (9999-12-31) is the sentinel upper bound; maxSafeIter=1000 caps the loop. One-time items (nil BillingCadence) go through generateOneTimeAlignedItem and may be skipped (breaks=true) if not yet billable. Zero-length cadence periods are special-cased via AsPeriod().Closed() intersection handling. |
| `targetstateitem.go` | StateItem wraps SubscriptionItemWithPeriods with CurrencyCalculator and Subscription. IsBillable, GetServicePeriod, GetExpectedLine (builds billing.GatheringLine), shouldProrate, GetExpectedLineOrErr, and discountsToBillingDiscounts. | GetExpectedLine returns (nil,nil) for zero-amount/in-arrears-zero-period cases; use GetExpectedLineOrErr (returns ErrExpectedLineIsEmpty) when a non-nil line is required. shouldProrate only applies to flat prices with ProRatingModeProratePrices and a subscription that does not end exactly at the service period end. |
| `phaseiterator_test.go` | Table-driven suite (PhaseIteratorTestSuite) using subscriptionItemViewMock and expectedIterations to assert ServicePeriod/FullServicePeriod/BillingPeriod and the composed UniqueID Key across aligned cadences, phase ends, version splits, and 1s truncation. | billingAnchorRelativeToSubscriptionStart must be negative or the test fails; expected Key strings encode the exact UniqueID format and break if the join scheme changes. |

## Anti-Patterns

- Changing the UniqueID join format without updating persistedstate's ChildUniqueReferenceID, turning updates into create+delete churn
- Comparing or storing periods without Truncate(streaming.MinimumWindowSizeDuration), reintroducing ns rounding drift
- Writing an unbounded period-generation loop instead of respecting maxSafeIter and the deactivation/phase-end exits
- panicking on a bad period instead of returning an error up through span.Wrap
- Treating GetExpectedLine's (nil,nil) result as an error rather than a legitimate not-billable signal

## Decisions

- **Project target state per phase via a stateful PhaseIterator rather than a single query** — Billing periods are alignment- and cadence-dependent and must advance period-by-period until invoice-at crosses the generation horizon, which is inherently iterative.
- **Emit a deterministic composite UniqueID per (sub,phase,item,version,period)** — The reconciler diffs target against persisted state purely by unique id, so the key must be reproducible identically on both sides.
- **Return empty State for a nil SubscriptionView** — A deleted subscription's target state is 'nothing', which forces reconciliation to delete dependent invoice lines/charges.

## Example: Build target state for a subscription as of a time

```
func (b Builder) Build(ctx context.Context, input BuildInput) (State, error) {\n\tspan := tracex.Start[State](ctx, b.tracer, "billing.worker.subscription.sync.targetstate.Build")\n\treturn span.Wrap(func(ctx context.Context) (State, error) {\n\t\tif err := input.Validate(); err != nil { return State{}, fmt.Errorf("validating input: %w", err) }\n\t\tif input.SubscriptionView == nil { return State{MaxGenerationTimeLimit: input.AsOf}, nil }\n\t\tsubs := *input.SubscriptionView\n\t\tupcomingLinesResult, err := b.collectUpcomingLines(ctx, subs, input.AsOf)\n\t\tif err != nil { return State{}, fmt.Errorf("collecting upcoming lines: %w", err) }\n\t\tinScopeLines, err := b.correctPeriodStartForUpcomingLines(ctx, subs.Subscription.ID, upcomingLinesResult.Lines, input.Persisted)\n\t\tif err != nil { return State{}, err }\n\t\tcurrencyCalculator, err := subs.Subscription.Currency.Calculator()\n\t\tif err != nil { return State{}, err }\n\t\treturn State{Items: lo.Map(inScopeLines, func(item SubscriptionItemWithPeriods, _ int) StateItem {\n\t\t\treturn StateItem{SubscriptionItemWithPeriods: item, CurrencyCalculator: currencyCalculator, Subscription: subs.Subscription}\n\t\t}), MaxGenerationTimeLimit: upcomingLinesResult.SubscriptionMaxGenerationTimeLimit}, nil\n\t})\n}
```

<!-- archie:ai-end -->
