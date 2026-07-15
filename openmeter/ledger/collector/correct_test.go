package collector

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgerbreakage "github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	ledgertestutils "github.com/openmeterio/openmeter/openmeter/ledger/testutils"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestCorrectCollectedAccruedUsesReverseFeatureAwareCollectionOrder(t *testing.T) {
	env := ledgertestutils.NewIntegrationEnv(t, "collector-correct")
	collector := newTestAccrualCollector(env)
	corrector := newTestAccrualCorrector(env, nil)

	// given:
	// - one spend charge fully consumes a feature-restricted source and an unrestricted source
	// when:
	// - half of the collected amount is corrected
	// then:
	// - correction unwinds in reverse collection order and keeps the remaining spend provenance
	priority := 1                      // both sources have the same priority, so feature-aware ordering decides collection order.
	restrictedAmount := int64(30)      // feature-restricted credit is the first 30 collected for the matching feature.
	unrestrictedAmount := int64(10)    // unrestricted credit supplies the remaining 10 collected for the spend.
	correctionAmount := int64(20)      // correcting 20 returns all 10 unrestricted credit plus 10 restricted credit.
	restrictedRemaining := int64(10)   // restricted started at 30, collected 30, then 10 was corrected back.
	unrestrictedRemaining := int64(10) // unrestricted started at 10, collected 10, then the full 10 was corrected back.
	accruedRemaining := int64(20)      // original 40 collection minus 20 correction remains accrued.
	restricted := fundPriorityWithFeatures(t, env, priority, restrictedAmount, []string{"api-calls"})
	unrestricted := fundPriorityWithFeatures(t, env, priority, unrestrictedAmount, nil)
	servicePeriod := testServicePeriod(env)
	chargeID := testChargeID(1)

	allocations, err := collector.collect(t.Context(), CollectToAccruedInput{
		Namespace:         env.Namespace,
		ChargeID:          chargeID,
		CustomerID:        env.CustomerID.ID,
		BookedAt:          env.Now(),
		SourceBalanceAsOf: env.Now(),
		Currency:          env.Currency,
		SettlementMode:    productcatalog.CreditThenInvoiceSettlementMode,
		ServicePeriod:     servicePeriod,
		FeatureKey:        "api-calls",
		Amount:            alpacadecimal.NewFromInt(restrictedAmount + unrestrictedAmount),
	})
	require.NoError(t, err)
	require.Len(t, allocations, 2) // two allocations: restricted source first, unrestricted source second.

	realizations := realizationsFromAllocations(env, allocations)
	currency, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeFiat).
		WithCode(env.Currency).
		Build()
	require.NoError(t, err)

	corrections, err := realizations.CreateCorrectionRequest(alpacadecimal.NewFromInt(-correctionAmount), currency)
	require.NoError(t, err)
	require.Len(t, corrections, 2) // the 20 correction spans the unrestricted source and part of the restricted source.

	_, err = corrector.correct(t.Context(), CorrectCollectedAccruedInput{
		Namespace:   env.Namespace,
		ChargeID:    chargeID,
		CustomerID:  env.CustomerID.ID,
		AllocateAt:  env.Now(),
		Corrections: corrections,
	})
	require.NoError(t, err)

	require.True(t, env.SumBalance(t, restricted).Equal(alpacadecimal.NewFromInt(restrictedRemaining)))
	require.True(t, env.SumBalance(t, unrestricted).Equal(alpacadecimal.NewFromInt(unrestrictedRemaining)))
	require.True(t, env.SumBalance(t, env.AccruedSubAccount(t)).Equal(alpacadecimal.NewFromInt(accruedRemaining)))
	requireAccruedBalanceBuckets(t, env, map[string]float64{
		sourceSpendChargeKey(nil, &chargeID): float64(accruedRemaining), // only the uncorrected 20 remains accrued to the spend charge.
	})
}

func TestCorrectCollectedAccruedReopensBreakageByReverseFeatureAwareCollectionOrder(t *testing.T) {
	env := ledgertestutils.NewIntegrationEnv(t, "collector-correct-breakage")
	breakageService := newTestBreakageService(t, env)
	collector := newTestAccrualCollectorWithBreakage(env, breakageService)
	corrector := newTestAccrualCorrector(env, breakageService)

	// given:
	// - two expiring sources are fully consumed and their planned breakage is released
	// when:
	// - the last collected source is corrected
	// then:
	// - the matching plan reopens and both FBO/breakage provenance point at that source
	priority := 1                              // both plans share priority, so expiry/collection order decides which plan reopens.
	restrictedAmount := int64(30)              // restricted plan is consumed first for the matching feature.
	unrestrictedAmount := int64(10)            // unrestricted plan is consumed last and therefore corrected first.
	correctionAmount := int64(10)              // correction exactly reopens the unrestricted 10 plan.
	restrictedExpiresAfter := 20 * time.Hour   // restricted plan expires later, so the earlier unrestricted plan is the one reopened.
	unrestrictedExpiresAfter := 10 * time.Hour // unrestricted plan expires first and is the only open plan after the 10 correction.
	restrictedSourceCharge := testChargeID(1)
	unrestrictedSourceCharge := testChargeID(2)
	restrictedPlanID := bookExpiringCreditWithFeatures(t, env, breakageService, priority, restrictedAmount, []string{"api-calls"}, &restrictedSourceCharge, env.Now().Add(restrictedExpiresAfter))
	unrestrictedPlanID := bookExpiringCreditWithFeatures(t, env, breakageService, priority, unrestrictedAmount, nil, &unrestrictedSourceCharge, env.Now().Add(unrestrictedExpiresAfter))
	servicePeriod := testServicePeriod(env)
	chargeID := testChargeID(3)

	allocations, err := collector.collect(t.Context(), CollectToAccruedInput{
		Namespace:         env.Namespace,
		ChargeID:          chargeID,
		CustomerID:        env.CustomerID.ID,
		BookedAt:          env.Now(),
		SourceBalanceAsOf: env.Now(),
		Currency:          env.Currency,
		SettlementMode:    productcatalog.CreditThenInvoiceSettlementMode,
		ServicePeriod:     servicePeriod,
		FeatureKey:        "api-calls",
		Amount:            alpacadecimal.NewFromInt(restrictedAmount + unrestrictedAmount),
	})
	require.NoError(t, err)
	require.Len(t, allocations, 2) // two consumed plans should produce two release records.

	openPlans, err := breakageService.ListPlans(t.Context(), ledgerbreakage.ListPlansInput{
		CustomerID: env.CustomerID,
		Currency:   env.Currency,
		AsOf:       env.Now(),
	})
	require.NoError(t, err)
	require.Empty(t, openPlans)

	realizations := realizationsFromAllocations(env, allocations)
	currency, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeFiat).
		WithCode(env.Currency).
		Build()
	require.NoError(t, err)

	corrections, err := realizations.CreateCorrectionRequest(alpacadecimal.NewFromInt(-correctionAmount), currency)
	require.NoError(t, err)

	_, err = corrector.correct(t.Context(), CorrectCollectedAccruedInput{
		Namespace:   env.Namespace,
		ChargeID:    chargeID,
		CustomerID:  env.CustomerID.ID,
		AllocateAt:  env.Now(),
		Corrections: corrections,
	})
	require.NoError(t, err)

	openPlans, err = breakageService.ListPlans(t.Context(), ledgerbreakage.ListPlansInput{
		CustomerID: env.CustomerID,
		Currency:   env.Currency,
		AsOf:       env.Now(),
	})
	require.NoError(t, err)
	require.Len(t, openPlans, 1) // only the unrestricted plan is reopened by the 10 correction.
	require.Equal(t, unrestrictedPlanID, openPlans[0].ID.ID)
	require.NotEqual(t, restrictedPlanID, openPlans[0].ID.ID)
	require.True(t, openPlans[0].OpenAmount.Equal(alpacadecimal.NewFromInt(correctionAmount)), "open amount: %s", openPlans[0].OpenAmount)
	requireFBOBalanceBucketsAt(t, env, env.Now(), map[string]float64{
		sourceSpendChargeKey(&unrestrictedSourceCharge, nil): float64(correctionAmount), // current FBO is restored by the corrected 10 before future breakage reopen nets it out.
	})
	requireBreakageBalanceBuckets(t, env, map[string]float64{
		sourceSpendChargeKey(&unrestrictedSourceCharge, nil): float64(correctionAmount), // reopened breakage is source-attributed, but not spend-attributed.
	})
}

func TestCorrectCollectedAccruedBreakageReopenTracksSourceOnBreakage(t *testing.T) {
	env := ledgertestutils.NewIntegrationEnv(t, "collector-correct-breakage")
	breakageService := newTestBreakageService(t, env)
	collector := newTestAccrualCollectorWithBreakage(env, breakageService)
	corrector := newTestAccrualCorrector(env, breakageService)

	// given:
	// - sourced expiring credit has been consumed by one spend charge
	// when:
	// - part of that collection is corrected
	// then:
	// - reopened breakage is attributed to the original source without spend provenance
	priority := 1 // only one source is available, so priority just selects its FBO route.
	sourceCharge := testChargeID(1)
	spendCharge := testChargeID(2)
	sourceAmount := int64(20)      // the original expiring source is fully consumed before correction.
	correctionAmount := int64(8)   // correction reopens this much of the previously released breakage.
	expiresAfter := 10 * time.Hour // the single source expires in the future, so correction reopens one future breakage plan.
	bookExpiringCreditWithFeatures(t, env, breakageService, priority, sourceAmount, nil, &sourceCharge, env.Now().Add(expiresAfter))

	allocations, err := collector.collect(t.Context(), collectToAccruedInputForTest(
		env,
		spendCharge,
		alpacadecimal.NewFromInt(sourceAmount),
		productcatalog.CreditThenInvoiceSettlementMode,
	))
	require.NoError(t, err)
	require.Len(t, allocations, 1) // one source produces one allocation and one release to reopen.

	realizations := realizationsFromAllocations(env, allocations)
	currency, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeFiat).
		WithCode(env.Currency).
		Build()
	require.NoError(t, err)

	corrections, err := realizations.CreateCorrectionRequest(alpacadecimal.NewFromInt(-correctionAmount), currency)
	require.NoError(t, err)

	_, err = corrector.correct(t.Context(), CorrectCollectedAccruedInput{
		Namespace:   env.Namespace,
		ChargeID:    spendCharge,
		CustomerID:  env.CustomerID.ID,
		AllocateAt:  env.Now(),
		Corrections: corrections,
	})
	require.NoError(t, err)

	openPlans, err := breakageService.ListPlans(t.Context(), ledgerbreakage.ListPlansInput{
		CustomerID: env.CustomerID,
		Currency:   env.Currency,
		AsOf:       env.Now(),
	})
	require.NoError(t, err)
	require.Len(t, openPlans, 1) // the single release is reopened into one open plan.
	require.True(t, openPlans[0].OpenAmount.Equal(alpacadecimal.NewFromInt(correctionAmount)), "open amount: %s", openPlans[0].OpenAmount)
	requireFBOBalanceBucketsAt(t, env, env.Now(), map[string]float64{
		sourceSpendChargeKey(&sourceCharge, nil): float64(correctionAmount), // current FBO is restored by the corrected 8 before future breakage reopen nets it out.
	})
	requireAccruedBalanceBuckets(t, env, map[string]float64{
		sourceSpendChargeKey(&sourceCharge, &spendCharge): float64(sourceAmount - correctionAmount), // 20 original accrued minus 8 corrected leaves 12 accrued to the spend.
	})
	requireBreakageBalanceBuckets(t, env, map[string]float64{
		sourceSpendChargeKey(&sourceCharge, nil): float64(correctionAmount), // the corrected slice is broken again under the original source, without spend attribution.
	})
}

func TestCorrectCollectedAccruedPreservesSourceAndSpendBuckets(t *testing.T) {
	env := ledgertestutils.NewIntegrationEnv(t, "collector-correct-provenance")
	collector := newTestAccrualCollector(env)
	corrector := newTestAccrualCorrector(env, nil)

	// given:
	// - one spend charge consumes two same-route source charges
	// when:
	// - the collection is partially corrected
	// then:
	// - corrected value returns to the original source bucket
	// - remaining accrued value keeps the source x spend split
	sourceCharge1 := testChargeID(1)
	sourceCharge2 := testChargeID(2)
	spendCharge := testChargeID(3)
	priority := 1                   // both sources share one FBO route, so provenance is the only split.
	firstSourceAmount := int64(10)  // first source is fully consumed by the spend.
	secondSourceAmount := int64(20) // second source is partially restored by the correction.
	correctionAmount := int64(15)   // correction unwinds 15 from the second source by reverse collection order.
	fundSourceCharge(t, env, sourceCharge1, priority, firstSourceAmount)
	fundSourceCharge(t, env, sourceCharge2, priority, secondSourceAmount)

	allocations, err := collector.collect(t.Context(), collectToAccruedInputForTest(
		env,
		spendCharge,
		alpacadecimal.NewFromInt(firstSourceAmount+secondSourceAmount),
		productcatalog.CreditThenInvoiceSettlementMode,
	))
	require.NoError(t, err)
	require.Len(t, allocations, 1) // same-route sources coalesce into one ledger transaction allocation.

	realizations := realizationsFromAllocations(env, allocations)
	currency, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeFiat).
		WithCode(env.Currency).
		Build()
	require.NoError(t, err)

	corrections, err := realizations.CreateCorrectionRequest(alpacadecimal.NewFromInt(-correctionAmount), currency)
	require.NoError(t, err)

	_, err = corrector.correct(t.Context(), CorrectCollectedAccruedInput{
		Namespace:   env.Namespace,
		ChargeID:    spendCharge,
		CustomerID:  env.CustomerID.ID,
		AllocateAt:  env.Now(),
		Corrections: corrections,
	})
	require.NoError(t, err)

	requireFBOBalanceBuckets(t, env, map[string]float64{
		sourceSpendChargeKey(&sourceCharge2, nil): float64(correctionAmount), // only source 2 is restored by the partial correction.
	})
	requireAccruedBalanceBuckets(t, env, map[string]float64{
		sourceSpendChargeKey(&sourceCharge1, &spendCharge): float64(firstSourceAmount),                     // source 1 remains fully accrued.
		sourceSpendChargeKey(&sourceCharge2, &spendCharge): float64(secondSourceAmount - correctionAmount), // source 2 keeps only the uncorrected accrued remainder.
	})
}

func TestCorrectCollectedAccruedPartiallyReversesAdvanceBackedCollection(t *testing.T) {
	env := ledgertestutils.NewIntegrationEnv(t, "collector-correct-advance")
	collector := newTestAccrualCollector(env)
	corrector := newTestAccrualCorrector(env, nil)

	// given:
	// - credit-only usage has no real source, so it creates advance receivable and accrued exposure
	// when:
	// - part of that advance-backed collection is corrected
	// then:
	// - the remaining receivable/accrued exposure keeps spend provenance with no source
	advanceAmount := int64(30)    // original usage creates 30 of advance-backed receivable and accrued exposure.
	correctionAmount := int64(10) // correction removes 10 from the advance-backed exposure.
	remainingAdvance := int64(20) // 30 original advance minus 10 correction remains open/accrued.
	servicePeriod := testServicePeriod(env)
	chargeID := testChargeID(1)

	allocations, err := collector.collect(t.Context(), CollectToAccruedInput{
		Namespace:         env.Namespace,
		ChargeID:          chargeID,
		CustomerID:        env.CustomerID.ID,
		BookedAt:          env.Now(),
		SourceBalanceAsOf: env.Now(),
		Currency:          env.Currency,
		SettlementMode:    productcatalog.CreditOnlySettlementMode,
		ServicePeriod:     servicePeriod,
		Amount:            alpacadecimal.NewFromInt(advanceAmount),
	})
	require.NoError(t, err)
	require.Len(t, allocations, 1) // credit-only shortfall creates one advance-backed allocation.

	realizations := realizationsFromAllocations(env, allocations)
	_, err = corrector.correct(t.Context(), CorrectCollectedAccruedInput{
		Namespace:  env.Namespace,
		ChargeID:   chargeID,
		CustomerID: env.CustomerID.ID,
		AllocateAt: env.Now(),
		Corrections: creditrealization.CorrectionRequest{
			{
				Allocation: realizations[0],
				Amount:     alpacadecimal.NewFromInt(-correctionAmount),
			},
		},
	})
	require.NoError(t, err)

	require.True(t, env.SumBalance(t, env.ReceivableSubAccount(t)).Equal(alpacadecimal.NewFromInt(-remainingAdvance)))
	require.True(t, env.SumBalance(t, env.FBOSubAccount(t, ledger.DefaultCustomerFBOPriority)).Equal(alpacadecimal.Zero))
	require.True(t, env.SumBalance(t, env.AccruedSubAccount(t)).Equal(alpacadecimal.NewFromInt(remainingAdvance)))
	requireFBOProvenanceBalanceBuckets(t, env, map[string]float64{
		sourceSpendChargeKey(nil, nil):       float64(correctionAmount),  // accrual correction frees 10 back into unspent advance/FBO value.
		sourceSpendChargeKey(nil, &chargeID): float64(-correctionAmount), // companion receivable correction removes 10 from the spend-backed advance issuance.
	})
	requireReceivableBalanceBuckets(t, env, map[string]float64{
		sourceSpendChargeKey(nil, &chargeID): float64(-remainingAdvance), // receivable remains negative for the uncorrected 20 advance.
	})
	requireAccruedBalanceBuckets(t, env, map[string]float64{
		sourceSpendChargeKey(nil, &chargeID): float64(remainingAdvance), // accrued keeps the uncorrected 20 under spend provenance with no source.
	})
}

func newTestAccrualCorrector(
	env *ledgertestutils.IntegrationEnv,
	breakageService ledgerbreakage.Service,
) *accrualCorrector {
	return &accrualCorrector{
		ledger: env.Deps.HistoricalLedger,
		deps: transactions.ResolverDependencies{
			AccountService: env.Deps.ResolversService,
			AccountCatalog: env.Deps.AccountService,
			BalanceQuerier: env.Deps.HistoricalLedger,
		},
		breakage:           breakageService,
		transactionManager: enttx.NewCreator(env.DB),
	}
}

func testServicePeriod(env *ledgertestutils.IntegrationEnv) timeutil.ClosedPeriod {
	return timeutil.ClosedPeriod{
		From: env.Now().Add(-time.Hour),
		To:   env.Now(),
	}
}

func realizationsFromAllocations(env *ledgertestutils.IntegrationEnv, allocations creditrealization.CreateAllocationInputs) creditrealization.Realizations {
	now := env.Now()

	out := make(creditrealization.Realizations, 0, len(allocations))
	for i, allocation := range allocations.AsCreateInputs() {
		allocation.ID = ulid.Make().String()
		out = append(out, creditrealization.Realization{
			NamespacedModel: models.NamespacedModel{
				Namespace: env.Namespace,
			},
			ManagedModel: models.ManagedModel{
				CreatedAt: now,
				UpdatedAt: now,
			},
			CreateInput: allocation,
			SortHint:    i,
		})
	}

	return out
}
