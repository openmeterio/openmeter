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
	lineageadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/lineage/adapter"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/chargeadapter"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
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

		// Then each original occurrence retains its exact journal backing.
		require.Empty(t, result.BackfillAllocations, "new origins do not allocate legacy segments")
		for i, run := range runs {
			require.Empty(t, env.activeSegmentsByRealization(t, run.CreditsAllocated))
			group, err := env.Deps.HistoricalLedger.GetTransactionGroup(t.Context(), models.NamespacedID{Namespace: env.Namespace, ID: groups[run.CreditsAllocated[0].ID]})
			require.NoError(t, err)
			origin := group.Transactions()[0].Entries()[0].OriginID()
			require.NotNil(t, origin)
			buckets, err := env.Deps.HistoricalLedger.GetBalanceBuckets(t.Context(), ledger.BalanceBucketQuery{
				Namespace: env.Namespace, Filters: ledger.Filters{AccountID: lo.ToPtr(env.CustomerAccounts.AccruedAccount.ID().ID), SourceChargeID: mo.Some(&purchase.ID), OriginID: mo.Some(origin)},
			})
			require.NoError(t, err)
			var backed float64
			for _, bucket := range buckets {
				backed += bucket.SettledAmount.InexactFloat64()
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

func TestPartialBackfillKeepsOlderAdvanceAheadOfNewerAdvance(t *testing.T) {
	env := newCreditPurchaseHandlerTestEnv(t)
	ctx := t.Context()
	start := env.Now()
	adapter, err := lineageadapter.New(lineageadapter.Config{Client: env.DB})
	require.NoError(t, err)

	// Given A=20 was collected before B=20.
	chargeA, chargeB := ulid.Make().String(), ulid.Make().String()
	env.createAdvanceExposureForSpend(t, alpacadecimal.NewFromInt(20), nil, &chargeA)
	clock.FreezeTime(start.Add(time.Hour))
	defer clock.UnFreeze()
	env.createAdvanceExposureForSpend(t, alpacadecimal.NewFromInt(20), nil, &chargeB)
	roots, err := env.lineage.LoadLineagesByCustomer(ctx, lineage.LoadLineagesByCustomerInput{
		Namespace: env.Namespace, CustomerID: env.CustomerID.ID, Currency: env.currency.Reference(),
	})
	require.NoError(t, err)
	require.Len(t, roots, 2)
	require.Equal(t, chargeA, roots[0].ChargeID)
	require.Equal(t, chargeB, roots[1].ChargeID)

	// When a purchase backs 10 of A, its uncovered remainder becomes a new row,
	// created after B's original segment. Use the real grant and persistence path.
	clock.FreezeTime(start.Add(2 * time.Hour))
	defer clock.UnFreeze()
	costBasis := alpacadecimal.NewFromFloat(0.5)
	firstPurchase := env.newExternalCharge(alpacadecimal.NewFromInt(10), costBasis)
	firstPurchase.ID = ulid.Make().String()
	_, err = env.grantCredits(t, firstPurchase)
	require.NoError(t, err)

	// Then A's remaining 10 still precedes B's 20, even with reversed input IDs.
	uncovered := creditrealization.LineageSegmentStateAdvanceUncovered
	segments, err := adapter.ListActiveSegments(ctx, lineage.ListActiveSegmentsInput{
		LineageIDs: []string{roots[1].ID, roots[0].ID}, State: &uncovered,
	})
	require.NoError(t, err)
	require.Len(t, segments, 2)
	olderRemainder, err := env.DB.CreditRealizationLineageSegment.Get(ctx, segments[0].ID)
	require.NoError(t, err)
	newerOriginal, err := env.DB.CreditRealizationLineageSegment.Get(ctx, segments[1].ID)
	require.NoError(t, err)
	require.True(t, olderRemainder.CreatedAt.After(newerOriginal.CreatedAt), "A remainder is newer than B, but stays first")
	require.Equal(t, roots[0].ID, segments[0].LineageID)
	require.Equal(t, float64(10), segments[0].Amount.InexactFloat64())
	require.Equal(t, roots[1].ID, segments[1].LineageID)
	require.Equal(t, float64(20), segments[1].Amount.InexactFloat64())

	// The next purchase of 15 books A=10 and B=5 in both representations.
	clock.FreezeTime(start.Add(3 * time.Hour))
	defer clock.UnFreeze()
	secondPurchase := env.newExternalCharge(alpacadecimal.NewFromInt(15), costBasis)
	secondPurchase.ID = ulid.Make().String()
	result, err := env.grantCredits(t, secondPurchase)
	require.NoError(t, err)
	require.Len(t, result.BackfillAllocations, 2)
	require.Equal(t, segments[0].ID, result.BackfillAllocations[0].SegmentID)
	require.Equal(t, float64(10), result.BackfillAllocations[0].Amount.InexactFloat64())
	require.Equal(t, segments[1].ID, result.BackfillAllocations[1].SegmentID)
	require.Equal(t, float64(5), result.BackfillAllocations[1].Amount.InexactFloat64())
	env.requireAccountSourceSpendBucketAmounts(t, env.accruedSubAccount(t, costBasis).AccountID().ID, map[string]float64{
		sourceSpendChargeKey(&firstPurchase.ID, &chargeA):  10,
		sourceSpendChargeKey(&secondPurchase.ID, &chargeA): 10,
		sourceSpendChargeKey(&secondPurchase.ID, &chargeB): 5,
		sourceSpendChargeKey(nil, &chargeB):                15,
	})
}

func TestAdvanceBackfillSortsSuppliedRootsByCollectionTimeThenID(t *testing.T) {
	for _, sameTime := range []bool{false, true} {
		name := "collection time wins over ID"
		if sameTime {
			name = "ID breaks equal collection times"
		}
		t.Run(name, func(t *testing.T) {
			env := newCreditPurchaseHandlerTestEnv(t)
			ctx := t.Context()
			start := env.Now()
			chargeA, chargeB := ulid.Make().String(), ulid.Make().String()
			bTime := start.Add(time.Hour)
			if sameTime {
				bTime = start
			}

			// Given B has the smaller root ID, but A has the earlier collection
			// timestamp unless the two timestamps tie.
			clock.FreezeTime(bTime)
			defer clock.UnFreeze()
			env.createAdvanceExposureForSpend(t, alpacadecimal.NewFromInt(20), nil, &chargeB)
			clock.FreezeTime(start)
			defer clock.UnFreeze()
			env.createAdvanceExposureForSpend(t, alpacadecimal.NewFromInt(20), nil, &chargeA)
			roots, err := env.lineage.LoadLineagesByCustomer(ctx, lineage.LoadLineagesByCustomerInput{
				Namespace: env.Namespace, CustomerID: env.CustomerID.ID, Currency: env.currency.Reference(),
			})
			require.NoError(t, err)
			require.Len(t, roots, 2)
			byCharge := lo.KeyBy(roots, func(root lineage.Lineage) string { return root.ChargeID })
			require.Less(t, byCharge[chargeB].ID, byCharge[chargeA].ID)
			require.True(t, byCharge[chargeA].CreatedAt.Equal(start))
			require.True(t, byCharge[chargeB].CreatedAt.Equal(bTime))
			for i := range roots {
				roots[i].OriginalTransactionGroupID = env.originalAdvanceGroups[roots[i].RootRealizationID]
			}
			slices.Reverse(roots)
			originalOrder := lo.Map(roots, func(root lineage.Lineage, _ int) string { return root.ID })

			// When the ledger receives reverse-ordered roots for a purchase of 25.
			clock.FreezeTime(start.Add(2 * time.Hour))
			defer clock.UnFreeze()
			costBasis := alpacadecimal.NewFromFloat(0.5)
			purchase := env.newExternalCharge(alpacadecimal.NewFromInt(25), costBasis)
			result, err := transaction.Run(ctx, enttx.NewCreator(env.DB), func(ctx context.Context) (creditpurchase.CreditGrantResult, error) {
				result, err := env.handler.OnCreditPurchaseInitiated(ctx, creditpurchase.CreditGrantInput{Charge: purchase, AdvanceLineages: roots})
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

			// Then the canonical first occurrence receives 20 and the other 5,
			// regardless of caller order. Equal timestamps use the smaller root ID.
			first, second := chargeA, chargeB
			if sameTime {
				first, second = chargeB, chargeA
			}
			require.Len(t, result.BackfillAllocations, 2)
			require.Equal(t, byCharge[first].Segments[0].ID, result.BackfillAllocations[0].SegmentID)
			require.Equal(t, float64(20), result.BackfillAllocations[0].Amount.InexactFloat64())
			require.Equal(t, byCharge[second].Segments[0].ID, result.BackfillAllocations[1].SegmentID)
			require.Equal(t, float64(5), result.BackfillAllocations[1].Amount.InexactFloat64())
			require.Equal(t, originalOrder, lo.Map(roots, func(root lineage.Lineage, _ int) string { return root.ID }))
			env.requireAccountSourceSpendBucketAmounts(t, env.CustomerAccounts.AccruedAccount.ID().ID, map[string]float64{
				sourceSpendChargeKey(&purchase.ID, &first):  20,
				sourceSpendChargeKey(&purchase.ID, &second): 5,
				sourceSpendChargeKey(nil, &second):          15,
			})
		})
	}
}

func TestAdvanceBackfillStaleSelectionRollsBackPurchase(t *testing.T) {
	env := newCreditPurchaseHandlerTestEnv(t)
	ctx := t.Context()

	// Given both purchases read the same uncovered 20, then the first backs 10.
	spendID := ulid.Make().String()
	env.createAdvanceExposureForSpend(t, alpacadecimal.NewFromInt(20), nil, &spendID)
	staleRoots, err := env.lineage.LoadLineagesByCustomer(ctx, lineage.LoadLineagesByCustomerInput{
		Namespace: env.Namespace, CustomerID: env.CustomerID.ID, Currency: env.currency.Reference(),
	})
	require.NoError(t, err)
	require.Len(t, staleRoots, 1)
	staleRoots[0].OriginalTransactionGroupID = env.originalAdvanceGroups[staleRoots[0].RootRealizationID]
	costBasis := alpacadecimal.NewFromFloat(0.5)
	first := env.newExternalCharge(alpacadecimal.NewFromInt(10), costBasis)
	first.ID = ulid.Make().String()
	_, err = env.grantCredits(t, first)
	require.NoError(t, err)

	// When the second books 5 against its stale selection, persistence rejects
	// the closed segment rather than assigning that money to a different row.
	second := env.newExternalCharge(alpacadecimal.NewFromInt(5), costBasis)
	second.ID = ulid.Make().String()
	var rejectedGroupID string
	err = transaction.RunWithNoValue(ctx, enttx.NewCreator(env.DB), func(ctx context.Context) error {
		result, err := env.handler.OnCreditPurchaseInitiated(ctx, creditpurchase.CreditGrantInput{Charge: second, AdvanceLineages: staleRoots})
		if err != nil {
			return err
		}
		rejectedGroupID = result.TransactionGroupID
		return env.lineage.BackfillAdvanceLineageSegments(ctx, lineage.BackfillAdvanceLineageSegmentsInput{
			Namespace: env.Namespace, CustomerID: env.CustomerID.ID, Currency: env.currency,
			Amount: second.Intent.CreditAmount, BackingTransactionGroupID: result.TransactionGroupID,
			Allocations: result.BackfillAllocations,
		})
	})
	require.ErrorContains(t, err, "backfill allocation exceeds active eligible segment")

	// Then the rejected ledger group is gone and only the first purchase's 10
	// remains backed. Retrying with fresh lineage can safely back another 5.
	require.NotEmpty(t, rejectedGroupID)
	_, err = env.DB.LedgerTransactionGroup.Get(ctx, rejectedGroupID)
	require.True(t, entdb.IsNotFound(err), "rejected purchase group must roll back: %v", err)
	env.requireAccountSourceSpendBucketAmounts(t, env.CustomerAccounts.AccruedAccount.ID().ID, map[string]float64{
		sourceSpendChargeKey(&first.ID, &spendID): 10,
		sourceSpendChargeKey(nil, &spendID):       10,
	})
	_, err = env.grantCredits(t, second)
	require.NoError(t, err)
	env.requireAccountSourceSpendBucketAmounts(t, env.CustomerAccounts.AccruedAccount.ID().ID, map[string]float64{
		sourceSpendChargeKey(&first.ID, &spendID):  10,
		sourceSpendChargeKey(&second.ID, &spendID): 5,
		sourceSpendChargeKey(nil, &spendID):        5,
	})
}

func TestCreditPurchaseAttributesReceivableWithoutLineage(t *testing.T) {
	for _, scenario := range []struct {
		name  string
		roots []lineage.Lineage
	}{
		{name: "nil"},
		{name: "empty", roots: []lineage.Lineage{}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// Given 40 of journal-only receivable, with no accrued or lineage.
			env := newCreditPurchaseHandlerTestEnv(t)
			env.createReceivableOnlyExposure(t, advanceExposureInput{Currency: env.currency, Amount: alpacadecimal.NewFromInt(40)})
			costBasis := alpacadecimal.NewFromFloat(0.5)
			purchase := env.newExternalCharge(alpacadecimal.NewFromInt(25), costBasis)

			// When 25 of credit is purchased, nil and empty roots behave identically.
			result, err := env.handler.OnCreditPurchaseInitiated(t.Context(), creditpurchase.CreditGrantInput{
				Charge: purchase, AdvanceLineages: scenario.roots,
			})
			require.NoError(t, err)

			// Then 25 is attributed and 15 remains; no accrued backing is invented.
			require.Empty(t, result.BackfillAllocations)
			require.Equal(t, float64(-15), env.sumBalance(t, env.unknownReceivableSubAccount(t)).InexactFloat64())
			require.Equal(t, float64(-25), env.sumBalance(t, env.receivableSubAccount(t, costBasis)).InexactFloat64())
			require.Equal(t, float64(0), env.sumBalance(t, env.accruedSubAccount(t, costBasis)).InexactFloat64())
			require.Equal(t, float64(0), env.sumBalance(t, env.fboSubAccount(t, costBasis)).InexactFloat64())
			require.Equal(t, []string{transactions.TemplateCode(transactions.AttributeCustomerAdvanceReceivableCostBasisTemplate{})}, env.transactionTemplateCodes(t, result.TransactionGroupID))
		})
	}
}

func TestCreditPurchaseAttributesRemainingReceivableAfterAccruedBackfill(t *testing.T) {
	env := newCreditPurchaseHandlerTestEnv(t)
	spendChargeID := ulid.Make().String()
	taxCode := ulid.Make().String()
	// Given 20 of receivable but only 5 of accrued with uncovered lineage.
	// The additional receivable is a standalone journal posting, not a normal collection.
	env.createAdvance(t, advanceExposureInput{
		Currency: env.currency, Amount: alpacadecimal.NewFromInt(5), SpendChargeID: &spendChargeID, TaxCode: &taxCode,
	})
	env.createReceivableOnlyExposure(t, advanceExposureInput{
		Currency: env.currency, Amount: alpacadecimal.NewFromInt(15), SpendChargeID: &spendChargeID,
	})

	// When a purchase of 15 arrives, 5 backs accrued and 10 only attributes receivable.
	firstCostBasis := alpacadecimal.NewFromFloat(0.5)
	firstPurchase := env.newExternalCharge(alpacadecimal.NewFromInt(15), firstCostBasis)
	firstPurchase.ID = ulid.Make().String()
	first, err := env.grantCredits(t, firstPurchase)
	require.NoError(t, err)
	require.Len(t, first.BackfillAllocations, 1)
	require.Equal(t, float64(5), first.BackfillAllocations[0].Amount.InexactFloat64())
	require.Equal(t, float64(-5), env.sumBalance(t, env.unknownReceivableSubAccount(t)).InexactFloat64())
	require.Equal(t, float64(-15), env.sumBalance(t, env.receivableSubAccount(t, firstCostBasis)).InexactFloat64())
	require.Equal(t, float64(5), env.sumBalance(t, env.AccruedSubAccountWithCostBasisAndTaxCode(t, &firstCostBasis, &taxCode)).InexactFloat64())
	require.Equal(t, float64(0), env.sumBalance(t, env.accruedSubAccount(t, firstCostBasis)).InexactFloat64())
	require.Equal(t, float64(0), env.sumBalance(t, env.fboSubAccount(t, firstCostBasis)).InexactFloat64())
	// Tax stays on accrued; both receivable portions share one attribution leg.
	require.ElementsMatch(t, []string{
		transactions.TemplateCode(transactions.AttributeCustomerAdvanceReceivableCostBasisTemplate{}),
		transactions.TemplateCode(transactions.TranslateCustomerAccruedCostBasisTemplate{}),
	}, env.transactionTemplateCodes(t, first.TransactionGroupID))

	// Then a second purchase of 10 attributes only the remaining 5 and issues 5 into FBO.
	secondCostBasis := alpacadecimal.NewFromFloat(0.8)
	secondPurchase := env.newExternalCharge(alpacadecimal.NewFromInt(10), secondCostBasis)
	secondPurchase.ID = ulid.Make().String()
	second, err := env.grantCredits(t, secondPurchase)
	require.NoError(t, err)
	require.Empty(t, second.BackfillAllocations)
	require.Equal(t, float64(0), env.sumBalance(t, env.unknownReceivableSubAccount(t)).InexactFloat64())
	require.Equal(t, float64(5), env.sumBalance(t, env.fboSubAccount(t, secondCostBasis)).InexactFloat64())
	require.Equal(t, float64(0), env.sumBalance(t, env.accruedSubAccount(t, secondCostBasis)).InexactFloat64())
	env.requireAccountSourceSpendBucketAmounts(t, env.receivableSubAccount(t, secondCostBasis).AccountID().ID, map[string]float64{
		sourceSpendChargeKey(&firstPurchase.ID, &spendChargeID):  -15,
		sourceSpendChargeKey(&secondPurchase.ID, &spendChargeID): -5,
		sourceSpendChargeKey(&secondPurchase.ID, nil):            -5,
	})
	roots, err := env.lineage.LoadLineagesByCustomer(t.Context(), lineage.LoadLineagesByCustomerInput{
		Namespace: env.Namespace, CustomerID: env.CustomerID.ID, Currency: env.currency.Reference(),
	})
	require.NoError(t, err)
	require.Len(t, roots, 1)
	require.Len(t, roots[0].Segments, 1)
	segment := roots[0].Segments[0]
	require.Equal(t, creditrealization.LineageSegmentStateAdvanceBackfilled, segment.State)
	require.Equal(t, float64(5), segment.Amount.InexactFloat64())
	require.NotNil(t, segment.BackingTransactionGroupID)
	require.Equal(t, first.TransactionGroupID, *segment.BackingTransactionGroupID)
}

func TestCreditPurchaseReceivableOnlyAttributionPreservesLegacyFeatureRoutes(t *testing.T) {
	env := newCreditPurchaseHandlerTestEnv(t)
	// Given nil-spend receivable of 20 for API, 30 for storage, and 10 unrestricted.
	for _, exposure := range []struct {
		amount   int64
		features []string
	}{
		{20, []string{"api-calls"}},
		{30, []string{"storage"}},
		{10, nil},
	} {
		env.createReceivableOnlyExposure(t, advanceExposureInput{
			Currency: env.currency, Amount: alpacadecimal.NewFromInt(exposure.amount), Features: exposure.features,
		})
	}

	// When 25 of API-restricted credit arrives without any accrued or lineage.
	costBasis := alpacadecimal.NewFromFloat(0.5)
	purchase := env.newExternalCharge(alpacadecimal.NewFromInt(25), costBasis)
	purchase.Intent.FeatureFilters = creditpurchase.FeatureFilters{"api-calls"}
	result, err := env.grantCredits(t, purchase)
	require.NoError(t, err)

	// Then only the API receivable is attributed; the excess 5 becomes restricted FBO.
	require.Empty(t, result.BackfillAllocations)
	require.Equal(t, float64(0), env.sumBalance(t, env.unknownReceivableSubAccountWithFeatures(t, []string{"api-calls"})).InexactFloat64())
	require.Equal(t, float64(-30), env.sumBalance(t, env.unknownReceivableSubAccountWithFeatures(t, []string{"storage"})).InexactFloat64())
	require.Equal(t, float64(-10), env.sumBalance(t, env.unknownReceivableSubAccount(t)).InexactFloat64())
	require.Equal(t, float64(-25), env.sumBalance(t, env.receivableSubAccountWithFeatures(t, costBasis, []string{"api-calls"})).InexactFloat64())
	require.Equal(t, float64(5), env.sumBalance(t, env.fboSubAccountWithFeatures(t, costBasis, []string{"api-calls"})).InexactFloat64())
}

// Standalone receivable postings exercise the historical receivable-only path
// without inventing an accrued collection or lineage for the unmatched amount.
func (e *creditPurchaseHandlerTestEnv) createReceivableOnlyExposure(t *testing.T, input advanceExposureInput) {
	t.Helper()
	inputs, err := transactions.ResolveTransactions(t.Context(), transactions.ResolverDependencies{
		AccountService: e.Deps.ResolversService, AccountCatalog: e.Deps.AccountService, BalanceQuerier: e.Deps.HistoricalLedger,
	}, transactions.ResolutionScope{CustomerID: e.CustomerID, Namespace: e.Namespace}, transactions.IssueCustomerReceivableTemplate{
		At: e.Now(), Amount: input.Amount, Currency: input.Currency.Reference(), Features: input.Features, SpendChargeID: input.SpendChargeID,
	})
	require.NoError(t, err)
	_, err = e.Deps.HistoricalLedger.CommitGroup(t.Context(), transactions.GroupInputs(e.Namespace, nil, inputs...))
	require.NoError(t, err)
}
