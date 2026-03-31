package transactions

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ledger"
)

func TestIssueCustomerReceivableTemplate(t *testing.T) {
	env := newTransactionsTestEnv(t)

	priority := 7
	inputs := env.resolveAndCommit(
		t,
		IssueCustomerReceivableTemplate{
			At:             env.Now(),
			Amount:         alpacadecimal.NewFromInt(50),
			Currency:       env.Currency,
			CreditPriority: &priority,
		},
	)
	require.Len(t, inputs, 1)

	require.True(t, env.SumBalance(t, env.FBOSubAccount(t, priority)).Equal(alpacadecimal.NewFromInt(50)))
	require.True(t, env.SumBalance(t, env.ReceivableSubAccount(t)).Equal(alpacadecimal.NewFromInt(-50)))
}

func TestIssueCustomerReceivableTemplate_DefaultPriority(t *testing.T) {
	env := newTransactionsTestEnv(t)

	inputs := env.resolveAndCommit(
		t,
		IssueCustomerReceivableTemplate{
			At:       env.Now(),
			Amount:   alpacadecimal.NewFromInt(15),
			Currency: env.Currency,
		},
	)
	require.Len(t, inputs, 1)

	require.True(t, env.SumBalance(t, env.FBOSubAccount(t, ledger.DefaultCustomerFBOPriority)).Equal(alpacadecimal.NewFromInt(15)))
	require.True(t, env.SumBalance(t, env.ReceivableSubAccount(t)).Equal(alpacadecimal.NewFromInt(-15)))
}

func TestFundCustomerReceivableTemplate(t *testing.T) {
	env := newTransactionsTestEnv(t)

	env.resolveAndCommit(
		t,
		IssueCustomerReceivableTemplate{
			At:       env.Now(),
			Amount:   alpacadecimal.NewFromInt(40),
			Currency: env.Currency,
		},
	)

	inputs := env.resolveAndCommit(
		t,
		FundCustomerReceivableTemplate{
			At:       env.Now(),
			Amount:   alpacadecimal.NewFromInt(40),
			Currency: env.Currency,
		},
	)
	require.Len(t, inputs, 1)

	require.True(t, env.SumBalance(t, env.ReceivableSubAccount(t)).Equal(alpacadecimal.NewFromInt(-40)))
	require.True(t, env.SumBalance(t, env.ReceivableSubAccountWithStatus(t, ledger.TransactionAuthorizationStatusAuthorized)).Equal(alpacadecimal.NewFromInt(40)))
	require.True(t, env.SumBalance(t, env.WashSubAccount(t)).Equal(alpacadecimal.NewFromInt(-40)))
	require.True(t, env.SumBalance(t, env.FBOSubAccount(t, ledger.DefaultCustomerFBOPriority)).Equal(alpacadecimal.NewFromInt(40)))
}

func TestCoverCustomerReceivableTemplate(t *testing.T) {
	env := newTransactionsTestEnv(t)

	priority := 3
	env.resolveAndCommit(
		t,
		IssueCustomerReceivableTemplate{
			At:             env.Now(),
			Amount:         alpacadecimal.NewFromInt(45),
			Currency:       env.Currency,
			CreditPriority: &priority,
		},
	)

	inputs := env.resolveAndCommit(
		t,
		CoverCustomerReceivableTemplate{
			At:             env.Now(),
			Amount:         alpacadecimal.NewFromInt(45),
			Currency:       env.Currency,
			CreditPriority: &priority,
		},
	)
	require.Len(t, inputs, 1)

	require.True(t, env.SumBalance(t, env.FBOSubAccount(t, priority)).Equal(alpacadecimal.Zero))
	require.True(t, env.SumBalance(t, env.ReceivableSubAccount(t)).Equal(alpacadecimal.Zero))
}

func TestSettleCustomerReceivablePaymentTemplate(t *testing.T) {
	env := newTransactionsTestEnv(t)

	env.resolveAndCommit(
		t,
		IssueCustomerReceivableTemplate{
			At:       env.Now(),
			Amount:   alpacadecimal.NewFromInt(40),
			Currency: env.Currency,
		},
		FundCustomerReceivableTemplate{
			At:       env.Now(),
			Amount:   alpacadecimal.NewFromInt(40),
			Currency: env.Currency,
		},
	)

	inputs := env.resolveAndCommit(
		t,
		SettleCustomerReceivablePaymentTemplate{
			At:       env.Now(),
			Amount:   alpacadecimal.NewFromInt(40),
			Currency: env.Currency,
		},
	)
	require.Len(t, inputs, 1)

	require.True(t, env.SumBalance(t, env.ReceivableSubAccount(t)).Equal(alpacadecimal.Zero))
	require.True(t, env.SumBalance(t, env.ReceivableSubAccountWithStatus(t, ledger.TransactionAuthorizationStatusAuthorized)).Equal(alpacadecimal.Zero))
	require.True(t, env.SumBalance(t, env.WashSubAccount(t)).Equal(alpacadecimal.NewFromInt(-40)))
}

func TestAttributeCustomerAdvanceReceivableCostBasisTemplate(t *testing.T) {
	env := newTransactionsTestEnv(t)
	purchasedCostBasis := alpacadecimal.NewFromInt(1)

	env.resolveAndCommit(
		t,
		IssueCustomerReceivableTemplate{
			At:       env.Now(),
			Amount:   alpacadecimal.NewFromInt(40),
			Currency: env.Currency,
		},
		FundCustomerReceivableTemplate{
			At:        env.Now(),
			Amount:    alpacadecimal.NewFromInt(40),
			Currency:  env.Currency,
			CostBasis: &purchasedCostBasis,
		},
	)

	inputs := env.resolveAndCommit(
		t,
		AttributeCustomerAdvanceReceivableCostBasisTemplate{
			At:        env.Now(),
			Amount:    alpacadecimal.NewFromInt(40),
			Currency:  env.Currency,
			CostBasis: &purchasedCostBasis,
		},
	)
	require.Len(t, inputs, 1)

	require.True(t, env.SumBalance(t, env.ReceivableSubAccountWithCostBasis(t, &purchasedCostBasis)).Equal(alpacadecimal.NewFromInt(-40)))
	require.True(t, env.SumBalance(t, env.ReceivableSubAccount(t)).Equal(alpacadecimal.Zero))
}
