package chargeadapter_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/chargeadapter"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func TestAdvanceBackfillInterleavedRunsAndRecognizedCorrection(t *testing.T) {
	env := newUsageBasedHandlerTestEnv(t)
	purchaseEnv := &creditPurchaseHandlerTestEnv{IntegrationEnv: env.IntegrationEnv, currency: env.currency}
	handler, err := chargeadapter.NewCreditPurchaseHandler(env.Deps.HistoricalLedger, env.Deps.HistoricalLedger, env.Deps.ResolversService, env.Deps.AccountService, nil, enttx.NewCreator(env.DB))
	require.NoError(t, err)

	// Given A's two collection occurrences surround B, while B's charge ID sorts first.
	chargeB := env.newCreditsOnlyCharge()
	chargeB.ID = ulid.Make().String()
	chargeA := env.newCreditsOnlyCharge()
	chargeA.ID = ulid.Make().String()
	env.ensureCharge(t, chargeA.ID)
	env.ensureCharge(t, chargeB.ID)
	start := env.Now()
	var runs []usagebased.RealizationRun
	groups := map[string]string{}
	for i, charge := range []usagebased.Charge{chargeA, chargeB, chargeA} {
		clock.FreezeTime(start.Add(time.Duration(i) * time.Hour))
		defer clock.UnFreeze()
		run := env.newRun()
		allocations, err := env.handler.OnCreditsOnlyUsageAccrued(t.Context(), usagebased.CreditsOnlyUsageAccruedInput{
			Charge: charge, Run: run, BookedAt: env.Now(), AmountToAllocate: alpacadecimal.NewFromInt(20),
		})
		require.NoError(t, err)
		run.CreditsAllocated = env.realizationsFromAllocations(allocations)
		require.Len(t, run.CreditsAllocated, 1)
		groups[run.CreditsAllocated[0].ID] = run.CreditsAllocated[0].LedgerTransaction.TransactionGroupID
		require.NoError(t, env.lineage.CreateInitialLineages(t.Context(), lineage.CreateInitialLineagesInput{
			Namespace: env.Namespace, CustomerID: env.CustomerID.ID, ChargeID: charge.ID, Currency: env.currency,
			Features: []string{"api_requests"}, Realizations: run.CreditsAllocated,
		}))
		runs = append(runs, run)
	}

	// When partial purchases arrive, an older remainder keeps its original place.
	for i, scenario := range []struct {
		amount    int64
		costBasis float64
		perRun    []float64
	}{
		{10, 0.5, []float64{10, 0, 0}},
		{15, 0.8, []float64{10, 5, 0}},
		{25, 1.1, []float64{0, 15, 10}},
	} {
		clock.FreezeTime(start.Add(time.Duration(i+3) * time.Hour))
		defer clock.UnFreeze()
		purchase := purchaseEnv.newExternalCharge(alpacadecimal.NewFromInt(scenario.amount), alpacadecimal.NewFromFloat(scenario.costBasis))
		purchase.ID = ulid.Make().String()
		result, err := transaction.Run(t.Context(), enttx.NewCreator(env.DB), func(ctx context.Context) (creditpurchase.CreditGrantResult, error) {
			roots, err := env.lineage.LoadLineagesByCustomer(ctx, lineage.LoadLineagesByCustomerInput{Namespace: env.Namespace, CustomerID: env.CustomerID.ID, Currency: env.currency.Reference()})
			if err != nil {
				return creditpurchase.CreditGrantResult{}, err
			}
			// These handler fixtures persist real ledger groups and lineage; the
			// charge service normally hydrates this link from its allocation rows.
			for i := range roots {
				roots[i].OriginalTransactionGroupID = groups[roots[i].RootRealizationID]
			}
			// Deliberately pass newest-first: FIFO must be owned by the ledger.
			slices.Reverse(roots)
			inputOrder := lo.Map(roots, func(root lineage.Lineage, _ int) string { return root.ID })
			result, err := handler.OnCreditPurchaseInitiated(ctx, creditpurchase.CreditGrantInput{Charge: purchase, AdvanceLineages: roots})
			require.Equal(t, inputOrder, lo.Map(roots, func(root lineage.Lineage, _ int) string { return root.ID }), "caller order must remain unchanged")
			if err != nil {
				return result, err
			}
			err = env.lineage.BackfillAdvanceLineageSegments(ctx, lineage.BackfillAdvanceLineageSegmentsInput{
				Namespace: env.Namespace, CustomerID: env.CustomerID.ID, Currency: env.currency,
				Amount: purchase.Intent.CreditAmount, BackingTransactionGroupID: result.TransactionGroupID,
				Allocations: result.BackfillAllocations,
			})
			return result, err
		})
		require.NoError(t, err)

		// Then both persisted representations carry the same occurrence allocation.
		for i, run := range runs {
			segments := env.activeSegmentsByRealization(t, run.CreditsAllocated)[run.CreditsAllocated[0].ID]
			var backed float64
			for _, segment := range segments {
				if lo.FromPtr(segment.BackingTransactionGroupID) == result.TransactionGroupID {
					backed += segment.Amount.InexactFloat64()
				}
			}
			require.Equal(t, scenario.perRun[i], backed, "run %d", i)
		}
		buckets, err := env.Deps.HistoricalLedger.GetBalanceBuckets(t.Context(), ledger.BalanceBucketQuery{
			Namespace: env.Namespace,
			Filters:   ledger.Filters{AccountID: lo.ToPtr(env.CustomerAccounts.AccruedAccount.ID().ID), SourceChargeID: mo.Some(&purchase.ID)},
			GroupBy:   []string{ledger.BalanceBucketGroupBySpendChargeID},
		})
		require.NoError(t, err)
		booked := map[string]float64{}
		for _, bucket := range buckets {
			booked[lo.FromPtr(bucket.GroupByValues[ledger.BalanceBucketGroupBySpendChargeID])] += bucket.SettledAmount.InexactFloat64()
		}
		require.Equal(t, scenario.perRun[0]+scenario.perRun[2], booked[chargeA.ID])
		require.Equal(t, scenario.perRun[1], booked[chargeB.ID])
	}

	// When all purchased backing is recognized, then B is corrected by 7 and the remaining 13,
	// A's recognized 30 remains and only B's actual 20 returns to purchased FBO.
	env.recognizeCreditAccrued(t, alpacadecimal.NewFromInt(50))
	for _, amount := range []int64{7, 13} {
		request, err := runs[1].CreditsAllocated.CreateCorrectionRequest(alpacadecimal.NewFromInt(-amount), env.currency)
		require.NoError(t, err)
		corrections, err := env.handler.OnCreditsOnlyUsageAccruedCorrection(t.Context(), usagebased.CreditsOnlyUsageAccruedCorrectionInput{
			Charge: chargeB, Run: runs[1], BookedAt: env.Now(), Corrections: request,
			LineageSegmentsByRealization: env.activeSegmentsByRealization(t, runs[1].CreditsAllocated),
		})
		require.NoError(t, err)
		correctionInputs, err := corrections.AsCreateInputs(runs[1].CreditsAllocated)
		require.NoError(t, err)
		var corrected creditrealization.Realizations
		for _, correction := range correctionInputs {
			corrected = append(corrected, creditrealization.Realization{CreateInput: correction})
		}
		require.NoError(t, env.lineage.PersistCorrectionLineageSegments(t.Context(), lineage.PersistCorrectionLineageSegmentsInput{Namespace: env.Namespace, Realizations: corrected}))
		require.Len(t, corrections, 1)
	}
	require.Empty(t, env.activeSegmentsByRealization(t, runs[1].CreditsAllocated)[runs[1].CreditsAllocated[0].ID])
	for accountType, expected := range map[ledger.AccountType]map[string]float64{
		ledger.AccountTypeEarnings:        {chargeA.ID: 30},
		ledger.AccountTypeCustomerAccrued: {chargeA.ID: 10},
	} {
		accountID := env.BusinessAccounts.EarningsAccount.ID().ID
		if accountType == ledger.AccountTypeCustomerAccrued {
			accountID = env.CustomerAccounts.AccruedAccount.ID().ID
		}
		buckets, err := env.Deps.HistoricalLedger.GetBalanceBuckets(t.Context(), ledger.BalanceBucketQuery{
			Namespace: env.Namespace, Filters: ledger.Filters{AccountID: &accountID}, GroupBy: []string{ledger.BalanceBucketGroupBySpendChargeID},
		})
		require.NoError(t, err)
		actual := map[string]float64{}
		for _, bucket := range buckets {
			if !bucket.SettledAmount.IsZero() {
				actual[lo.FromPtr(bucket.GroupByValues[ledger.BalanceBucketGroupBySpendChargeID])] += bucket.SettledAmount.InexactFloat64()
			}
		}
		require.Equal(t, expected, actual)
	}
	require.Equal(t, float64(5), env.SumBalance(t, purchaseEnv.fboSubAccount(t, alpacadecimal.NewFromFloat(0.8))).InexactFloat64())
	require.Equal(t, float64(15), env.SumBalance(t, purchaseEnv.fboSubAccount(t, alpacadecimal.NewFromFloat(1.1))).InexactFloat64())
}
