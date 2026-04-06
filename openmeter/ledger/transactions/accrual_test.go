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
			Amount:   alpacadecimal.NewFromInt(60),
			Currency: env.Currency,
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
			Amount:   alpacadecimal.NewFromInt(60),
			Currency: env.Currency,
		},
	)
	require.Len(t, inputs, 1)

	require.True(t, env.SumBalance(t, promoFBO).Equal(alpacadecimal.Zero))
	require.True(t, env.SumBalance(t, purchasedFBO).Equal(alpacadecimal.NewFromInt(20)))
	require.True(t, env.SumBalance(t, env.AccruedSubAccountWithCostBasis(t, &promoCostBasis)).Equal(alpacadecimal.NewFromInt(30)))
	require.True(t, env.SumBalance(t, env.AccruedSubAccountWithCostBasis(t, &purchasedCostBasis)).Equal(alpacadecimal.NewFromInt(30)))
	require.True(t, env.SumBalance(t, env.AccruedSubAccount(t)).Equal(alpacadecimal.Zero))
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
