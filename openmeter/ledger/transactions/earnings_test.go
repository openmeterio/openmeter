package transactions

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ledger"
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
	require.Contains(t, err.Error(), "exceeds original recognized amount")
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

// findForwardTransaction finds the forward transaction for a given template in a group.
func findForwardTransaction(t *testing.T, group ledger.TransactionGroup, template TransactionTemplate) ledger.Transaction {
	t.Helper()

	name := templateName(template)
	for _, tx := range group.Transactions() {
		txName, err := ledger.TransactionTemplateNameFromAnnotations(tx.Annotations())
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
