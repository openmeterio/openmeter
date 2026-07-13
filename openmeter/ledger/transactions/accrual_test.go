package transactions

import (
	"fmt"
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestTransferCustomerFBOToAccruedTemplate(t *testing.T) {
	env := newTransactionsTestEnv(t)

	priorityTwo := env.fundPriority(t, 2, 50)
	priorityOne := env.fundPriority(t, 1, 30)

	inputs := env.resolveAndCommit(
		t,
		TransferCustomerFBOToAccruedTemplate{
			At:       env.Now(),
			Currency: env.Currency,
			Sources: []PostingAmount{
				{
					Address: priorityOne.Address(),
					Amount:  alpacadecimal.NewFromInt(30),
				},
				{
					Address: priorityTwo.Address(),
					Amount:  alpacadecimal.NewFromInt(30),
				},
			},
		},
	)
	require.Len(t, inputs, 1)

	require.True(t, env.SumBalance(t, priorityOne).Equal(alpacadecimal.NewFromInt(0)))
	require.True(t, env.SumBalance(t, priorityTwo).Equal(alpacadecimal.NewFromInt(20)))
	require.True(t, env.SumBalance(t, env.AccruedSubAccount(t)).Equal(alpacadecimal.NewFromInt(60)))
}

func TestTransferCustomerFBOToAccruedTemplate_PreservesCostBasisAcrossBuckets(t *testing.T) {
	env := newTransactionsTestEnv(t)

	promoCostBasis := alpacadecimal.Zero
	purchasedCostBasis := alpacadecimal.NewFromInt(1)

	promoFBO := env.fundPriorityWithCostBasis(t, 1, 30, &promoCostBasis, nil)
	purchasedFBO := env.fundPriorityWithCostBasis(t, 2, 50, &purchasedCostBasis, nil)

	inputs := env.resolveAndCommit(
		t,
		TransferCustomerFBOToAccruedTemplate{
			At:       env.Now(),
			Currency: env.Currency,
			Sources: []PostingAmount{
				{
					Address: promoFBO.Address(),
					Amount:  alpacadecimal.NewFromInt(30),
				},
				{
					Address: purchasedFBO.Address(),
					Amount:  alpacadecimal.NewFromInt(30),
				},
			},
		},
	)
	require.Len(t, inputs, 1)

	require.True(t, env.SumBalance(t, promoFBO).Equal(alpacadecimal.Zero))
	require.True(t, env.SumBalance(t, purchasedFBO).Equal(alpacadecimal.NewFromInt(20)))
	require.True(t, env.SumBalance(t, env.AccruedSubAccountWithCostBasis(t, &promoCostBasis)).Equal(alpacadecimal.NewFromInt(30)))
	require.True(t, env.SumBalance(t, env.AccruedSubAccountWithCostBasis(t, &purchasedCostBasis)).Equal(alpacadecimal.NewFromInt(30)))
	require.True(t, env.SumBalance(t, env.AccruedSubAccount(t)).Equal(alpacadecimal.Zero))
}

func TestTransferCustomerFBOToAccruedTemplate_SeparatesRouteSources(t *testing.T) {
	env := newTransactionsTestEnv(t)
	env.Currency = currencyx.Code("ACME")

	usd := currencyx.Code("USD")
	eur := currencyx.Code("EUR")
	costBasis := alpacadecimal.RequireFromString("0.5")
	priority := ledger.DefaultCustomerFBOPriority

	env.resolveAndCommit(
		t,
		IssueCustomerReceivableTemplate{
			At:             env.Now(),
			Amount:         alpacadecimal.NewFromInt(30),
			Currency:       env.Currency,
			Source:         &usd,
			CostBasis:      &costBasis,
			CreditPriority: &priority,
		},
		IssueCustomerReceivableTemplate{
			At:             env.Now(),
			Amount:         alpacadecimal.NewFromInt(20),
			Currency:       env.Currency,
			Source:         &eur,
			CostBasis:      &costBasis,
			CreditPriority: &priority,
		},
	)

	usdFBO, err := env.CustomerAccounts.FBOAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerFBORouteParams{
		Currency:       env.Currency,
		Source:         &usd,
		CostBasis:      &costBasis,
		CreditPriority: priority,
	})
	require.NoError(t, err)
	eurFBO, err := env.CustomerAccounts.FBOAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerFBORouteParams{
		Currency:       env.Currency,
		Source:         &eur,
		CostBasis:      &costBasis,
		CreditPriority: priority,
	})
	require.NoError(t, err)

	inputs := env.resolveAndCommit(
		t,
		TransferCustomerFBOToAccruedTemplate{
			At:       env.Now(),
			Currency: env.Currency,
			Sources: []PostingAmount{
				{
					Address: usdFBO.Address(),
					Amount:  alpacadecimal.NewFromInt(30),
				},
				{
					Address: eurFBO.Address(),
					Amount:  alpacadecimal.NewFromInt(20),
				},
			},
		},
	)
	require.Len(t, inputs, 1)

	usdAccrued, err := env.CustomerAccounts.AccruedAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerAccruedRouteParams{
		Currency:  env.Currency,
		Source:    &usd,
		CostBasis: &costBasis,
	})
	require.NoError(t, err)
	eurAccrued, err := env.CustomerAccounts.AccruedAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerAccruedRouteParams{
		Currency:  env.Currency,
		Source:    &eur,
		CostBasis: &costBasis,
	})
	require.NoError(t, err)

	require.Equal(t, &usd, usdAccrued.Route().Source)
	require.Equal(t, &eur, eurAccrued.Route().Source)
	require.Equal(t, float64(0), env.SumBalance(t, usdFBO).InexactFloat64())
	require.Equal(t, float64(0), env.SumBalance(t, eurFBO).InexactFloat64())
	require.Equal(t, float64(30), env.SumBalance(t, usdAccrued).InexactFloat64())
	require.Equal(t, float64(20), env.SumBalance(t, eurAccrued).InexactFloat64())
}

func TestTransferCustomerFBOToAccruedTemplate_PreservesChargeProvenance(t *testing.T) {
	env := newTransactionsTestEnv(t)

	sourceChargeID := testChargeID(1)
	spendChargeID := testChargeID(2)
	fbo := env.fundPriority(t, 1, 50)

	inputs := env.resolveAndCommit(
		t,
		TransferCustomerFBOToAccruedTemplate{
			At:       env.Now(),
			Currency: env.Currency,
			Sources: []PostingAmount{
				{
					Address: fbo.Address(),
					Amount:  alpacadecimal.NewFromInt(30),
					Identity: ledger.EntryIdentityParts{
						SourceChargeID: &sourceChargeID,
						SpendChargeID:  &spendChargeID,
					},
				},
			},
		},
	)
	require.Len(t, inputs, 1)

	requireAccruedBalanceBuckets(t, env, map[string]float64{
		sourceSpendChargeKey(&sourceChargeID, &spendChargeID): 30,
	})
}

func TestTransferCustomerFBOToAccruedCorrection_UsesReverseCollectionPriority(t *testing.T) {
	env := newTransactionsTestEnv(t)
	costBasis := alpacadecimal.NewFromInt(1)

	fboPriorityOne := env.fundPriorityWithCostBasis(t, 1, 10, &costBasis, nil)
	fboPriorityTwo := env.fundPriorityWithCostBasis(t, 2, 20, &costBasis, nil)

	originalInputs := env.resolve(t, TransferCustomerFBOToAccruedTemplate{
		At:       env.Now(),
		Currency: env.Currency,
		Sources: []PostingAmount{
			{
				Address: fboPriorityOne.Address(),
				Amount:  alpacadecimal.NewFromInt(10),
			},
			{
				Address: fboPriorityTwo.Address(),
				Amount:  alpacadecimal.NewFromInt(20),
			},
		},
	})
	require.Len(t, originalInputs, 1)

	group, err := env.Deps.HistoricalLedger.CommitGroup(t.Context(), GroupInputs(env.Namespace, nil, originalInputs...))
	require.NoError(t, err)

	correctionInputs, err := CorrectTransaction(t.Context(), env.resolverDeps(), CorrectionInput{
		At:                  env.Now(),
		Amount:              alpacadecimal.NewFromInt(25),
		OriginalTransaction: group.Transactions()[0],
		OriginalGroup:       group,
	})
	require.NoError(t, err)

	env.commit(t, correctionInputs...)

	require.True(t, env.SumBalance(t, fboPriorityOne).Equal(alpacadecimal.NewFromInt(5)))
	require.True(t, env.SumBalance(t, fboPriorityTwo).Equal(alpacadecimal.NewFromInt(20)))
	require.True(t, env.SumBalance(t, env.AccruedSubAccountWithCostBasis(t, &costBasis)).Equal(alpacadecimal.NewFromInt(5)))
}

func TestTransferCustomerFBOToAccruedCorrection_PreservesChargeProvenance(t *testing.T) {
	env := newTransactionsTestEnv(t)

	// given:
	// - one same-route accrued transaction is split across two source x spend identities
	// when:
	// - the transaction is partially corrected
	// then:
	// - reverse correction restores the last source and preserves accrued provenance on the remainder
	sourceCharge1 := testChargeID(1)
	sourceCharge2 := testChargeID(2)
	spendCharge1 := testChargeID(3)
	spendCharge2 := testChargeID(4)
	priority := 1                     // same priority forces identity, not route, to distinguish the two source buckets.
	firstSourceAmount := int64(10)    // first source is fully consumed and remains fully accrued after correction.
	secondSourceAmount := int64(20)   // second source is partially corrected because it was collected last.
	correctionAmount := int64(15)     // correction unwinds 15 from the second source by reverse collection order.
	secondSourceRemainder := int64(5) // second source started accrued at 20 and keeps only 5 after correcting 15.

	sourceFBO := env.fundPriorityWithCostBasis(t, priority, firstSourceAmount, nil, &sourceCharge1)
	env.fundPriorityWithCostBasis(t, priority, secondSourceAmount, nil, &sourceCharge2)

	collectionSource0 := "0" // first collection source remains fully accrued after the reverse-order correction.
	collectionSource1 := "1" // second collection source is corrected first because it was collected last.
	originalInputs := env.resolve(t, TransferCustomerFBOToAccruedTemplate{
		At:       env.Now(),
		Currency: env.Currency,
		Sources: []PostingAmount{
			{
				Address: sourceFBO.Address(),
				Amount:  alpacadecimal.NewFromInt(firstSourceAmount),
				Identity: ledger.EntryIdentityParts{
					CollectionSource: &collectionSource0,
					SourceChargeID:   &sourceCharge1,
					SpendChargeID:    &spendCharge1,
				},
			},
			{
				Address: sourceFBO.Address(),
				Amount:  alpacadecimal.NewFromInt(secondSourceAmount),
				Identity: ledger.EntryIdentityParts{
					CollectionSource: &collectionSource1,
					SourceChargeID:   &sourceCharge2,
					SpendChargeID:    &spendCharge2,
				},
			},
		},
	})
	require.Len(t, originalInputs, 1) // both source legs are posted inside one accrued transfer transaction.

	group, err := env.Deps.HistoricalLedger.CommitGroup(t.Context(), GroupInputs(env.Namespace, nil, originalInputs...))
	require.NoError(t, err)

	correctionInputs, err := CorrectTransaction(t.Context(), env.resolverDeps(), CorrectionInput{
		At:                  env.Now(),
		Amount:              alpacadecimal.NewFromInt(correctionAmount),
		OriginalTransaction: group.Transactions()[0],
		OriginalGroup:       group,
	})
	require.NoError(t, err)

	env.commit(t, correctionInputs...)

	requireFBOBalanceBuckets(t, env, map[string]float64{
		sourceSpendChargeKey(&sourceCharge2, nil): float64(correctionAmount), // corrected 15 returns to the last collected source.
	})
	requireAccruedBalanceBuckets(t, env, map[string]float64{
		sourceSpendChargeKey(&sourceCharge1, &spendCharge1): float64(firstSourceAmount),     // first source keeps its full 10 accrued balance.
		sourceSpendChargeKey(&sourceCharge2, &spendCharge2): float64(secondSourceRemainder), // second source keeps 5 after correcting 15 from 20.
	})
}

func TestTransferCustomerReceivableToAccruedTemplate(t *testing.T) {
	env := newTransactionsTestEnv(t)
	costBasis := alpacadecimal.NewFromInt(1)

	inputs := env.resolveAndCommit(
		t,
		TransferCustomerReceivableToAccruedTemplate{
			At:        env.Now(),
			Amount:    alpacadecimal.NewFromInt(50),
			Currency:  env.Currency,
			CostBasis: &costBasis,
		},
	)
	require.Len(t, inputs, 1)

	require.True(t, env.SumBalance(t, env.ReceivableSubAccountWithCostBasis(t, &costBasis)).Equal(alpacadecimal.NewFromInt(-50)))
	require.True(t, env.SumBalance(t, env.AccruedSubAccountWithCostBasis(t, &costBasis)).Equal(alpacadecimal.NewFromInt(50)))
}

func TestTransferCustomerReceivableToAccruedTemplate_StampsSpendCharge(t *testing.T) {
	env := newTransactionsTestEnv(t)
	costBasis := alpacadecimal.NewFromInt(1)

	spendChargeID := testChargeID(1)
	inputs := env.resolveAndCommit(
		t,
		TransferCustomerReceivableToAccruedTemplate{
			At:            env.Now(),
			Amount:        alpacadecimal.NewFromInt(50),
			Currency:      env.Currency,
			CostBasis:     &costBasis,
			SpendChargeID: &spendChargeID,
		},
	)
	require.Len(t, inputs, 1)

	requireAccruedBalanceBuckets(t, env, map[string]float64{
		sourceSpendChargeKey(nil, &spendChargeID): 50,
	})
}

func TestTransferCustomerFBOAdvanceToAccruedTemplate_UnknownCostBasisAdvanceNetEffect(t *testing.T) {
	env := newTransactionsTestEnv(t)

	inputs := env.resolveAndCommit(
		t,
		IssueCustomerReceivableTemplate{
			At:       env.Now(),
			Amount:   alpacadecimal.NewFromInt(30),
			Currency: env.Currency,
		},
		TransferCustomerFBOAdvanceToAccruedTemplate{
			At:       env.Now(),
			Amount:   alpacadecimal.NewFromInt(30),
			Currency: env.Currency,
		},
	)
	require.Len(t, inputs, 2)

	require.True(t, env.SumBalance(t, env.FBOSubAccount(t, ledger.DefaultCustomerFBOPriority)).Equal(alpacadecimal.Zero))
	require.True(t, env.SumBalance(t, env.ReceivableSubAccount(t)).Equal(alpacadecimal.NewFromInt(-30)))
	require.True(t, env.SumBalance(t, env.AccruedSubAccount(t)).Equal(alpacadecimal.NewFromInt(30)))
}

func TestTransferCustomerFBOAdvanceToAccruedTemplate_AppliesTaxBehaviorToAccrued(t *testing.T) {
	env := newTransactionsTestEnv(t)
	taxCode := "tax_A"
	taxBehavior := ledger.TaxBehaviorExclusive

	inputs := env.resolveAndCommit(
		t,
		IssueCustomerReceivableTemplate{
			At:       env.Now(),
			Amount:   alpacadecimal.NewFromInt(30),
			Currency: env.Currency,
		},
		TransferCustomerFBOAdvanceToAccruedTemplate{
			At:          env.Now(),
			Amount:      alpacadecimal.NewFromInt(30),
			Currency:    env.Currency,
			TaxCode:     &taxCode,
			TaxBehavior: &taxBehavior,
		},
	)
	require.Len(t, inputs, 2)

	accruedWithTaxBehavior, err := env.CustomerAccounts.AccruedAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerAccruedRouteParams{
		Currency:    env.Currency,
		TaxCode:     &taxCode,
		TaxBehavior: &taxBehavior,
	})
	require.NoError(t, err)

	require.True(t, env.SumBalance(t, env.FBOSubAccount(t, ledger.DefaultCustomerFBOPriority)).Equal(alpacadecimal.Zero))
	require.True(t, env.SumBalance(t, accruedWithTaxBehavior).Equal(alpacadecimal.NewFromInt(30)))
	require.True(t, env.SumBalance(t, env.AccruedSubAccount(t)).Equal(alpacadecimal.Zero))
}

func TestTransferCustomerFBOToAccruedTemplate_AppliesTaxConfigToAccrued(t *testing.T) {
	env := newTransactionsTestEnv(t)
	costBasis := alpacadecimal.NewFromInt(1)

	taxA := "tax_A"
	taxBehavior := ledger.TaxBehaviorInclusive

	// Tax dimensions come from accrual, not from selected FBO sources.
	fboA := env.fundPriorityWithCostBasis(t, 1, 30, &costBasis, nil)
	fboB := env.fundPriorityWithCostBasis(t, 2, 50, &costBasis, nil)

	inputs := env.resolveAndCommit(
		t,
		TransferCustomerFBOToAccruedTemplate{
			At:          env.Now(),
			Currency:    env.Currency,
			TaxCode:     &taxA,
			TaxBehavior: &taxBehavior,
			Sources: []PostingAmount{
				{Address: fboA.Address(), Amount: alpacadecimal.NewFromInt(30)},
				{Address: fboB.Address(), Amount: alpacadecimal.NewFromInt(50)},
			},
		},
	)
	require.Len(t, inputs, 1)

	// Both FBO sources fully drained.
	require.True(t, env.SumBalance(t, fboA).Equal(alpacadecimal.Zero))
	require.True(t, env.SumBalance(t, fboB).Equal(alpacadecimal.Zero))

	accrued, err := env.CustomerAccounts.AccruedAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerAccruedRouteParams{
		Currency:    env.Currency,
		CostBasis:   &costBasis,
		TaxCode:     &taxA,
		TaxBehavior: &taxBehavior,
	})
	require.NoError(t, err)
	require.True(t, env.SumBalance(t, accrued).Equal(alpacadecimal.NewFromInt(80)))
}

func TestTransferCustomerFBOToAccruedTemplate_NilTaxConfigUsesNilAccruedRoute(t *testing.T) {
	env := newTransactionsTestEnv(t)

	fboA := env.fundPriority(t, 1, 40)
	fboB := env.fundPriority(t, 2, 20)

	inputs := env.resolveAndCommit(
		t,
		TransferCustomerFBOToAccruedTemplate{
			At:       env.Now(),
			Currency: env.Currency,
			Sources: []PostingAmount{
				{Address: fboA.Address(), Amount: alpacadecimal.NewFromInt(40)},
				{Address: fboB.Address(), Amount: alpacadecimal.NewFromInt(20)},
			},
		},
	)
	require.Len(t, inputs, 1)

	require.True(t, env.SumBalance(t, fboA).Equal(alpacadecimal.Zero))
	require.True(t, env.SumBalance(t, fboB).Equal(alpacadecimal.Zero))
	require.True(t, env.SumBalance(t, env.AccruedSubAccount(t)).Equal(alpacadecimal.NewFromInt(60)))
}

func TestTranslateCustomerAccruedCostBasisTemplate(t *testing.T) {
	env := newTransactionsTestEnv(t)
	purchasedCostBasis := alpacadecimal.NewFromInt(1)
	sourceChargeID := testChargeID(1)
	spendChargeID := testChargeID(2)

	env.resolveAndCommit(
		t,
		IssueCustomerReceivableTemplate{
			At:            env.Now(),
			Amount:        alpacadecimal.NewFromInt(30),
			Currency:      env.Currency,
			SpendChargeID: &spendChargeID,
		},
		TransferCustomerFBOAdvanceToAccruedTemplate{
			At:            env.Now(),
			Amount:        alpacadecimal.NewFromInt(30),
			Currency:      env.Currency,
			SpendChargeID: &spendChargeID,
		},
	)

	inputs := env.resolveAndCommit(
		t,
		TranslateCustomerAccruedCostBasisTemplate{
			At:             env.Now(),
			Amount:         alpacadecimal.NewFromInt(30),
			Currency:       env.Currency,
			FromCostBasis:  nil,
			ToCostBasis:    &purchasedCostBasis,
			SourceChargeID: &sourceChargeID,
			SpendChargeID:  &spendChargeID,
		},
	)
	require.Len(t, inputs, 1)

	require.True(t, env.SumBalance(t, env.AccruedSubAccount(t)).Equal(alpacadecimal.Zero))
	require.True(t, env.SumBalance(t, env.AccruedSubAccountWithCostBasis(t, &purchasedCostBasis)).Equal(alpacadecimal.NewFromInt(30)))
	requireAccruedBalanceBuckets(t, env, map[string]float64{
		sourceSpendChargeKey(&sourceChargeID, &spendChargeID): 30,
	})
}

func requireAccruedBalanceBuckets(t *testing.T, env *transactionsTestEnv, expected map[string]float64) {
	t.Helper()

	accruedAccount, ok := env.CustomerAccounts.AccruedAccount.(accountIdentifier)
	require.True(t, ok)
	accruedAccountID := accruedAccount.ID().ID

	buckets, err := env.Deps.HistoricalLedger.GetBalanceBuckets(t.Context(), ledger.BalanceBucketQuery{
		Namespace: env.Namespace,
		Filters: ledger.Filters{
			AccountID: &accruedAccountID,
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

func requireFBOBalanceBuckets(t *testing.T, env *transactionsTestEnv, expected map[string]float64) {
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
