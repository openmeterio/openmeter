package transactions

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestRecognizeEarningsFromAttributableAccruedTemplate(t *testing.T) {
	env := newTransactionsTestEnv(t)
	costBasis := alpacadecimal.NewFromInt(1)

	env.resolveAndCommit(
		t,
		TransferCustomerReceivableToAccruedTemplate{
			At:        env.Now(),
			Amount:    alpacadecimal.NewFromInt(50),
			Currency:  env.Currency,
			CostBasis: &costBasis,
		},
	)

	inputs := env.resolveAndCommit(
		t,
		RecognizeEarningsFromAttributableAccruedTemplate{
			At:       env.Now(),
			Amount:   alpacadecimal.NewFromInt(50),
			Currency: env.Currency,
		},
	)
	require.Len(t, inputs, 1)

	require.True(t, env.SumBalance(t, env.AccruedSubAccountWithCostBasis(t, &costBasis)).Equal(alpacadecimal.Zero))
	require.True(t, env.SumBalance(t, env.EarningsSubAccountWithCostBasis(t, &costBasis)).Equal(alpacadecimal.NewFromInt(50)))
}

func TestRecognizeEarningsFromAttributableAccruedTemplate_IgnoresUnknownCostBasis(t *testing.T) {
	env := newTransactionsTestEnv(t)
	costBasis := alpacadecimal.NewFromInt(1)

	env.resolveAndCommit(
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
		TransferCustomerReceivableToAccruedTemplate{
			At:        env.Now(),
			Amount:    alpacadecimal.NewFromInt(20),
			Currency:  env.Currency,
			CostBasis: &costBasis,
		},
	)

	inputs := env.resolveAndCommit(
		t,
		RecognizeEarningsFromAttributableAccruedTemplate{
			At:       env.Now(),
			Amount:   alpacadecimal.NewFromInt(50),
			Currency: env.Currency,
		},
	)
	require.Len(t, inputs, 1)

	require.True(t, env.SumBalance(t, env.AccruedSubAccount(t)).Equal(alpacadecimal.NewFromInt(30)))
	require.True(t, env.SumBalance(t, env.AccruedSubAccountWithCostBasis(t, &costBasis)).Equal(alpacadecimal.Zero))
	require.True(t, env.SumBalance(t, env.EarningsSubAccountWithCostBasis(t, &costBasis)).Equal(alpacadecimal.NewFromInt(20)))
}

func TestRecognizeEarningsFromAttributableAccruedTemplate_PreservesChargeProvenance(t *testing.T) {
	env := newTransactionsTestEnv(t)
	costBasis := alpacadecimal.NewFromInt(1)
	sourceCharge1 := testChargeID(1)
	sourceCharge2 := testChargeID(2)
	spendCharge := testChargeID(3)
	priority := 1
	source1AccruedAmount := int64(10)
	source2AccruedAmount := int64(20)
	totalAccruedAmount := source1AccruedAmount + source2AccruedAmount
	correctedRecognizedAmount := int64(5)
	recognizedAmount := totalAccruedAmount / 2
	source1RecognizedAmount := source1AccruedAmount
	source2RecognizedAmount := recognizedAmount - source1RecognizedAmount
	source2AccruedAfterRecognition := source2AccruedAmount - source2RecognizedAmount
	source2AccruedAfterCorrection := source2AccruedAfterRecognition + correctedRecognizedAmount

	sourceFBO := env.fundPriorityWithCostBasis(t, priority, source1AccruedAmount, &costBasis, &sourceCharge1)
	env.fundPriorityWithCostBasis(t, priority, source2AccruedAmount, &costBasis, &sourceCharge2)

	// given:
	// - one spend charge has 30 accrued from two creditpurchase sources:
	//   - 10 from source 1
	//   - 20 from source 2
	env.resolveAndCommit(t, TransferCustomerFBOToAccruedTemplate{
		At:       env.Now(),
		Currency: env.Currency,
		Sources: []PostingAmount{
			{
				Address: sourceFBO.Address(),
				Amount:  alpacadecimal.NewFromInt(source1AccruedAmount),
				Identity: ledger.EntryIdentityParts{
					SourceChargeID: &sourceCharge1,
					SpendChargeID:  &spendCharge,
				},
			},
			{
				Address: sourceFBO.Address(),
				Amount:  alpacadecimal.NewFromInt(source2AccruedAmount),
				Identity: ledger.EntryIdentityParts{
					SourceChargeID: &sourceCharge2,
					SpendChargeID:  &spendCharge,
				},
			},
		},
	})

	// when:
	// - 15 of the 30 accrued amount is recognized
	recognizeInputs := env.resolve(t, RecognizeEarningsFromAttributableAccruedTemplate{
		At:       env.Now(),
		Amount:   alpacadecimal.NewFromInt(recognizedAmount),
		Currency: env.Currency,
	})
	require.Len(t, recognizeInputs, 1)

	group, err := env.Deps.HistoricalLedger.CommitGroup(t.Context(), GroupInputs(env.Namespace, nil, recognizeInputs...))
	require.NoError(t, err)

	// then:
	// - recognition keeps the exact source/spend provenance it consumed
	requireAccruedBalanceBuckets(t, env, map[string]float64{
		// 15 = source 2's original 20 less the 5 recognized after source 1.
		sourceSpendChargeKey(&sourceCharge2, &spendCharge): float64(source2AccruedAfterRecognition),
	})
	requireEarningsBalanceBuckets(t, env, map[string]float64{
		// 10 = source 1 was recognized first by deterministic bucket order.
		sourceSpendChargeKey(&sourceCharge1, &spendCharge): float64(source1RecognizedAmount),
		// 5 = remaining recognition amount came from source 2.
		sourceSpendChargeKey(&sourceCharge2, &spendCharge): float64(source2RecognizedAmount),
	})

	recognizeTx := findForwardTransaction(t, group, RecognizeEarningsFromAttributableAccruedTemplate{})

	// when:
	// - the 5 recognized from source 2 is corrected
	correctionInputs, err := CorrectTransaction(t.Context(), env.resolverDeps(), CorrectionInput{
		At:                  env.Now(),
		Amount:              alpacadecimal.NewFromInt(correctedRecognizedAmount),
		OriginalTransaction: recognizeTx,
		OriginalGroup:       group,
	})
	require.NoError(t, err)
	require.NotEmpty(t, correctionInputs)

	env.commit(t, correctionInputs...)

	// then:
	// - the correction restores the same provenance bucket it reversed
	requireAccruedBalanceBuckets(t, env, map[string]float64{
		// 20 = source 2's recognized slice was restored to accrued.
		sourceSpendChargeKey(&sourceCharge2, &spendCharge): float64(source2AccruedAfterCorrection),
	})
	requireEarningsBalanceBuckets(t, env, map[string]float64{
		// 10 = source 1 remains recognized.
		sourceSpendChargeKey(&sourceCharge1, &spendCharge): float64(source1RecognizedAmount),
	})
}

func TestRecognizeEarningsFromAttributableAccruedTemplate_PreservesRouteSource(t *testing.T) {
	env := newTransactionsTestEnv(t)
	currency := currencyx.Code("ACME")
	usd := currencyx.Code("USD")
	eur := currencyx.Code("EUR")
	costBasis := alpacadecimal.NewFromInt(1)
	priority := 1

	env.resolveAndCommit(t,
		IssueCustomerReceivableTemplate{
			At:             env.Now(),
			Amount:         alpacadecimal.NewFromInt(10),
			Currency:       currency,
			Source:         &usd,
			CostBasis:      &costBasis,
			CreditPriority: &priority,
		},
		IssueCustomerReceivableTemplate{
			At:             env.Now(),
			Amount:         alpacadecimal.NewFromInt(20),
			Currency:       currency,
			Source:         &eur,
			CostBasis:      &costBasis,
			CreditPriority: &priority,
		},
	)

	usdFBO, err := env.CustomerAccounts.FBOAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerFBORouteParams{
		Currency:       currency,
		Source:         &usd,
		CostBasis:      &costBasis,
		CreditPriority: priority,
	})
	require.NoError(t, err)
	eurFBO, err := env.CustomerAccounts.FBOAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerFBORouteParams{
		Currency:       currency,
		Source:         &eur,
		CostBasis:      &costBasis,
		CreditPriority: priority,
	})
	require.NoError(t, err)

	env.resolveAndCommit(t, TransferCustomerFBOToAccruedTemplate{
		At:       env.Now(),
		Currency: currency,
		Sources: []PostingAmount{
			{Address: usdFBO.Address(), Amount: alpacadecimal.NewFromInt(10)},
			{Address: eurFBO.Address(), Amount: alpacadecimal.NewFromInt(20)},
		},
	})

	recognizeInputs := env.resolve(t, RecognizeEarningsFromAttributableAccruedTemplate{
		At:       env.Now(),
		Amount:   alpacadecimal.NewFromInt(30),
		Currency: currency,
	})
	group, err := env.Deps.HistoricalLedger.CommitGroup(t.Context(), GroupInputs(env.Namespace, nil, recognizeInputs...))
	require.NoError(t, err)

	usdAccrued, err := env.CustomerAccounts.AccruedAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerAccruedRouteParams{
		Currency:  currency,
		Source:    &usd,
		CostBasis: &costBasis,
	})
	require.NoError(t, err)
	eurAccrued, err := env.CustomerAccounts.AccruedAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerAccruedRouteParams{
		Currency:  currency,
		Source:    &eur,
		CostBasis: &costBasis,
	})
	require.NoError(t, err)
	usdEarnings, err := env.BusinessAccounts.EarningsAccount.GetSubAccountForRoute(t.Context(), ledger.BusinessRouteParams{
		Currency:  currency,
		Source:    &usd,
		CostBasis: &costBasis,
	})
	require.NoError(t, err)
	eurEarnings, err := env.BusinessAccounts.EarningsAccount.GetSubAccountForRoute(t.Context(), ledger.BusinessRouteParams{
		Currency:  currency,
		Source:    &eur,
		CostBasis: &costBasis,
	})
	require.NoError(t, err)

	require.Equal(t, float64(0), env.SumBalance(t, usdAccrued).InexactFloat64())
	require.Equal(t, float64(0), env.SumBalance(t, eurAccrued).InexactFloat64())
	require.Equal(t, float64(10), env.SumBalance(t, usdEarnings).InexactFloat64())
	require.Equal(t, float64(20), env.SumBalance(t, eurEarnings).InexactFloat64())

	recognizeTx := findForwardTransaction(t, group, RecognizeEarningsFromAttributableAccruedTemplate{})
	correctionInputs, err := CorrectTransaction(t.Context(), env.resolverDeps(), CorrectionInput{
		At:                  env.Now(),
		Amount:              alpacadecimal.NewFromInt(30),
		OriginalTransaction: recognizeTx,
		OriginalGroup:       group,
	})
	require.NoError(t, err)
	env.commit(t, correctionInputs...)

	require.Equal(t, float64(10), env.SumBalance(t, usdAccrued).InexactFloat64())
	require.Equal(t, float64(20), env.SumBalance(t, eurAccrued).InexactFloat64())
	require.Equal(t, float64(0), env.SumBalance(t, usdEarnings).InexactFloat64())
	require.Equal(t, float64(0), env.SumBalance(t, eurEarnings).InexactFloat64())
}

func TestRecognizeEarningsCorrection_DoesNotTouchUnrecognizedInvoiceBackedAccrued(t *testing.T) {
	env := newTransactionsTestEnv(t)
	creditCostBasis := alpacadecimal.Zero
	invoiceCostBasis := alpacadecimal.NewFromInt(1)
	sourceChargeID := testChargeID(1)
	spendChargeID := testChargeID(2)
	priority := 1
	creditBackedAmount := int64(5)      // 5 = credit-backed accrued value eligible for recognition.
	invoiceBackedAmount := float64(7.5) // 7.5 = invoice-backed accrued value that remains unrecognized for now.

	sourceFBO := env.fundPriorityWithCostBasis(t, priority, creditBackedAmount, &creditCostBasis, &sourceChargeID)

	// given:
	// - one spend charge has accrued value from a credit source
	// - the same spend charge also has invoice-backed accrued value with source_charge_id unset
	env.resolveAndCommit(t,
		TransferCustomerFBOToAccruedTemplate{
			At:       env.Now(),
			Currency: env.Currency,
			Sources: []PostingAmount{
				{
					Address: sourceFBO.Address(),
					Amount:  alpacadecimal.NewFromInt(creditBackedAmount),
					Identity: ledger.EntryIdentityParts{
						SourceChargeID: &sourceChargeID,
						SpendChargeID:  &spendChargeID,
					},
				},
			},
		},
		TransferCustomerReceivableToAccruedTemplate{
			At:            env.Now(),
			Amount:        alpacadecimal.NewFromFloat(invoiceBackedAmount),
			Currency:      env.Currency,
			CostBasis:     &invoiceCostBasis,
			SpendChargeID: &spendChargeID,
		},
	)

	// when:
	// - only the credit-backed accrued amount is recognized
	recognizeInputs := env.resolve(t, RecognizeEarningsFromAttributableAccruedTemplate{
		At:       env.Now(),
		Amount:   alpacadecimal.NewFromInt(creditBackedAmount),
		Currency: env.Currency,
	})
	require.Len(t, recognizeInputs, 1)

	group, err := env.Deps.HistoricalLedger.CommitGroup(t.Context(), GroupInputs(env.Namespace, nil, recognizeInputs...))
	require.NoError(t, err)

	// then:
	// - invoice-backed accrued stays accrued
	// - earnings only contains the credit-backed source/spend bucket
	requireAccruedBalanceBuckets(t, env, map[string]float64{
		// 7.5 = invoice-backed accrued was intentionally not recognized.
		sourceSpendChargeKey(nil, &spendChargeID): invoiceBackedAmount,
	})
	requireEarningsBalanceBuckets(t, env, map[string]float64{
		// 5 = only the credit-backed accrued slice was recognized.
		sourceSpendChargeKey(&sourceChargeID, &spendChargeID): float64(creditBackedAmount),
	})

	recognizeTx := findForwardTransaction(t, group, RecognizeEarningsFromAttributableAccruedTemplate{})

	// when:
	// - recognition correction reverses the recognized credit-backed amount
	correctionInputs, err := CorrectTransaction(t.Context(), env.resolverDeps(), CorrectionInput{
		At:                  env.Now(),
		Amount:              alpacadecimal.NewFromInt(creditBackedAmount),
		OriginalTransaction: recognizeTx,
		OriginalGroup:       group,
	})
	require.NoError(t, err)
	require.NotEmpty(t, correctionInputs)

	env.commit(t, correctionInputs...)

	// then:
	// - correction restores only the recognized credit-backed slice
	// - invoice-backed accrued is unchanged because it was never recognized
	requireAccruedBalanceBuckets(t, env, map[string]float64{
		// 5 = credit-backed recognition was corrected back to accrued.
		sourceSpendChargeKey(&sourceChargeID, &spendChargeID): float64(creditBackedAmount),
		// 7.5 = invoice-backed accrued was not part of recognition or correction.
		sourceSpendChargeKey(nil, &spendChargeID): invoiceBackedAmount,
	})
	requireEarningsBalanceBuckets(t, env, map[string]float64{})
}

func TestRecognizeEarningsCorrection_FullReversal(t *testing.T) {
	env := newTransactionsTestEnv(t)
	costBasis := alpacadecimal.NewFromInt(1)

	// Set up accrued.
	env.resolveAndCommit(t, TransferCustomerReceivableToAccruedTemplate{
		At: env.Now(), Amount: alpacadecimal.NewFromInt(50), Currency: env.Currency, CostBasis: &costBasis,
	})

	// Recognize earnings — resolve and commit separately to get the group.
	recognizeInputs := env.resolve(t, RecognizeEarningsFromAttributableAccruedTemplate{
		At: env.Now(), Amount: alpacadecimal.NewFromInt(50), Currency: env.Currency,
	})
	require.Len(t, recognizeInputs, 1)

	group, err := env.Deps.HistoricalLedger.CommitGroup(t.Context(), GroupInputs(env.Namespace, nil, recognizeInputs...))
	require.NoError(t, err)

	recognizeTx := findForwardTransaction(t, group, RecognizeEarningsFromAttributableAccruedTemplate{})

	// Correct the full recognition amount.
	correctionInputs, err := CorrectTransaction(t.Context(), env.resolverDeps(), CorrectionInput{
		At:                  env.Now(),
		Amount:              alpacadecimal.NewFromInt(50),
		OriginalTransaction: recognizeTx,
		OriginalGroup:       group,
	})
	require.NoError(t, err)
	require.NotEmpty(t, correctionInputs)

	env.commit(t, correctionInputs...)

	// Accrued should be restored, earnings should be zero.
	require.True(t, env.SumBalance(t, env.AccruedSubAccountWithCostBasis(t, &costBasis)).Equal(alpacadecimal.NewFromInt(50)))
	require.True(t, env.SumBalance(t, env.EarningsSubAccountWithCostBasis(t, &costBasis)).Equal(alpacadecimal.Zero))
}

func TestRecognizeEarningsCorrection_PartialReversal(t *testing.T) {
	env := newTransactionsTestEnv(t)
	costBasis := alpacadecimal.NewFromInt(1)

	env.resolveAndCommit(t, TransferCustomerReceivableToAccruedTemplate{
		At: env.Now(), Amount: alpacadecimal.NewFromInt(50), Currency: env.Currency, CostBasis: &costBasis,
	})
	recognizeInputs := env.resolve(t, RecognizeEarningsFromAttributableAccruedTemplate{
		At: env.Now(), Amount: alpacadecimal.NewFromInt(50), Currency: env.Currency,
	})
	require.Len(t, recognizeInputs, 1)

	group, err := env.Deps.HistoricalLedger.CommitGroup(t.Context(), GroupInputs(env.Namespace, nil, recognizeInputs...))
	require.NoError(t, err)

	recognizeTx := findForwardTransaction(t, group, RecognizeEarningsFromAttributableAccruedTemplate{})

	// Correct only 20 of the 50 recognized.
	correctionInputs, err := CorrectTransaction(t.Context(), env.resolverDeps(), CorrectionInput{
		At:                  env.Now(),
		Amount:              alpacadecimal.NewFromInt(20),
		OriginalTransaction: recognizeTx,
		OriginalGroup:       group,
	})
	require.NoError(t, err)
	require.NotEmpty(t, correctionInputs)

	env.commit(t, correctionInputs...)

	require.True(t, env.SumBalance(t, env.AccruedSubAccountWithCostBasis(t, &costBasis)).Equal(alpacadecimal.NewFromInt(20)))
	require.True(t, env.SumBalance(t, env.EarningsSubAccountWithCostBasis(t, &costBasis)).Equal(alpacadecimal.NewFromInt(30)))
}

func TestRecognizeEarningsCorrection_OverCorrectionError(t *testing.T) {
	env := newTransactionsTestEnv(t)
	costBasis := alpacadecimal.NewFromInt(1)

	env.resolveAndCommit(t, TransferCustomerReceivableToAccruedTemplate{
		At: env.Now(), Amount: alpacadecimal.NewFromInt(50), Currency: env.Currency, CostBasis: &costBasis,
	})
	recognizeInputs := env.resolve(t, RecognizeEarningsFromAttributableAccruedTemplate{
		At: env.Now(), Amount: alpacadecimal.NewFromInt(50), Currency: env.Currency,
	})
	require.Len(t, recognizeInputs, 1)

	group, err := env.Deps.HistoricalLedger.CommitGroup(t.Context(), GroupInputs(env.Namespace, nil, recognizeInputs...))
	require.NoError(t, err)

	recognizeTx := findForwardTransaction(t, group, RecognizeEarningsFromAttributableAccruedTemplate{})

	// Attempt to correct more than was recognized.
	_, err = CorrectTransaction(t.Context(), env.resolverDeps(), CorrectionInput{
		At:                  env.Now(),
		Amount:              alpacadecimal.NewFromInt(100),
		OriginalTransaction: recognizeTx,
		OriginalGroup:       group,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "exceeds available amount")
}

func TestRecognizeEarningsCorrection_MultipleCostBases(t *testing.T) {
	env := newTransactionsTestEnv(t)
	costBasis1 := alpacadecimal.NewFromInt(1)
	costBasis2 := alpacadecimal.NewFromInt(2)

	// Set up two cost basis buckets.
	env.resolveAndCommit(t,
		TransferCustomerReceivableToAccruedTemplate{
			At: env.Now(), Amount: alpacadecimal.NewFromInt(30), Currency: env.Currency, CostBasis: &costBasis1,
		},
		TransferCustomerReceivableToAccruedTemplate{
			At: env.Now(), Amount: alpacadecimal.NewFromInt(20), Currency: env.Currency, CostBasis: &costBasis2,
		},
	)

	// Recognize earnings from both.
	recognizeInputs := env.resolve(t, RecognizeEarningsFromAttributableAccruedTemplate{
		At: env.Now(), Amount: alpacadecimal.NewFromInt(50), Currency: env.Currency,
	})
	require.Len(t, recognizeInputs, 1)

	group, err := env.Deps.HistoricalLedger.CommitGroup(t.Context(), GroupInputs(env.Namespace, nil, recognizeInputs...))
	require.NoError(t, err)

	recognizeTx := findForwardTransaction(t, group, RecognizeEarningsFromAttributableAccruedTemplate{})

	// Partially correct — should LIFO from the last entries.
	correctionInputs, err := CorrectTransaction(t.Context(), env.resolverDeps(), CorrectionInput{
		At:                  env.Now(),
		Amount:              alpacadecimal.NewFromInt(25),
		OriginalTransaction: recognizeTx,
		OriginalGroup:       group,
	})
	require.NoError(t, err)

	env.commit(t, correctionInputs...)

	// Cost basis 2 (20) should be fully restored, cost basis 1 should have 5 restored.
	require.True(t, env.SumBalance(t, env.AccruedSubAccountWithCostBasis(t, &costBasis2)).Equal(alpacadecimal.NewFromInt(20)))
	require.True(t, env.SumBalance(t, env.AccruedSubAccountWithCostBasis(t, &costBasis1)).Equal(alpacadecimal.NewFromInt(5)))
	require.True(t, env.SumBalance(t, env.EarningsSubAccountWithCostBasis(t, &costBasis1)).Equal(alpacadecimal.NewFromInt(25)))
	require.True(t, env.SumBalance(t, env.EarningsSubAccountWithCostBasis(t, &costBasis2)).Equal(alpacadecimal.Zero))
}

func requireEarningsBalanceBuckets(t *testing.T, env *transactionsTestEnv, expected map[string]float64) {
	t.Helper()

	earningsAccount, ok := env.BusinessAccounts.EarningsAccount.(accountIdentifier)
	require.True(t, ok)
	earningsAccountID := earningsAccount.ID().ID

	buckets, err := env.Deps.HistoricalLedger.GetBalanceBuckets(t.Context(), ledger.BalanceBucketQuery{
		Namespace: env.Namespace,
		Filters: ledger.Filters{
			AccountID: &earningsAccountID,
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

// findForwardTransaction finds the forward transaction for a given template in a group.
func findForwardTransaction(t *testing.T, group ledger.TransactionGroup, template TransactionTemplate) ledger.Transaction {
	t.Helper()

	name := TemplateCode(template)
	for _, tx := range group.Transactions() {
		txName, err := ledger.TransactionTemplateCodeFromAnnotations(tx.Annotations())
		if err != nil {
			continue
		}

		direction, err := ledger.TransactionDirectionFromAnnotations(tx.Annotations())
		if err != nil {
			continue
		}

		if txName == name && direction == ledger.TransactionDirectionForward {
			return tx
		}
	}

	t.Fatalf("forward transaction for %s not found in group", name)
	return nil
}
