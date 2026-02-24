package ledger_test

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/pkg/models"
)

// The point of this code is to test the primitive APIs with a more complex flow

type testTransactionGroupInput struct {
	transactions []ledger.TransactionInput
}

var _ ledger.TransactionGroupInput = testTransactionGroupInput{}

func (t testTransactionGroupInput) Namespace() string {
	return "default-ns"
}

func (t testTransactionGroupInput) Transactions() []ledger.TransactionInput {
	return t.transactions
}

func (t testTransactionGroupInput) Annotations() models.Annotations {
	return nil
}

func asTxGroupInput(transactions []ledger.TransactionInput) ledger.TransactionGroupInput {
	return testTransactionGroupInput{transactions: transactions}
}

func TestFXOnInvoiceIssued(t *testing.T) {
	t.Skipf("This is just to assert the types, it would fail on unimplemented")
	// We're simulating the scenario where we're effective converting an outstanding CRD balance to an outstanding FIAT balance

	// Let's define some mocks to see a more realistic control flow
	type customerAccounts struct {
		USD            ledger.SubAccount
		CRD            ledger.SubAccount
		USDOutstanding ledger.SubAccount
		CRDOutstanding ledger.SubAccount
	}

	var l ledger.Ledger
	costBasis := alpacadecimal.NewFromFloat(0.5)

	t.Run("Transactions", func(t *testing.T) {
		amountUSD := alpacadecimal.NewFromInt(100)

		bookedAt := time.Now()

		var resolvers transactions.Resolvers
		// Step 1: Outstanding USD for Customer
		tx1, err := transactions.IssueCustomerReceivableTemplate{
			At:       bookedAt,
			Amount:   amountUSD,
			Currency: "USD",
		}.Resolve(t.Context(), customer.CustomerID{ID: "123", Namespace: "test"}, resolvers)
		require.NoError(t, err)

		// Step 2: Convert USD to CRD into customer account
		tx2, err := transactions.ConvertCurrencyTemplate{
			At:             bookedAt,
			TargetAmount:   amountUSD,
			CostBasis:      costBasis,
			SourceCurrency: "USD",
			TargetCurrency: "CRD",
		}.Resolve(t.Context(), customer.CustomerID{ID: "123", Namespace: "test"}, resolvers)
		require.NoError(t, err)

		// Step 3: Cover outstanding CRD from customer account
		tx3, err := transactions.CoverCustomerReceivableTemplate{
			At:       bookedAt,
			Amount:   amountUSD,
			Currency: "CRD",
		}.Resolve(t.Context(), customer.CustomerID{ID: "123", Namespace: "test"}, resolvers)
		require.NoError(t, err)

		// tx1, tx2 & tx3 should be written to the ledger AT THE SAME TIME
		_, err = l.CommitGroup(t.Context(), asTxGroupInput([]ledger.TransactionInput{tx1, tx2, tx3}))
		require.NoError(t, err)
	})
}
