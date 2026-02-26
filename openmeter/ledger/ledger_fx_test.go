package ledger_test

import (
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	"github.com/openmeterio/openmeter/openmeter/ledger/testutil"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/tools/migrate"
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

	// === Build stack via wire ===
	deps, err := testutil.InitDeps(dbClient, slog.Default())
	require.NoError(t, err)

	acctSvc := deps.AccountService
	resolversSvc := deps.ResolversService
	histLedger := deps.HistoricalLedger

	// === Provision currency dimensions ===
	_, err = acctSvc.CreateDimension(ctx, ledgeraccount.CreateDimensionInput{
		Namespace:    namespace,
		Key:          string(ledger.DimensionKeyCurrency),
		Value:        "USD",
		DisplayValue: "US Dollar",
	})
	require.NoError(t, err)

	_, err = acctSvc.CreateDimension(ctx, ledgeraccount.CreateDimensionInput{
		Namespace:    namespace,
		Key:          string(ledger.DimensionKeyCurrency),
		Value:        "CRD",
		DisplayValue: "Credits",
	})
	require.NoError(t, err)

	// === Provision customer accounts ===
	customerID := customer.CustomerID{
		Namespace: namespace,
		ID:        "test-customer-01",
	}
	_, err = resolversSvc.CreateCustomerAccounts(ctx, customerID)
	require.NoError(t, err)

	// We're simulating the scenario where we're effectively converting
	// an outstanding CRD balance to an outstanding FIAT balance.
	t.Run("Transactions", func(t *testing.T) {
		amountUSD := alpacadecimal.NewFromInt(100)
		costBasis := alpacadecimal.NewFromFloat(0.5)
		bookedAt := time.Now()

		deps := transactions.ResolverDependencies{
			AccountService: resolversSvc,
			DimensionService: &ledgeraccount.DimensionResolver{
				Namespace: namespace,
				Service:   acctSvc,
			},
		}

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
