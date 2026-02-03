package ledger_test

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/stretchr/testify/require"
)

// The point of this code is to test the primitive APIs with a more complex flow

type testTransactionGroup struct {
	transactions []ledger.Transaction
}

var _ ledger.TransactionGroup = testTransactionGroup{}

func (t testTransactionGroup) Transactions() []ledger.Transaction {
	return t.transactions
}

func (t testTransactionGroup) Annotations() models.Annotations {
	return nil
}

func asTxGroup(transactions []ledger.Transaction) ledger.TransactionGroup {
	return testTransactionGroup{transactions: transactions}
}

func TestFXOnInvoiceIssued(t *testing.T) {
	t.Skipf("This is just to assert the types, it would fail on unimplemented")
	// We're simulating the scenario where we're effective converting an outstanding CRD balance to an outstanding FIAT balance

	// Let's define some mocks to see a more realistic control flow
	type customerAccounts struct {
		USD            ledger.Account
		CRD            ledger.Account
		USDOutstanding ledger.Account
		CRDOutstanding ledger.Account
	}

	getCustomerAccounts := func(c customer.Customer) (customerAccounts, error) {
		return customerAccounts{}, nil
	}

	var l ledger.Ledger
	var c customer.Customer

	var BRKUSD ledger.Account
	var BRKCRD ledger.Account

	var costBasis alpacadecimal.Decimal = alpacadecimal.NewFromFloat(0.5)

	t.Run("Transactions", func(t *testing.T) {
		amountUSD := alpacadecimal.NewFromInt(100)

		bookedAt := time.Now()

		custAccs, err := getCustomerAccounts(c)
		require.NoError(t, err)

		// Step 1: Outstanding USD for Customer
		tx1, err := l.SetUpTransaction(t.Context(), bookedAt, []ledger.LedgerEntryInput{
			exampleEntryInput{
				account: custAccs.USDOutstanding.Address(),
				amount:  amountUSD,
				typ:     ledger.EntryTypeCredit,
			},
			exampleEntryInput{
				account: custAccs.USD.Address(),
				amount:  amountUSD,
				typ:     ledger.EntryTypeDebit,
			},
		})
		require.NoError(t, err)
		require.NotNil(t, tx1)

		// Step 2: Convert USD to CRD into customer account
		tx2, err := l.SetUpTransaction(t.Context(), bookedAt, []ledger.LedgerEntryInput{
			// USD entries
			exampleEntryInput{
				account: custAccs.USD.Address(),
				amount:  amountUSD,
				typ:     ledger.EntryTypeCredit,
			},
			exampleEntryInput{
				account: BRKUSD.Address(),
				amount:  amountUSD,
				typ:     ledger.EntryTypeDebit,
			},
			// CRD entries
			exampleEntryInput{
				account: BRKCRD.Address(),
				amount:  amountUSD.Mul(costBasis),
				typ:     ledger.EntryTypeCredit,
			},
			exampleEntryInput{
				account: custAccs.CRD.Address(),
				amount:  amountUSD.Mul(costBasis),
				typ:     ledger.EntryTypeDebit,
			},
		})
		require.NoError(t, err)
		require.NotNil(t, tx2)

		// Step 3: Cover outstanding CRD from customer account
		tx3, err := l.SetUpTransaction(t.Context(), bookedAt, []ledger.LedgerEntryInput{
			exampleEntryInput{
				account: custAccs.CRDOutstanding.Address(),
				amount:  amountUSD.Mul(costBasis),
				typ:     ledger.EntryTypeDebit,
			},
			exampleEntryInput{
				account: custAccs.CRD.Address(),
				amount:  amountUSD.Mul(costBasis),
				typ:     ledger.EntryTypeCredit,
			},
		})
		require.NoError(t, err)
		require.NotNil(t, tx3)

		// tx1, tx2 & tx3 should be written to the ledger AT THE SAME TIME
		err = l.CommitGroup(t.Context(), asTxGroup([]ledger.Transaction{tx1, tx2, tx3}))
		require.NoError(t, err)
	})
}
