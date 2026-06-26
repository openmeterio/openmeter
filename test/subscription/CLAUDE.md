# subscription

<!-- archie:ai-start -->

> Integration test suite (package subscription_test) for end-to-end subscription lifecycle scenarios that cross the subscription, productcatalog, billing, and billing/worker/subscriptionsync boundaries. Each scenario_*_test.go file wires a full real-service stack via setup() and drives one named bug-regression or alignment scenario (aligned edits, edit+cancel, entitlement-in-next-phase, first-of-month/anchored billing).

## Patterns

**Single shared setup() builds the full real-service stack** — Every test calls tDeps := setup(t, setupConfig{}) then defer tDeps.cleanup(t). setup() constructs real adapters/services (app, billing, taxcode, subscriptionsync) on top of subscriptiontestutils.SetupDBDeps + NewService — not mocks — so scenarios exercise production code paths. (`tDeps := setup(t, setupConfig{}); defer tDeps.cleanup(t)`)
**Hardcoded test-namespace** — All scenarios use namespace := "test-namespace"; this string is baked into setup() (app CreateApp calls use it). Do not parameterize the namespace per-test — the framework's sandbox app provisioning assumes it. (`namespace := "test-namespace"`)
**clock.SetTime drives the timeline** — Tests freeze and advance time with pkg/clock: clock.SetTime(currentTime) at start, then reassign currentTime = currentTime.Add(...) / clock.Now().Add(...) and SetTime again before each lifecycle step (edit, cancel, sync). (`currentTime = currentTime.Add(time.Minute); clock.SetTime(currentTime)`)
**Standard plan->publish->customer->subscribe arc** — Scenarios build features (FeatureConnector.CreateFeature / CreateExampleFeatures), then PlanService.CreatePlan + PublishPlan, CustomerService.CreateCustomer, build a pcsubscription.PlanInput via FromRef, then pcSubscriptionService.Create with a CreateSubscriptionWorkflowInput. (`pi := &pcsubscription.PlanInput{}; pi.FromRef(&pcsubscription.PlanRefInput{Key: p.Key, Version: &p.Version})`)
**Edits go through the workflow EditRunning + patches** — Subscription edits use subscriptionWorkflowService.EditRunning with []subscription.Patch (patch.PatchRemoveItem then patch.PatchAddItem with a full SubscriptionItemSpec) and an explicit subscription.Timing. Cancels use subscriptionService.Cancel with subscription.Timing. (`EditRunning(ctx, s.NamespacedID, []subscription.Patch{patch.PatchRemoveItem{...}, patch.PatchAddItem{...}}, subscription.Timing{Enum: lo.ToPtr(subscription.TimingImmediate)})`)
**Billing assertions via gathering invoices + sync** — Billing scenarios create a profile from minimalCreateProfileInputTemplate, call subscriptionSyncService.SyncByView(ctx, view, until), then assert on billingService.ListGatheringInvoices results (GatheringLine ServicePeriod/InvoiceAt), grouping lines with lo.GroupBy on FeatureKey/ChildUniqueReferenceID. (`require.NoError(t, tDeps.subscriptionSyncService.SyncByView(ctx, view, firstOfMonth.AddDate(0, 1, 0)))`)
**Durations via datetime.MustParseDuration(t, ...)** — ISO durations are built with datetime.MustParseDuration(t, "P1M") in test bodies; profile template uses lo.Must(datetime.ISODurationString("P1D").Parse()). Prices use alpacadecimal.NewFromInt and productcatalog.NewPriceFrom. (`BillingCadence: datetime.MustParseDuration(t, "P1M")`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `framework_test.go` | Defines testDeps struct, setupConfig, setup(t, cfg) which wires the entire real-service stack, and minimalCreateProfileInputTemplate(appID) for billing profiles. The only non-scenario file; the shared harness all scenarios depend on. | setup() creates TWO sandbox apps ('Test Sandbox' and 'Sandbox') and returns the 'Sandbox' one as tDeps.sandboxApp. billingService is wrapped with a MockableCalculator via WithInvoiceCalculator. Profile interval is PT0S so invoices collect immediately unless a scenario overrides it. |
| `scenario_editaligned_test.go` | TestEditingEntitlementOfAlignedSub — verifies that editing a metered entitlement item preserves the entitlement's CurrentUsagePeriod (cadence alignment) across PatchRemoveItem+PatchAddItem. | Asserts the edited item is at index [1] in ItemsByKey (the original is [0]); compares CurrentUsagePeriod.From/To equality and CreatedAt strictly increasing. |
| `scenario_editcancel_test.go` | TestEditingAndCanceling — boolean-entitlement plan; creates a main customer plus 10 extra customers/subscriptions, edits one sub, then cancels it. Regression coverage for edit-then-cancel sequencing. | Extra customers use SubjectKeys subject_2..subject_11 (fmt.Sprintf with i+2). No billing assertions here — purely lifecycle success (require.NoError). |
| `scenario_entinnextphase_test.go` | TestSubWithMeteredEntitlement — two-phase plan (1-week first phase, then second) each with a metered entitlement ratecard; reproduces a creation failure when an entitlement spans into the next phase. The assertion is simply that Create succeeds. | Comment 'THIS IS THE TEST, it used to fail' marks the regression intent; BillingAnchor is nil (aligns billing to subscription start). |
| `scenario_firstofmonth_test.go` | TestBillingOnFirstOfMonth and TestAnchoredAlignment_MidMonthStart_EarlyCancel_IssueNextAnchor — the most assertion-heavy file; mixes in-arrears monthly, in-advance flat-fee monthly, and in-arrears daily ratecards, then checks gathering-invoice line ServicePeriod/InvoiceAt for proration and anchored alignment. | Uses BillingAnchor (&firstOfMonth) on the workflow input AND AlignmentKindAnchored + AnchoredAlignmentDetail on the profile in the second test. Daily ratecard produces 16 lines (15th->30th) with the first being a partial half-day line. Subtests use t.Run. |

## Anti-Patterns

- Constructing services from app/common wiring instead of the underlying adapter/service constructors used in setup() — risks test-only import cycles (see subscriptiontestutils guidance).
- Using context.Background() time/now or time.Now() instead of pkg/clock — scenarios depend on a frozen, advanceable clock and would become non-deterministic.
- Changing the hardcoded 'test-namespace' string — sandbox app provisioning and profile inputs assume it.
- Asserting subscription edits by mutating the original view in place — edited items are appended (new index) into ItemsByKey, the original entry is retained.
- Calling lower-level billing/charge adapters directly to model usage instead of driving through SyncByView and ListGatheringInvoices.

## Decisions

- **Each scenario file is a self-contained named regression test rather than table-driven cases.** — Each scenario reproduces a distinct historical bug or alignment edge case with bespoke plan shapes and timeline manipulation; table-driving would obscure the per-scenario intent comments (e.g. 'it used to fail').
- **setup() builds the real billing + subscriptionsync stack and only mocks the streaming connector and invoice calculator.** — These tests exist to validate the subscription->billing sync bridge end to end, so production service paths must run; only external usage data (MockStreamingConnector) and final invoice calc are substituted.

## Example: Standard create-plan-and-subscribe arc shared by every scenario

```
tDeps := setup(t, setupConfig{})
defer tDeps.cleanup(t)
clock.SetTime(currentTime)

f, _ := tDeps.FeatureConnector.CreateFeature(ctx, feature.CreateFeatureInputs{
    Name: "Example Feature", Key: "test_feature_1", Namespace: namespace,
    MeterID: lo.ToPtr(tDeps.ExampleMeterID),
})
p, _ := tDeps.PlanService.CreatePlan(ctx, plan.CreatePlanInput{ /* PlanMeta + Phases + RateCards */ })
p, _ = tDeps.PlanService.PublishPlan(ctx, plan.PublishPlanInput{
    NamespacedID: p.NamespacedID,
    EffectivePeriod: productcatalog.EffectivePeriod{EffectiveFrom: lo.ToPtr(currentTime)},
})
c, _ := tDeps.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{ /* ... */ })
pi := &pcsubscription.PlanInput{}
// ...
```

<!-- archie:ai-end -->
