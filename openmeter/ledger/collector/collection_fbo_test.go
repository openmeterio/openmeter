package collector

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgerbreakage "github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	ledgerbreakageadapter "github.com/openmeterio/openmeter/openmeter/ledger/breakage/adapter"
	ledgertestutils "github.com/openmeterio/openmeter/openmeter/ledger/testutils"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestCollectCustomerFBOUsesPriorityOrder(t *testing.T) {
	env := ledgertestutils.NewIntegrationEnv(t, "collector")
	collector := newTestAccrualCollector(env)

	priorityTwo := fundPriority(t, env, 2, 50)
	priorityOne := fundPriority(t, env, 1, 30)

	sources, err := collectCustomerFBOForTest(t, env, collector, alpacadecimal.NewFromInt(60), env.Now())
	require.NoError(t, err)
	require.Len(t, sources, 2)

	require.Equal(t, priorityOne.Address().SubAccountID(), sources[0].Address.SubAccountID())
	require.True(t, alpacadecimal.NewFromInt(30).Equal(sources[0].Amount))
	require.Equal(t, priorityTwo.Address().SubAccountID(), sources[1].Address.SubAccountID())
	require.True(t, alpacadecimal.NewFromInt(30).Equal(sources[1].Amount))
}

func TestCollectCustomerFBOUsesSubAccountIDTieBreaker(t *testing.T) {
	env := ledgertestutils.NewIntegrationEnv(t, "collector")
	collector := newTestAccrualCollector(env)

	costBasisOne := alpacadecimal.NewFromInt(1)
	costBasisTwo := alpacadecimal.NewFromInt(2)

	first := fundPriorityWithCostBasis(t, env, 1, 10, &costBasisOne)
	second := fundPriorityWithCostBasis(t, env, 1, 10, &costBasisTwo)

	sources, err := collectCustomerFBOForTest(t, env, collector, alpacadecimal.NewFromInt(20), env.Now())
	require.NoError(t, err)
	require.Len(t, sources, 2)

	expected := []ledger.SubAccount{first, second}
	if expected[0].Address().SubAccountID() > expected[1].Address().SubAccountID() {
		expected[0], expected[1] = expected[1], expected[0]
	}

	require.Equal(t, expected[0].Address().SubAccountID(), sources[0].Address.SubAccountID())
	require.Equal(t, expected[1].Address().SubAccountID(), sources[1].Address.SubAccountID())
}

func TestCollectCustomerFBOUsesAsOfBalance(t *testing.T) {
	env := ledgertestutils.NewIntegrationEnv(t, "collector")
	collector := newTestAccrualCollector(env)

	source := fundPriority(t, env, 1, 50)
	bookFutureFBOCollection(t, env, 1, 30, env.Now().AddDate(0, 0, 1))

	currentSources, err := collectCustomerFBOForTest(t, env, collector, alpacadecimal.NewFromInt(50), env.Now())
	require.NoError(t, err)
	require.Len(t, currentSources, 1)
	require.Equal(t, source.Address().SubAccountID(), currentSources[0].Address.SubAccountID())
	require.True(t, alpacadecimal.NewFromInt(50).Equal(currentSources[0].Amount), "current amount: %s", currentSources[0].Amount)

	futureAsOf := env.Now().AddDate(0, 0, 1)
	futureSources, err := collectCustomerFBOForTest(t, env, collector, alpacadecimal.NewFromInt(50), futureAsOf)
	require.NoError(t, err)
	require.Len(t, futureSources, 1)
	require.Equal(t, source.Address().SubAccountID(), futureSources[0].Address.SubAccountID())
	require.True(t, alpacadecimal.NewFromInt(20).Equal(futureSources[0].Amount), "future amount: %s", futureSources[0].Amount)
}

func TestCollectCustomerFBOFiltersByFeatureEligibility(t *testing.T) {
	env := ledgertestutils.NewIntegrationEnv(t, "collector")
	collector := newTestAccrualCollector(env)

	unrestricted := fundPriorityWithFeatures(t, env, 1, 10, nil)
	matchingKey := fundPriorityWithFeatures(t, env, 1, 30, []string{"api-calls"})
	fundPriorityWithFeatures(t, env, 1, 40, []string{"storage"})

	sources, err := collectCustomerFBOForFeatureForTest(
		t,
		env,
		collector,
		"api-calls",
		alpacadecimal.NewFromInt(200),
		env.Now(),
	)
	require.NoError(t, err)
	require.Len(t, sources, 2)

	require.Equal(t, matchingKey.Address().SubAccountID(), sources[0].Address.SubAccountID())
	require.True(t, alpacadecimal.NewFromInt(30).Equal(sources[0].Amount), "restricted source amount: %s", sources[0].Amount)
	require.Equal(t, unrestricted.Address().SubAccountID(), sources[1].Address.SubAccountID())
	require.True(t, alpacadecimal.NewFromInt(10).Equal(sources[1].Amount), "unrestricted source amount: %s", sources[1].Amount)

	unattributedSources, err := collectCustomerFBOForFeatureForTest(
		t,
		env,
		collector,
		"",
		alpacadecimal.NewFromInt(200),
		env.Now(),
	)
	require.NoError(t, err)
	require.Len(t, unattributedSources, 1)
	require.Equal(t, unrestricted.Address().SubAccountID(), unattributedSources[0].Address.SubAccountID())
	require.True(t, alpacadecimal.NewFromInt(10).Equal(unattributedSources[0].Amount))
}

func TestCollectCustomerFBOFiltersBreakageByFeatureEligibility(t *testing.T) {
	env := ledgertestutils.NewIntegrationEnv(t, "collector")
	breakageService := newTestBreakageService(t, env)
	collector := newTestAccrualCollectorWithBreakage(env, breakageService)

	expiresAt := env.Now().Add(10 * time.Hour)
	bookExpiringCreditWithFeatures(t, env, breakageService, 1, 10, nil, nil, expiresAt)
	bookExpiringCreditWithFeatures(t, env, breakageService, 1, 30, []string{"api-calls"}, nil, expiresAt)
	bookExpiringCreditWithFeatures(t, env, breakageService, 1, 40, []string{"storage"}, nil, expiresAt)

	sources, err := collectCustomerFBOForFeatureForTest(
		t,
		env,
		collector,
		"api-calls",
		alpacadecimal.NewFromInt(200),
		env.Now(),
	)
	require.NoError(t, err)
	require.Len(t, sources, 2)

	require.Equal(t, []string{"api-calls"}, sources[0].Address.Route().Route().Features)
	require.True(t, alpacadecimal.NewFromInt(30).Equal(sources[0].Amount), "restricted source amount: %s", sources[0].Amount)
	require.Empty(t, sources[1].Address.Route().Route().Features)
	require.True(t, alpacadecimal.NewFromInt(10).Equal(sources[1].Amount), "unrestricted source amount: %s", sources[1].Amount)

	unattributedSources, err := collectCustomerFBOForFeatureForTest(
		t,
		env,
		collector,
		"",
		alpacadecimal.NewFromInt(200),
		env.Now(),
	)
	require.NoError(t, err)
	require.Len(t, unattributedSources, 1)
	require.Empty(t, unattributedSources[0].Address.Route().Route().Features)
	require.True(t, alpacadecimal.NewFromInt(10).Equal(unattributedSources[0].Amount), "unrestricted source amount: %s", unattributedSources[0].Amount)
}

func TestCollectCustomerFBOUsesPriorityBeforeFeatureRestriction(t *testing.T) {
	env := ledgertestutils.NewIntegrationEnv(t, "collector")
	collector := newTestAccrualCollector(env)

	higherPriorityUnrestricted := fundPriorityWithFeatures(t, env, 1, 10, nil)
	lowerPriorityRestricted := fundPriorityWithFeatures(t, env, 2, 30, []string{"api-calls"})

	sources, err := collectCustomerFBOForFeatureForTest(
		t,
		env,
		collector,
		"api-calls",
		alpacadecimal.NewFromInt(40),
		env.Now(),
	)
	require.NoError(t, err)
	require.Len(t, sources, 2)

	require.Equal(t, higherPriorityUnrestricted.Address().SubAccountID(), sources[0].Address.SubAccountID())
	require.True(t, alpacadecimal.NewFromInt(10).Equal(sources[0].Amount), "higher-priority unrestricted amount: %s", sources[0].Amount)
	require.Equal(t, lowerPriorityRestricted.Address().SubAccountID(), sources[1].Address.SubAccountID())
	require.True(t, alpacadecimal.NewFromInt(30).Equal(sources[1].Amount), "lower-priority restricted amount: %s", sources[1].Amount)
}

func TestCollectCustomerFBOReleasesBreakageInExpiryOrder(t *testing.T) {
	env := ledgertestutils.NewIntegrationEnv(t, "collector")
	breakageService := newTestBreakageService(t, env)
	collector := newTestAccrualCollectorWithBreakage(env, breakageService)

	firstPlanID := bookExpiringCredit(t, env, breakageService, 1, 10, env.Now().Add(10*time.Hour))
	secondPlanID := bookExpiringCredit(t, env, breakageService, 1, 15, env.Now().Add(15*time.Hour))

	servicePeriod := timeutil.ClosedPeriod{
		From: env.Now().Add(-time.Hour),
		To:   env.Now(),
	}
	allocations, err := collector.collect(t.Context(), CollectToAccruedInput{
		Namespace:         env.Namespace,
		ChargeID:          testChargeID(1),
		CustomerID:        env.CustomerID.ID,
		BookedAt:          env.Now(),
		SourceBalanceAsOf: env.Now(),
		Currency:          env.Currency,
		SettlementMode:    productcatalog.CreditThenInvoiceSettlementMode,
		ServicePeriod:     servicePeriod,
		Amount:            alpacadecimal.NewFromInt(15),
	})
	require.NoError(t, err)
	require.Len(t, allocations, 1)
	require.True(t, allocations[0].Amount.Equal(alpacadecimal.NewFromInt(15)))

	openPlans, err := breakageService.ListPlans(t.Context(), ledgerbreakage.ListPlansInput{
		CustomerID: env.CustomerID,
		Currency:   env.Currency,
		AsOf:       env.Now(),
	})
	require.NoError(t, err)
	require.Len(t, openPlans, 1)
	require.Equal(t, secondPlanID, openPlans[0].ID.ID)
	require.NotEqual(t, firstPlanID, openPlans[0].ID.ID)
	require.True(t, openPlans[0].OpenAmount.Equal(alpacadecimal.NewFromInt(10)), "open amount: %s", openPlans[0].OpenAmount)
}

func TestCollectCustomerFBOBreakageReleaseTracksSpendOnFBOAndSourceOnBreakage(t *testing.T) {
	env := ledgertestutils.NewIntegrationEnv(t, "collector")
	breakageService := newTestBreakageService(t, env)
	collector := newTestAccrualCollectorWithBreakage(env, breakageService)

	// given:
	// - two same-route expiring credit sources with separate breakage plans
	// when:
	// - one spend charge consumes across both sources
	// then:
	// - accrued preserves source x spend provenance
	// - breakage preserves the remaining broken amount by source only
	sourceCharge1 := testChargeID(1)
	sourceCharge2 := testChargeID(2)
	spendCharge := testChargeID(3)
	firstSourceAmount := int64(10)  // first source expires first, so collection consumes it completely.
	secondSourceAmount := int64(15) // second source is only partially consumed after the first source is exhausted.
	spendAmount := int64(15)        // spend consumes 10 from the first source and 5 from the second source.
	secondSourceConsumed := spendAmount - firstSourceAmount
	secondSourceRemaining := secondSourceAmount - secondSourceConsumed
	bookExpiringCreditWithFeatures(t, env, breakageService, 1, firstSourceAmount, nil, &sourceCharge1, env.Now().Add(10*time.Hour))
	secondPlanID := bookExpiringCreditWithFeatures(t, env, breakageService, 1, secondSourceAmount, nil, &sourceCharge2, env.Now().Add(15*time.Hour))

	allocations, err := collector.collect(t.Context(), collectToAccruedInputForTest(
		env,
		spendCharge,
		alpacadecimal.NewFromInt(spendAmount),
		productcatalog.CreditThenInvoiceSettlementMode,
	))
	require.NoError(t, err)
	require.Len(t, allocations, 1)
	require.Equal(t, float64(spendAmount), allocations[0].Amount.InexactFloat64())

	openPlans, err := breakageService.ListPlans(t.Context(), ledgerbreakage.ListPlansInput{
		CustomerID: env.CustomerID,
		Currency:   env.Currency,
		AsOf:       env.Now(),
	})
	require.NoError(t, err)
	require.Len(t, openPlans, 1)
	require.Equal(t, secondPlanID, openPlans[0].ID.ID)
	require.True(t, openPlans[0].OpenAmount.Equal(alpacadecimal.NewFromInt(secondSourceRemaining)), "open amount: %s", openPlans[0].OpenAmount)

	requireAccruedBalanceBuckets(t, env, map[string]float64{
		sourceSpendChargeKey(&sourceCharge1, &spendCharge): float64(firstSourceAmount),    // first source is fully accrued to this spend.
		sourceSpendChargeKey(&sourceCharge2, &spendCharge): float64(secondSourceConsumed), // only the remainder of the spend hits source 2.
	})
	requireFBOBalanceBuckets(t, env, map[string]float64{})
	requireBreakageBalanceBuckets(t, env, map[string]float64{
		sourceSpendChargeKey(&sourceCharge2, nil): float64(secondSourceRemaining), // source 2 keeps only the unconsumed planned breakage; source 1 is fully released.
	})
}

func TestCollectCustomerFBOBreakageReleaseUsesPlanSourceBeforeBucketCursor(t *testing.T) {
	env := ledgertestutils.NewIntegrationEnv(t, "collector")
	breakageService := newTestBreakageService(t, env)
	collector := newTestAccrualCollectorWithBreakage(env, breakageService)

	// given:
	// - two same-route expiring credit sources sharing one FBO sub-account
	// - the earlier-expiring source has the lexicographically later source charge
	// when:
	// - a spend charge consumes less than the earlier-expiring source amount
	// then:
	// - collection consumes the earlier-expiring source, not the first balance
	//   bucket by source-charge cursor
	sourceCharge1 := testChargeID(1)
	sourceCharge2 := testChargeID(2)
	spendCharge := testChargeID(3)
	laterSourceAmount := int64(10)   // source 1 expires later and should remain untouched.
	earlierSourceAmount := int64(15) // source 2 expires first and should fund this spend.
	spendAmount := int64(5)          // spend only consumes part of source 2.
	earlierSourceRemaining := earlierSourceAmount - spendAmount
	bookExpiringCreditWithFeatures(t, env, breakageService, 1, laterSourceAmount, nil, &sourceCharge1, env.Now().Add(15*time.Hour))
	bookExpiringCreditWithFeatures(t, env, breakageService, 1, earlierSourceAmount, nil, &sourceCharge2, env.Now().Add(10*time.Hour))

	allocations, err := collector.collect(t.Context(), collectToAccruedInputForTest(
		env,
		spendCharge,
		alpacadecimal.NewFromInt(spendAmount),
		productcatalog.CreditThenInvoiceSettlementMode,
	))
	require.NoError(t, err)
	require.Len(t, allocations, 1)
	require.Equal(t, float64(spendAmount), allocations[0].Amount.InexactFloat64()) // 5 = requested spend was fully covered by credit.

	requireAccruedBalanceBuckets(t, env, map[string]float64{
		sourceSpendChargeKey(&sourceCharge2, &spendCharge): float64(spendAmount), // 5 = collection used the earlier-expiring source 2.
	})
	requireBreakageBalanceBuckets(t, env, map[string]float64{
		sourceSpendChargeKey(&sourceCharge1, nil): float64(laterSourceAmount),      // 10 = later source 1 was not consumed.
		sourceSpendChargeKey(&sourceCharge2, nil): float64(earlierSourceRemaining), // 10 = source 2 started at 15 and released 5.
	})
}

func TestCollectToAccruedSplitsAccruedBySourceCharge(t *testing.T) {
	env := ledgertestutils.NewIntegrationEnv(t, "collector")
	collector := newTestAccrualCollector(env)

	sourceCharge1 := testChargeID(1)
	sourceCharge2 := testChargeID(2)
	spendCharge := testChargeID(3)
	fundSourceCharge(t, env, sourceCharge1, 1, 100)
	fundSourceCharge(t, env, sourceCharge2, 1, 50)

	allocations, err := collector.collect(t.Context(), collectToAccruedInputForTest(env, spendCharge, alpacadecimal.NewFromInt(120), productcatalog.CreditThenInvoiceSettlementMode))
	require.NoError(t, err)
	require.Len(t, allocations, 1)
	require.Equal(t, float64(120), allocations[0].Amount.InexactFloat64())

	requireAccruedBalanceBuckets(t, env, map[string]float64{
		sourceSpendChargeKey(&sourceCharge1, &spendCharge): 100,
		sourceSpendChargeKey(&sourceCharge2, &spendCharge): 20,
	})
	requireFBOBalanceBuckets(t, env, map[string]float64{
		sourceSpendChargeKey(&sourceCharge2, nil): 30,
	})
}

func TestCollectToAccruedSplitsAccruedBySpendCharge(t *testing.T) {
	env := ledgertestutils.NewIntegrationEnv(t, "collector")
	collector := newTestAccrualCollector(env)

	sourceCharge := testChargeID(1)
	spendCharge1 := testChargeID(2)
	spendCharge2 := testChargeID(3)
	fundSourceCharge(t, env, sourceCharge, 1, 100)

	firstAllocations, err := collector.collect(t.Context(), collectToAccruedInputForTest(env, spendCharge1, alpacadecimal.NewFromInt(40), productcatalog.CreditThenInvoiceSettlementMode))
	require.NoError(t, err)
	require.Len(t, firstAllocations, 1)

	secondAllocations, err := collector.collect(t.Context(), collectToAccruedInputForTest(env, spendCharge2, alpacadecimal.NewFromInt(30), productcatalog.CreditThenInvoiceSettlementMode))
	require.NoError(t, err)
	require.Len(t, secondAllocations, 1)

	requireAccruedBalanceBuckets(t, env, map[string]float64{
		sourceSpendChargeKey(&sourceCharge, &spendCharge1): 40,
		sourceSpendChargeKey(&sourceCharge, &spendCharge2): 30,
	})
	requireFBOBalanceBuckets(t, env, map[string]float64{
		sourceSpendChargeKey(&sourceCharge, nil): 30,
	})
}

func TestCollectToAccruedAdvanceShortfallStampsSpendCharge(t *testing.T) {
	env := ledgertestutils.NewIntegrationEnv(t, "collector")
	collector := newTestAccrualCollector(env)

	spendCharge := testChargeID(1)
	allocations, err := collector.collect(t.Context(), collectToAccruedInputForTest(env, spendCharge, alpacadecimal.NewFromInt(30), productcatalog.CreditOnlySettlementMode))
	require.NoError(t, err)
	require.Len(t, allocations, 1)
	require.Equal(t, float64(30), allocations[0].Amount.InexactFloat64())

	requireAccruedBalanceBuckets(t, env, map[string]float64{
		sourceSpendChargeKey(nil, &spendCharge): 30,
	})
	requireReceivableBalanceBuckets(t, env, map[string]float64{
		sourceSpendChargeKey(nil, &spendCharge): -30,
	})
}

func TestCollectToAccruedCreditThenInvoiceOnlyCollectsAvailableCredit(t *testing.T) {
	env := ledgertestutils.NewIntegrationEnv(t, "collector")
	collector := newTestAccrualCollector(env)

	sourceCharge := testChargeID(1)
	spendCharge := testChargeID(2)
	fundSourceCharge(t, env, sourceCharge, 1, 40)

	allocations, err := collector.collect(t.Context(), collectToAccruedInputForTest(env, spendCharge, alpacadecimal.NewFromInt(70), productcatalog.CreditThenInvoiceSettlementMode))
	require.NoError(t, err)
	require.Len(t, allocations, 1)
	require.Equal(t, float64(40), allocations[0].Amount.InexactFloat64())

	requireAccruedBalanceBuckets(t, env, map[string]float64{
		sourceSpendChargeKey(&sourceCharge, &spendCharge): 40,
	})
}

func newTestAccrualCollector(env *ledgertestutils.IntegrationEnv) *accrualCollector {
	return &accrualCollector{
		ledger: env.Deps.HistoricalLedger,
		deps: transactions.ResolverDependencies{
			AccountService: env.Deps.ResolversService,
			AccountCatalog: env.Deps.AccountService,
			BalanceQuerier: env.Deps.HistoricalLedger,
		},
		accountLocker:      env.Deps.AccountService,
		transactionManager: enttx.NewCreator(env.DB),
	}
}

func collectCustomerFBOForTest(
	t *testing.T,
	env *ledgertestutils.IntegrationEnv,
	collector *accrualCollector,
	target alpacadecimal.Decimal,
	asOf time.Time,
) ([]transactions.PostingAmount, error) {
	t.Helper()

	return collectCustomerFBOForFeatureForTest(t, env, collector, "", target, asOf)
}

func collectCustomerFBOForFeatureForTest(
	t *testing.T,
	env *ledgertestutils.IntegrationEnv,
	collector *accrualCollector,
	featureKey string,
	target alpacadecimal.Decimal,
	asOf time.Time,
) ([]transactions.PostingAmount, error) {
	t.Helper()

	return transaction.Run(t.Context(), enttx.NewCreator(env.DB), func(ctx context.Context) ([]transactions.PostingAmount, error) {
		selections, err := collector.collectCustomerFBOSelections(ctx, env.CustomerID, env.Currency, featureKey, target, asOf)
		if err != nil {
			return nil, err
		}

		return fboCollectionSelections(selections).postingAmounts(nil), nil
	})
}

func newTestAccrualCollectorWithBreakage(
	env *ledgertestutils.IntegrationEnv,
	breakageService ledgerbreakage.Service,
) *accrualCollector {
	collector := newTestAccrualCollector(env)
	collector.breakage = breakageService
	collector.transactionManager = enttx.NewCreator(env.DB)

	return collector
}

func newTestBreakageService(t *testing.T, env *ledgertestutils.IntegrationEnv) ledgerbreakage.Service {
	t.Helper()

	breakageAdapter, err := ledgerbreakageadapter.New(ledgerbreakageadapter.Config{
		Client: env.DB,
	})
	require.NoError(t, err)

	breakageService, err := ledgerbreakage.NewService(ledgerbreakage.Config{
		Adapter: breakageAdapter,
		Dependencies: transactions.ResolverDependencies{
			AccountService: env.Deps.ResolversService,
			AccountCatalog: env.Deps.AccountService,
			BalanceQuerier: env.Deps.HistoricalLedger,
		},
	})
	require.NoError(t, err)

	return breakageService
}

func fundPriority(t *testing.T, env *ledgertestutils.IntegrationEnv, priority int, amount int64) ledger.SubAccount {
	t.Helper()

	return fundPriorityWithCostBasis(t, env, priority, amount, nil)
}

func fundPriorityWithCostBasis(
	t *testing.T,
	env *ledgertestutils.IntegrationEnv,
	priority int,
	amount int64,
	costBasis *alpacadecimal.Decimal,
) ledger.SubAccount {
	t.Helper()

	return fundPriorityWithCostBasisAndFeatures(t, env, priority, amount, costBasis, nil)
}

func fundPriorityWithFeatures(
	t *testing.T,
	env *ledgertestutils.IntegrationEnv,
	priority int,
	amount int64,
	features []string,
) ledger.SubAccount {
	t.Helper()

	return fundPriorityWithCostBasisAndFeatures(t, env, priority, amount, nil, features)
}

func fundPriorityWithCostBasisAndFeatures(
	t *testing.T,
	env *ledgertestutils.IntegrationEnv,
	priority int,
	amount int64,
	costBasis *alpacadecimal.Decimal,
	features []string,
) ledger.SubAccount {
	t.Helper()

	subAccount, err := env.CustomerAccounts.FBOAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerFBORouteParams{
		Currency:       env.Currency,
		CostBasis:      costBasis,
		CreditPriority: priority,
		Features:       features,
	})
	require.NoError(t, err)

	inputs, err := transactions.ResolveTransactions(
		t.Context(),
		transactions.ResolverDependencies{
			AccountService: env.Deps.ResolversService,
			AccountCatalog: env.Deps.AccountService,
			BalanceQuerier: env.Deps.HistoricalLedger,
		},
		transactions.ResolutionScope{
			CustomerID: env.CustomerID,
			Namespace:  env.Namespace,
		},
		transactions.IssueCustomerReceivableTemplate{
			At:             env.Now(),
			Amount:         alpacadecimal.NewFromInt(amount),
			Currency:       env.Currency,
			CostBasis:      costBasis,
			CreditPriority: &priority,
			Features:       features,
		},
		transactions.AuthorizeCustomerReceivablePaymentTemplate{
			At:        env.Now(),
			Amount:    alpacadecimal.NewFromInt(amount),
			Currency:  env.Currency,
			CostBasis: costBasis,
			Features:  features,
		},
		transactions.SettleCustomerReceivableFromPaymentTemplate{
			At:        env.Now(),
			Amount:    alpacadecimal.NewFromInt(amount),
			Currency:  env.Currency,
			CostBasis: costBasis,
			Features:  features,
		},
	)
	require.NoError(t, err)

	_, err = env.Deps.HistoricalLedger.CommitGroup(t.Context(), transactions.GroupInputs(env.Namespace, nil, inputs...))
	require.NoError(t, err)

	return subAccount
}

func fundSourceCharge(t *testing.T, env *ledgertestutils.IntegrationEnv, sourceChargeID string, priority int, amount int64) ledger.SubAccount {
	t.Helper()

	subAccount, err := env.CustomerAccounts.FBOAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerFBORouteParams{
		Currency:       env.Currency,
		CreditPriority: priority,
	})
	require.NoError(t, err)

	inputs, err := transactions.ResolveTransactions(
		t.Context(),
		transactions.ResolverDependencies{
			AccountService: env.Deps.ResolversService,
			AccountCatalog: env.Deps.AccountService,
			BalanceQuerier: env.Deps.HistoricalLedger,
		},
		transactions.ResolutionScope{
			CustomerID: env.CustomerID,
			Namespace:  env.Namespace,
		},
		transactions.IssueCustomerReceivableTemplate{
			At:             env.Now(),
			Amount:         alpacadecimal.NewFromInt(amount),
			Currency:       env.Currency,
			SourceChargeID: &sourceChargeID,
			CreditPriority: &priority,
		},
		transactions.AuthorizeCustomerReceivablePaymentTemplate{
			At:             env.Now(),
			Amount:         alpacadecimal.NewFromInt(amount),
			Currency:       env.Currency,
			SourceChargeID: &sourceChargeID,
		},
		transactions.SettleCustomerReceivableFromPaymentTemplate{
			At:             env.Now(),
			Amount:         alpacadecimal.NewFromInt(amount),
			Currency:       env.Currency,
			SourceChargeID: &sourceChargeID,
		},
	)
	require.NoError(t, err)

	_, err = env.Deps.HistoricalLedger.CommitGroup(t.Context(), transactions.GroupInputs(env.Namespace, nil, inputs...))
	require.NoError(t, err)

	return subAccount
}

func collectToAccruedInputForTest(
	env *ledgertestutils.IntegrationEnv,
	chargeID string,
	amount alpacadecimal.Decimal,
	settlementMode productcatalog.SettlementMode,
) CollectToAccruedInput {
	return CollectToAccruedInput{
		Namespace:         env.Namespace,
		ChargeID:          chargeID,
		CustomerID:        env.CustomerID.ID,
		BookedAt:          env.Now(),
		SourceBalanceAsOf: env.Now(),
		Currency:          env.Currency,
		SettlementMode:    settlementMode,
		ServicePeriod: timeutil.ClosedPeriod{
			From: env.Now().Add(-time.Hour),
			To:   env.Now(),
		},
		Amount: amount,
	}
}

func requireAccruedBalanceBuckets(t *testing.T, env *ledgertestutils.IntegrationEnv, expected map[string]float64) {
	t.Helper()
	accruedAccountID := env.CustomerAccounts.AccruedAccount.ID().ID

	requireBalanceBuckets(t, env, accruedAccountID, expected)
}

func requireReceivableBalanceBuckets(t *testing.T, env *ledgertestutils.IntegrationEnv, expected map[string]float64) {
	t.Helper()

	requireBalanceBuckets(t, env, env.CustomerAccounts.ReceivableAccount.ID().ID, expected)
}

func requireFBOBalanceBuckets(t *testing.T, env *ledgertestutils.IntegrationEnv, expected map[string]float64) {
	t.Helper()

	requireFBOBalanceBucketsWithAsOf(t, env, nil, expected)
}

func requireFBOBalanceBucketsAt(t *testing.T, env *ledgertestutils.IntegrationEnv, asOf time.Time, expected map[string]float64) {
	t.Helper()

	requireFBOBalanceBucketsWithAsOf(t, env, &asOf, expected)
}

func requireFBOProvenanceBalanceBuckets(t *testing.T, env *ledgertestutils.IntegrationEnv, expected map[string]float64) {
	t.Helper()

	requireBalanceBuckets(t, env, env.CustomerAccounts.FBOAccount.ID().ID, expected)
}

func requireFBOBalanceBucketsWithAsOf(t *testing.T, env *ledgertestutils.IntegrationEnv, asOf *time.Time, expected map[string]float64) {
	t.Helper()

	buckets, err := env.Deps.HistoricalLedger.GetBalanceBuckets(t.Context(), ledger.BalanceBucketQuery{
		Namespace: env.Namespace,
		Filters: ledger.Filters{
			AccountID: lo.ToPtr(env.CustomerAccounts.FBOAccount.ID().ID),
			AsOf:      asOf,
			Route: ledger.RouteFilter{
				Currency: env.Currency,
			},
		},
		GroupBy: []string{ledger.BalanceBucketGroupBySourceChargeID},
	})
	require.NoError(t, err)

	actual := make(map[string]float64, len(buckets))
	for _, bucket := range buckets {
		if bucket.SettledAmount.IsZero() {
			continue
		}
		actual[sourceSpendChargeKey(
			bucket.GroupByValues[ledger.BalanceBucketGroupBySourceChargeID],
			nil,
		)] = bucket.SettledAmount.InexactFloat64()
	}
	require.Equal(t, expected, actual)
}

func requireBreakageBalanceBuckets(t *testing.T, env *ledgertestutils.IntegrationEnv, expected map[string]float64) {
	t.Helper()

	requireBalanceBuckets(t, env, env.BusinessAccounts.BreakageAccount.ID().ID, expected)
}

func requireBalanceBuckets(t *testing.T, env *ledgertestutils.IntegrationEnv, accountID string, expected map[string]float64) {
	t.Helper()

	buckets, err := env.Deps.HistoricalLedger.GetBalanceBuckets(t.Context(), ledger.BalanceBucketQuery{
		Namespace: env.Namespace,
		Filters: ledger.Filters{
			AccountID: &accountID,
			Route: ledger.RouteFilter{
				Currency: env.Currency,
			},
		},
		GroupBy: []string{
			ledger.BalanceBucketGroupBySourceChargeID,
			ledger.BalanceBucketGroupBySpendChargeID,
		},
	})
	require.NoError(t, err)

	actual := make(map[string]float64, len(buckets))
	for _, bucket := range buckets {
		if bucket.SettledAmount.IsZero() {
			continue
		}
		actual[sourceSpendChargeKey(
			bucket.GroupByValues[ledger.BalanceBucketGroupBySourceChargeID],
			bucket.GroupByValues[ledger.BalanceBucketGroupBySpendChargeID],
		)] = bucket.SettledAmount.InexactFloat64()
	}
	require.Equal(t, expected, actual)
}

func sourceSpendChargeKey(sourceChargeID, spendChargeID *string) string {
	return fmt.Sprintf("source=%s spend=%s", lo.FromPtrOr(sourceChargeID, "<nil>"), lo.FromPtrOr(spendChargeID, "<nil>"))
}

func testChargeID(n int) string {
	return fmt.Sprintf("01J%023d", n)
}

func bookExpiringCredit(
	t *testing.T,
	env *ledgertestutils.IntegrationEnv,
	breakageService ledgerbreakage.Service,
	priority int,
	amount int64,
	expiresAt time.Time,
) string {
	t.Helper()

	return bookExpiringCreditWithFeatures(t, env, breakageService, priority, amount, nil, nil, expiresAt)
}

func bookExpiringCreditWithFeatures(
	t *testing.T,
	env *ledgertestutils.IntegrationEnv,
	breakageService ledgerbreakage.Service,
	priority int,
	amount int64,
	features []string,
	sourceChargeID *string,
	expiresAt time.Time,
) string {
	t.Helper()

	creditAmount := alpacadecimal.NewFromInt(amount)
	inputs, err := transactions.ResolveTransactions(
		t.Context(),
		transactions.ResolverDependencies{
			AccountService: env.Deps.ResolversService,
			AccountCatalog: env.Deps.AccountService,
			BalanceQuerier: env.Deps.HistoricalLedger,
		},
		transactions.ResolutionScope{
			CustomerID: env.CustomerID,
			Namespace:  env.Namespace,
		},
		transactions.IssueCustomerReceivableTemplate{
			At:             env.Now(),
			Amount:         creditAmount,
			Currency:       env.Currency,
			SourceChargeID: sourceChargeID,
			CreditPriority: &priority,
			Features:       features,
		},
	)
	require.NoError(t, err)

	breakageInputs, pending, err := breakageService.PlanIssuance(t.Context(), ledgerbreakage.PlanIssuanceInput{
		CustomerID:     env.CustomerID,
		Amount:         creditAmount,
		Currency:       env.Currency,
		CreditPriority: &priority,
		Features:       features,
		ExpiresAt:      expiresAt,
		SourceChargeID: sourceChargeID,
	})
	require.NoError(t, err)
	require.Len(t, pending, 1)

	inputs = append(inputs, breakageInputs...)
	group, err := env.Deps.HistoricalLedger.CommitGroup(t.Context(), transactions.GroupInputs(env.Namespace, nil, inputs...))
	require.NoError(t, err)
	require.NoError(t, breakageService.PersistCommittedRecords(t.Context(), pending, group))

	return pending[0].ID.ID
}

func bookFutureFBOCollection(t *testing.T, env *ledgertestutils.IntegrationEnv, priority int, amount int64, at time.Time) {
	t.Helper()

	inputs, err := transactions.ResolveTransactions(
		t.Context(),
		transactions.ResolverDependencies{
			AccountService: env.Deps.ResolversService,
			AccountCatalog: env.Deps.AccountService,
			BalanceQuerier: env.Deps.HistoricalLedger,
		},
		transactions.ResolutionScope{
			CustomerID: env.CustomerID,
			Namespace:  env.Namespace,
		},
		transactions.CoverCustomerReceivableTemplate{
			At:             at,
			Amount:         alpacadecimal.NewFromInt(amount),
			Currency:       env.Currency,
			CreditPriority: &priority,
		},
	)
	require.NoError(t, err)

	_, err = env.Deps.HistoricalLedger.CommitGroup(t.Context(), transactions.GroupInputs(env.Namespace, nil, inputs...))
	require.NoError(t, err)
}
