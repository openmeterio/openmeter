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
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestCorrectCollectedAccruedUsesReverseFeatureAwareCollectionOrder(t *testing.T) {
	env := ledgertestutils.NewIntegrationEnv(t, "collector-correct")
	collector := newTestAccrualCollector(env)
	corrector := newTestAccrualCorrector(env, nil)

	restricted := fundPriorityWithFeatures(t, env, 1, 30, []string{"api-calls"})
	unrestricted := fundPriorityWithFeatures(t, env, 1, 10, nil)
	servicePeriod := testServicePeriod(env)

	allocations, err := collector.collect(t.Context(), CollectToAccruedInput{
		Namespace:         env.Namespace,
		ChargeID:          "charge-01JABCDEF0123456789ABCDEF",
		CustomerID:        env.CustomerID.ID,
		BookedAt:          env.Now(),
		SourceBalanceAsOf: env.Now(),
		Currency:          env.Currency,
		SettlementMode:    productcatalog.CreditThenInvoiceSettlementMode,
		ServicePeriod:     servicePeriod,
		FeatureKey:        "api-calls",
		Amount:            alpacadecimal.NewFromInt(40),
	})
	require.NoError(t, err)
	require.Len(t, allocations, 2)

	realizations := realizationsFromAllocations(env, allocations)
	currencyCalculator, err := env.Currency.Calculator()
	require.NoError(t, err)

	corrections, err := realizations.CreateCorrectionRequest(alpacadecimal.NewFromInt(-20), currencyCalculator)
	require.NoError(t, err)
	require.Len(t, corrections, 2)

	_, err = corrector.correct(t.Context(), CorrectCollectedAccruedInput{
		Namespace:   env.Namespace,
		ChargeID:    "charge-01JABCDEF0123456789ABCDEF",
		CustomerID:  env.CustomerID.ID,
		AllocateAt:  env.Now(),
		Corrections: corrections,
	})
	require.NoError(t, err)

	require.True(t, env.SumBalance(t, restricted).Equal(alpacadecimal.NewFromInt(10)))
	require.True(t, env.SumBalance(t, unrestricted).Equal(alpacadecimal.NewFromInt(10)))
	require.True(t, env.SumBalance(t, env.AccruedSubAccount(t)).Equal(alpacadecimal.NewFromInt(20)))
}

func TestCorrectCollectedAccruedReopensBreakageByReverseFeatureAwareCollectionOrder(t *testing.T) {
	env := ledgertestutils.NewIntegrationEnv(t, "collector-correct-breakage")
	breakageService := newTestBreakageService(t, env)
	collector := newTestAccrualCollectorWithBreakage(env, breakageService)
	corrector := newTestAccrualCorrector(env, breakageService)

	priority := 1
	restrictedPlanID := bookExpiringCreditWithFeatures(t, env, breakageService, priority, 30, []string{"api-calls"}, env.Now().Add(20*time.Hour))
	unrestrictedPlanID := bookExpiringCredit(t, env, breakageService, priority, 10, env.Now().Add(10*time.Hour))
	servicePeriod := testServicePeriod(env)

	allocations, err := collector.collect(t.Context(), CollectToAccruedInput{
		Namespace:         env.Namespace,
		ChargeID:          "charge-01JABCDEF0123456789ABCDEF",
		CustomerID:        env.CustomerID.ID,
		BookedAt:          env.Now(),
		SourceBalanceAsOf: env.Now(),
		Currency:          env.Currency,
		SettlementMode:    productcatalog.CreditThenInvoiceSettlementMode,
		ServicePeriod:     servicePeriod,
		FeatureKey:        "api-calls",
		Amount:            alpacadecimal.NewFromInt(40),
	})
	require.NoError(t, err)
	require.Len(t, allocations, 2)

	openPlans, err := breakageService.ListPlans(t.Context(), ledgerbreakage.ListPlansInput{
		CustomerID: env.CustomerID,
		Currency:   env.Currency,
		AsOf:       env.Now(),
	})
	require.NoError(t, err)
	require.Empty(t, openPlans)

	realizations := realizationsFromAllocations(env, allocations)
	currencyCalculator, err := env.Currency.Calculator()
	require.NoError(t, err)

	corrections, err := realizations.CreateCorrectionRequest(alpacadecimal.NewFromInt(-10), currencyCalculator)
	require.NoError(t, err)

	_, err = corrector.correct(t.Context(), CorrectCollectedAccruedInput{
		Namespace:   env.Namespace,
		ChargeID:    "charge-01JABCDEF0123456789ABCDEF",
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
	require.Len(t, openPlans, 1)
	require.Equal(t, unrestrictedPlanID, openPlans[0].ID.ID)
	require.NotEqual(t, restrictedPlanID, openPlans[0].ID.ID)
	require.True(t, openPlans[0].OpenAmount.Equal(alpacadecimal.NewFromInt(10)), "open amount: %s", openPlans[0].OpenAmount)
}

func TestCorrectCollectedAccruedPartiallyReversesAdvanceBackedCollection(t *testing.T) {
	env := ledgertestutils.NewIntegrationEnv(t, "collector-correct-advance")
	collector := newTestAccrualCollector(env)
	corrector := newTestAccrualCorrector(env, nil)
	servicePeriod := testServicePeriod(env)

	allocations, err := collector.collect(t.Context(), CollectToAccruedInput{
		Namespace:         env.Namespace,
		ChargeID:          "charge-01JABCDEF0123456789ABCDEF",
		CustomerID:        env.CustomerID.ID,
		BookedAt:          env.Now(),
		SourceBalanceAsOf: env.Now(),
		Currency:          env.Currency,
		SettlementMode:    productcatalog.CreditOnlySettlementMode,
		ServicePeriod:     servicePeriod,
		Amount:            alpacadecimal.NewFromInt(30),
	})
	require.NoError(t, err)
	require.Len(t, allocations, 1)

	realizations := realizationsFromAllocations(env, allocations)
	_, err = corrector.correct(t.Context(), CorrectCollectedAccruedInput{
		Namespace:  env.Namespace,
		ChargeID:   "charge-01JABCDEF0123456789ABCDEF",
		CustomerID: env.CustomerID.ID,
		AllocateAt: env.Now(),
		Corrections: creditrealization.CorrectionRequest{
			{
				Allocation: realizations[0],
				Amount:     alpacadecimal.NewFromInt(-10),
			},
		},
	})
	require.NoError(t, err)

	require.True(t, env.SumBalance(t, env.ReceivableSubAccount(t)).Equal(alpacadecimal.NewFromInt(-20)))
	require.True(t, env.SumBalance(t, env.FBOSubAccount(t, ledger.DefaultCustomerFBOPriority)).Equal(alpacadecimal.Zero))
	require.True(t, env.SumBalance(t, env.AccruedSubAccount(t)).Equal(alpacadecimal.NewFromInt(20)))
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
