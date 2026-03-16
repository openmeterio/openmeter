package ledgerv2_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledgerv2"
	"github.com/openmeterio/openmeter/openmeter/ledgerv2/account"
	"github.com/openmeterio/openmeter/openmeter/ledgerv2/historical"
	"github.com/openmeterio/openmeter/openmeter/ledgerv2/transactions"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/tools/migrate"
)

func TestFXOnInvoiceIssued(t *testing.T) {
	// === Setup DB ===
	testDB := testutils.InitPostgresDB(t)
	t.Cleanup(func() {
		require.NoError(t, testDB.EntDriver.Close())
		require.NoError(t, testDB.PGDriver.Close())
	})

	migrator, err := migrate.New(migrate.MigrateOptions{
		ConnectionString: testDB.URL,
		Migrations:       migrate.OMMigrationsConfig,
		Logger:           testutils.NewDiscardLogger(t),
	})
	require.NoError(t, err)
	defer func() {
		srcErr, dbErr := migrator.Close()
		require.NoError(t, srcErr)
		require.NoError(t, dbErr)
	}()
	require.NoError(t, migrator.Up())

	ctx := t.Context()
	namespace := fmt.Sprintf("ledger-fx-test-%d", time.Now().UnixNano())
	dbClient := testDB.EntDriver.Client()
	require.NotNil(t, dbClient)

	// === Build stack via wire ===

	var acctSvc account.Service
	var ledger Service
	// var resolversSvc resolvers.AccountResolver
	var histLedger historical.Ledger

	// === Provision currency dimensions ===
	usdCurrency := currencyx.NewCode("USD")
	creditDimension := currencyx.NewCode("CRD") // TODO: how to handle polymorphism here

	// === Provision customer accounts ===
	customerID := customer.CustomerID{
		Namespace: namespace,
		ID:        "test-customer-01",
	}
	customerAccounts1, err := ledger.CreateCustomerAccounts(ctx, customerID)
	require.NoError(t, err)
	customerAccounts2, err := ledger.CreateCustomerAccounts(ctx, customerID)
	require.NoError(t, err)

	// validate sub-accounts (not needed for the test)
	fboSub1, err := acctSvc.GetSubAccountForDimensions(ctx, customerAccounts1.FBOAccount, ledgerv2.CustomerSubAccountDimensions{
		Currency: mo.Some(usdCurrency),
	})

	require.NoError(t, err)
	fboSub2, err := acctSvc.GetSubAccountForDimensions(ctx, customerAccounts2.FBOAccount, ledgerv2.CustomerSubAccountDimensions{
		Currency: mo.Some(usdCurrency),
	})
	require.NoError(t, err)
	require.Equal(t, fboSub1.ID, fboSub2.ID)

	// We're simulating the scenario where we're effectively converting
	// an outstanding CRD balance to an outstanding FIAT balance.
	t.Run("Transactions", func(t *testing.T) {
		amountUSD := alpacadecimal.NewFromInt(100)
		costBasis := alpacadecimal.NewFromFloat(0.5)
		bookedAt := time.Now()

		scope := transactions.ResolutionScope{
			CustomerID: customerID,
			Namespace:  namespace,
		}

		// Step 1: Outstanding USD for Customer
		tx1 := transactions.IssueCustomerReceivableTemplate{
			At:       bookedAt,
			Amount:   amountUSD,
			Currency: "USD",
		}

		// Step 2: Convert USD to CRD into customer account
		tx2 := transactions.ConvertCurrencyTemplate{
			At:             bookedAt,
			TargetAmount:   amountUSD,
			CostBasis:      costBasis,
			SourceCurrency: "USD",
			TargetCurrency: "CRD",
		}

		// Step 3: Cover outstanding CRD from customer account
		tx3 := transactions.CoverCustomerReceivableTemplate{
			At:       bookedAt,
			Amount:   amountUSD,
			Currency: "CRD",
		}

		inputs, err := transactions.ResolveTransactions(ctx, deps, scope, tx1, tx2, tx3)
		require.NoError(t, err)
		require.Len(t, inputs, 3)

		// tx1, tx2 & tx3 should be written to the ledger AT THE SAME TIME
		_, err = histLedger.CommitGroup(ctx, transactions.GroupInputs(namespace, nil, inputs...))
		require.NoError(t, err)
	})
}
