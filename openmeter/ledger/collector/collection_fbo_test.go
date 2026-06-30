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
	"github.com/openmeterio/openmeter/pkg/models"
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
	bookExpiringCreditWithFeatures(t, env, breakageService, 1, 10, nil, expiresAt)
	bookExpiringCreditWithFeatures(t, env, breakageService, 1, 30, []string{"api-calls"}, expiresAt)
	bookExpiringCreditWithFeatures(t, env, breakageService, 1, 40, []string{"storage"}, expiresAt)

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
	requireCollectedGroupEntries(t, env, allocations[0].LedgerTransaction.TransactionGroupID, []expectedCollectedEntry{
		{accountType: ledger.AccountTypeCustomerFBO, amount: -100, collectionSource: lo.ToPtr("0"), sourceChargeID: &sourceCharge1, spendChargeID: &spendCharge},
		{accountType: ledger.AccountTypeCustomerFBO, amount: -20, collectionSource: lo.ToPtr("1"), sourceChargeID: &sourceCharge2, spendChargeID: &spendCharge},
		{accountType: ledger.AccountTypeCustomerAccrued, amount: 100, sourceChargeID: &sourceCharge1, spendChargeID: &spendCharge},
		{accountType: ledger.AccountTypeCustomerAccrued, amount: 20, sourceChargeID: &sourceCharge2, spendChargeID: &spendCharge},
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
	requireCollectedGroupEntries(t, env, firstAllocations[0].LedgerTransaction.TransactionGroupID, []expectedCollectedEntry{
		{accountType: ledger.AccountTypeCustomerFBO, amount: -40, collectionSource: lo.ToPtr("0"), sourceChargeID: &sourceCharge, spendChargeID: &spendCharge1},
		{accountType: ledger.AccountTypeCustomerAccrued, amount: 40, sourceChargeID: &sourceCharge, spendChargeID: &spendCharge1},
	})
	requireCollectedGroupEntries(t, env, secondAllocations[0].LedgerTransaction.TransactionGroupID, []expectedCollectedEntry{
		{accountType: ledger.AccountTypeCustomerFBO, amount: -30, collectionSource: lo.ToPtr("0"), sourceChargeID: &sourceCharge, spendChargeID: &spendCharge2},
		{accountType: ledger.AccountTypeCustomerAccrued, amount: 30, sourceChargeID: &sourceCharge, spendChargeID: &spendCharge2},
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
	requireCollectedGroupEntries(t, env, allocations[0].LedgerTransaction.TransactionGroupID, []expectedCollectedEntry{
		{accountType: ledger.AccountTypeCustomerFBO, amount: 30, spendChargeID: &spendCharge},
		{accountType: ledger.AccountTypeCustomerReceivable, amount: -30, spendChargeID: &spendCharge},
		{accountType: ledger.AccountTypeCustomerFBO, amount: -30, spendChargeID: &spendCharge},
		{accountType: ledger.AccountTypeCustomerAccrued, amount: 30, spendChargeID: &spendCharge},
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
	requireCollectedGroupEntries(t, env, allocations[0].LedgerTransaction.TransactionGroupID, []expectedCollectedEntry{
		{accountType: ledger.AccountTypeCustomerFBO, amount: -40, collectionSource: lo.ToPtr("0"), sourceChargeID: &sourceCharge, spendChargeID: &spendCharge},
		{accountType: ledger.AccountTypeCustomerAccrued, amount: 40, sourceChargeID: &sourceCharge, spendChargeID: &spendCharge},
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
		return collector.collectCustomerFBO(ctx, env.CustomerID, env.Currency, featureKey, target, asOf)
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

	accruedAccount, ok := env.CustomerAccounts.AccruedAccount.(accountIdentifier)
	require.True(t, ok)
	accruedAccountID := accruedAccount.ID().ID

	requireBalanceBuckets(t, env, accruedAccountID, expected)
}

func requireFBOBalanceBuckets(t *testing.T, env *ledgertestutils.IntegrationEnv, expected map[string]float64) {
	t.Helper()

	fboAccount, ok := env.CustomerAccounts.FBOAccount.(accountIdentifier)
	require.True(t, ok)
	fboAccountID := fboAccount.ID().ID

	buckets, err := env.Deps.HistoricalLedger.GetBalanceBuckets(t.Context(), ledger.BalanceBucketQuery{
		Namespace: env.Namespace,
		Filters: ledger.Filters{
			AccountID: &fboAccountID,
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
		actual[sourceSpendChargeKey(
			bucket.GroupByValues[ledger.BalanceBucketGroupBySourceChargeID],
			bucket.GroupByValues[ledger.BalanceBucketGroupBySpendChargeID],
		)] = bucket.SettledAmount.InexactFloat64()
	}
	require.Equal(t, expected, actual)
}

type expectedCollectedEntry struct {
	accountType      ledger.AccountType
	amount           float64
	collectionSource *string
	sourceChargeID   *string
	spendChargeID    *string
}

func requireCollectedGroupEntries(t *testing.T, env *ledgertestutils.IntegrationEnv, transactionGroupID string, expected []expectedCollectedEntry) {
	t.Helper()

	group, err := env.Deps.HistoricalLedger.GetTransactionGroup(t.Context(), models.NamespacedID{
		Namespace: env.Namespace,
		ID:        transactionGroupID,
	})
	require.NoError(t, err)

	actual := make(map[string]int)
	for _, tx := range group.Transactions() {
		for _, entry := range tx.Entries() {
			_, identityParts, err := ledger.EntryIdentityKeyText(entry.IdentityKey()).Parse()
			require.NoError(t, err)

			key := collectedEntryKey(
				entry.PostingAddress().AccountType(),
				entry.Amount().InexactFloat64(),
				identityParts.CollectionSource,
				entry.SourceChargeID(),
				entry.SpendChargeID(),
			)
			actual[key]++
		}
	}

	expectedByKey := make(map[string]int, len(expected))
	for _, entry := range expected {
		key := collectedEntryKey(entry.accountType, entry.amount, entry.collectionSource, entry.sourceChargeID, entry.spendChargeID)
		expectedByKey[key]++
	}

	require.Equal(t, expectedByKey, actual)
}

func collectedEntryKey(accountType ledger.AccountType, amount float64, collectionSource, sourceChargeID, spendChargeID *string) string {
	return fmt.Sprintf(
		"account=%s amount=%g collection_source=%s %s",
		accountType,
		amount,
		chargeIDKeyPart(collectionSource),
		sourceSpendChargeKey(sourceChargeID, spendChargeID),
	)
}

func sourceSpendChargeKey(sourceChargeID, spendChargeID *string) string {
	return fmt.Sprintf("source=%s spend=%s", chargeIDKeyPart(sourceChargeID), chargeIDKeyPart(spendChargeID))
}

func chargeIDKeyPart(chargeID *string) string {
	if chargeID == nil {
		return "<nil>"
	}

	return *chargeID
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

	return bookExpiringCreditWithFeatures(t, env, breakageService, priority, amount, nil, expiresAt)
}

func bookExpiringCreditWithFeatures(
	t *testing.T,
	env *ledgertestutils.IntegrationEnv,
	breakageService ledgerbreakage.Service,
	priority int,
	amount int64,
	features []string,
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
