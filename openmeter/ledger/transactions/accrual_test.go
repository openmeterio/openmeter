package transactions

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ledger"
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

	promoFBO := env.fundPriorityWithCostBasis(t, 1, 30, &promoCostBasis)
	purchasedFBO := env.fundPriorityWithCostBasis(t, 2, 50, &purchasedCostBasis)

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

func TestTransferCustomerFBOToAccruedCorrection_UsesReverseCollectionPriority(t *testing.T) {
	env := newTransactionsTestEnv(t)
	costBasis := alpacadecimal.NewFromInt(1)

	fboPriorityOne := env.fundPriorityWithCostBasis(t, 1, 10, &costBasis)
	fboPriorityTwo := env.fundPriorityWithCostBasis(t, 2, 20, &costBasis)

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

func TestTransferCustomerFBOToAccruedTemplate_SeparatesTaxCodeBuckets(t *testing.T) {
	env := newTransactionsTestEnv(t)
	costBasis := alpacadecimal.NewFromInt(1)

	taxA := "tax_A"
	taxB := "tax_B"

	// Fund two FBO sub-accounts: same priority+costBasis but different TaxCodes.
	fboTaxA := env.fundPriorityWithCostBasisAndTaxCode(t, 1, 30, &costBasis, &taxA)
	fboTaxB := env.fundPriorityWithCostBasisAndTaxCode(t, 1, 50, &costBasis, &taxB)

	inputs := env.resolveAndCommit(
		t,
		TransferCustomerFBOToAccruedTemplate{
			At:       env.Now(),
			Amount:   alpacadecimal.NewFromInt(80),
			Currency: env.Currency,
		},
	)
	require.Len(t, inputs, 1)

	// Both FBO sources fully drained.
	require.True(t, env.SumBalance(t, fboTaxA).Equal(alpacadecimal.Zero))
	require.True(t, env.SumBalance(t, fboTaxB).Equal(alpacadecimal.Zero))

	// Each TaxCode lands in its own accrued bucket.
	require.True(t, env.AccruedSubAccountWithCostBasisAndTaxCode(t, &costBasis, &taxA) != env.AccruedSubAccountWithCostBasisAndTaxCode(t, &costBasis, &taxB))
	require.True(t, env.SumBalance(t, env.AccruedSubAccountWithCostBasisAndTaxCode(t, &costBasis, &taxA)).Equal(alpacadecimal.NewFromInt(30)))
	require.True(t, env.SumBalance(t, env.AccruedSubAccountWithCostBasisAndTaxCode(t, &costBasis, &taxB)).Equal(alpacadecimal.NewFromInt(50)))
}

func TestTransferCustomerFBOToAccruedTemplate_NilTaxCodeIsolatedFromNonNil(t *testing.T) {
	env := newTransactionsTestEnv(t)

	taxA := "tax_A"

	// One FBO with nil TaxCode, one with non-nil TaxCode.
	fboNilTax := env.fundPriorityWithCostBasisAndTaxCode(t, 1, 40, nil, nil)
	fboTaxA := env.fundPriorityWithCostBasisAndTaxCode(t, 2, 20, nil, &taxA)

	inputs := env.resolveAndCommit(
		t,
		TransferCustomerFBOToAccruedTemplate{
			At:       env.Now(),
			Amount:   alpacadecimal.NewFromInt(60),
			Currency: env.Currency,
		},
	)
	require.Len(t, inputs, 1)

	require.True(t, env.SumBalance(t, fboNilTax).Equal(alpacadecimal.Zero))
	require.True(t, env.SumBalance(t, fboTaxA).Equal(alpacadecimal.Zero))

	// Nil TaxCode and non-nil TaxCode land in separate accrued buckets.
	require.True(t, env.SumBalance(t, env.AccruedSubAccountWithCostBasisAndTaxCode(t, nil, nil)).Equal(alpacadecimal.NewFromInt(40)))
	require.True(t, env.SumBalance(t, env.AccruedSubAccountWithCostBasisAndTaxCode(t, nil, &taxA)).Equal(alpacadecimal.NewFromInt(20)))
}

func TestTranslateCustomerAccruedCostBasisTemplate(t *testing.T) {
	env := newTransactionsTestEnv(t)
	purchasedCostBasis := alpacadecimal.NewFromInt(1)

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
	)

	inputs := env.resolveAndCommit(
		t,
		TranslateCustomerAccruedCostBasisTemplate{
			At:            env.Now(),
			Amount:        alpacadecimal.NewFromInt(30),
			Currency:      env.Currency,
			FromCostBasis: nil,
			ToCostBasis:   &purchasedCostBasis,
		},
	)
	require.Len(t, inputs, 1)

	require.True(t, env.SumBalance(t, env.AccruedSubAccount(t)).Equal(alpacadecimal.Zero))
	require.True(t, env.SumBalance(t, env.AccruedSubAccountWithCostBasis(t, &purchasedCostBasis)).Equal(alpacadecimal.NewFromInt(30)))
}
