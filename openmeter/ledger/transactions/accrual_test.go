package transactions

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"
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

func TestTransferCustomerReceivableToAccruedTemplate(t *testing.T) {
	env := newTransactionsTestEnv(t)

	inputs := env.resolveAndCommit(
		t,
		TransferCustomerReceivableToAccruedTemplate{
			At:       env.Now(),
			Amount:   alpacadecimal.NewFromInt(50),
			Currency: env.Currency,
		},
	)
	require.Len(t, inputs, 1)

	require.True(t, env.SumBalance(t, env.ReceivableSubAccount(t)).Equal(alpacadecimal.NewFromInt(-50)))
	require.True(t, env.SumBalance(t, env.AccruedSubAccount(t)).Equal(alpacadecimal.NewFromInt(50)))
}
