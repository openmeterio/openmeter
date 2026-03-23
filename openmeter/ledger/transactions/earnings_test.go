package transactions

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"
)

func TestRecognizeEarningsFromAccruedTemplate(t *testing.T) {
	env := newTransactionsTestEnv(t)

	env.resolveAndCommit(
		t,
		TransferCustomerReceivableToAccruedTemplate{
			At:       env.Now(),
			Amount:   alpacadecimal.NewFromInt(50),
			Currency: env.Currency,
		},
	)

	inputs := env.resolveAndCommit(
		t,
		RecognizeEarningsFromAccruedTemplate{
			At:       env.Now(),
			Amount:   alpacadecimal.NewFromInt(50),
			Currency: env.Currency,
		},
	)
	require.Len(t, inputs, 1)

	require.True(t, env.SumBalance(t, env.AccruedSubAccount(t)).Equal(alpacadecimal.Zero))
	require.True(t, env.SumBalance(t, env.EarningsSubAccount(t)).Equal(alpacadecimal.NewFromInt(50)))
}
