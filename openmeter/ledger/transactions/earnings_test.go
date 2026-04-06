package transactions

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"
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
