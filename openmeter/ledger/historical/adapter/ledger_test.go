package adapter

import (
	"fmt"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	ledgerentrydb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgerentry"
	ledgertransactiondb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgertransaction"
	ledgertransactiongroupdb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgertransactiongroup"
	ledger "github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	ledgerhistorical "github.com/openmeterio/openmeter/openmeter/ledger/historical"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	"github.com/openmeterio/openmeter/tools/migrate"
)

type testEntryInput struct {
	address ledger.PostingAddress
	amount  alpacadecimal.Decimal
}

func (e testEntryInput) PostingAddress() ledger.PostingAddress {
	return e.address
}

func (e testEntryInput) Amount() alpacadecimal.Decimal {
	return e.amount
}

var _ ledger.EntryInput = (*testEntryInput)(nil)

func TestRepo_CreateTransactionGroup(t *testing.T) {
	env := NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})
	env.DBSchemaMigrate(t)

	ctx := t.Context()
	namespace := testNamespace()
	annotations := models.Annotations{"source": "adapter-test"}

	group, err := env.repo.CreateTransactionGroup(ctx, ledgerhistorical.CreateTransactionGroupInput{
		Namespace:   namespace,
		Annotations: annotations,
	})
	require.NoError(t, err)
	require.Equal(t, namespace, group.Namespace)
	require.Equal(t, annotations, group.Annotations)

	entity, err := env.client.LedgerTransactionGroup.Query().
		Where(
			ledgertransactiongroupdb.Namespace(namespace),
			ledgertransactiongroupdb.ID(group.ID),
		).
		Only(ctx)
	require.NoError(t, err)
	require.Equal(t, annotations, entity.Annotations)
}

func TestRepo_BookTransaction_CreatesTransactionAndEntries(t *testing.T) {
	env := NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})
	env.DBSchemaMigrate(t)

	ctx := t.Context()
	namespace := testNamespace()
	subAccountA := env.createSubAccount(t, namespace, "acc-a")
	subAccountB := env.createSubAccount(t, namespace, "acc-b")

	hLedger := &ledgerhistorical.Ledger{}
	txInputIntf, err := hLedger.SetUpTransactionInput(ctx, time.Now().UTC(), []ledger.EntryInput{
		testEntryInput{
			address: ledgeraccount.NewAddressFromData(ledgeraccount.AddressData{
				SubAccountID: subAccountA,
				AccountType:  ledger.AccountTypeCustomerFBO,
			}),
			amount: alpacadecimal.NewFromInt(-100),
		},
		testEntryInput{
			address: ledgeraccount.NewAddressFromData(ledgeraccount.AddressData{
				SubAccountID: subAccountB,
				AccountType:  ledger.AccountTypeCustomerFBO,
			}),
			amount: alpacadecimal.NewFromInt(100),
		},
	})
	require.NoError(t, err)

	txInput, ok := txInputIntf.(*ledgerhistorical.TransactionInput)
	require.True(t, ok)

	group, err := env.repo.CreateTransactionGroup(ctx, ledgerhistorical.CreateTransactionGroupInput{
		Namespace: namespace,
	})
	require.NoError(t, err)

	tx, err := env.repo.BookTransaction(ctx, models.NamespacedID{
		Namespace: namespace,
		ID:        group.ID,
	}, txInput)
	require.NoError(t, err)
	require.NotNil(t, tx)

	transactions, err := env.client.LedgerTransaction.Query().
		Where(
			ledgertransactiondb.Namespace(namespace),
			ledgertransactiondb.GroupID(group.ID),
		).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, transactions, 1)

	entries, err := env.client.LedgerEntry.Query().
		Where(
			ledgerentrydb.Namespace(namespace),
			ledgerentrydb.TransactionID(transactions[0].ID),
		).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, entries, 2)

	subAccountIDs := lo.Map(entries, func(e *entdb.LedgerEntry, _ int) string {
		return e.SubAccountID
	})
	require.Contains(t, subAccountIDs, subAccountA)
	require.Contains(t, subAccountIDs, subAccountB)
}

func TestRepo_BookTransaction_NilInput(t *testing.T) {
	env := NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})
	env.DBSchemaMigrate(t)

	ctx := t.Context()
	namespace := testNamespace()
	group, err := env.repo.CreateTransactionGroup(ctx, ledgerhistorical.CreateTransactionGroupInput{
		Namespace: namespace,
	})
	require.NoError(t, err)

	_, err = env.repo.BookTransaction(ctx, models.NamespacedID{
		Namespace: namespace,
		ID:        group.ID,
	}, nil)
	require.Error(t, err)
	require.ErrorContains(t, err, "transaction input is required")
}

func TestRepo_SumEntries_Filters(t *testing.T) {
	env := NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})
	env.DBSchemaMigrate(t)

	ctx := t.Context()
	namespace := testNamespace()

	currencyUSD := env.createDimension(t, namespace, string(ledger.DimensionKeyCurrency), "currency-usd", "USD")
	currencyEUR := env.createDimension(t, namespace, string(ledger.DimensionKeyCurrency), "currency-eur", "EUR")
	taxA := env.createDimension(t, namespace, string(ledger.DimensionKeyTaxCode), "tax-a", "TAX-A")
	taxB := env.createDimension(t, namespace, string(ledger.DimensionKeyTaxCode), "tax-b", "TAX-B")
	featureA := env.createDimension(t, namespace, string(ledger.DimensionKeyFeature), "feature-a", "FEATURE-A")
	featureB := env.createDimension(t, namespace, string(ledger.DimensionKeyFeature), "feature-b", "FEATURE-B")
	creditPriority1 := env.createDimension(t, namespace, string(ledger.DimensionKeyCreditPriority), "1", "1")
	creditPriority2 := env.createDimension(t, namespace, string(ledger.DimensionKeyCreditPriority), "2", "2")

	subAccountA := env.createSubAccountWithDimensions(t, namespace, "acc-a", currencyUSD, &taxA, &featureA, &creditPriority1)
	subAccountB := env.createSubAccountWithDimensions(t, namespace, "acc-b", currencyUSD, &taxB, &featureB, &creditPriority2)
	subAccountC := env.createSubAccountWithDimensions(t, namespace, "acc-c", currencyEUR, &taxA, &featureA, &creditPriority1)

	group, err := env.repo.CreateTransactionGroup(ctx, ledgerhistorical.CreateTransactionGroupInput{Namespace: namespace})
	require.NoError(t, err)

	hLedger := &ledgerhistorical.Ledger{}
	bookedAtEarly := time.Now().UTC().Add(-2 * time.Hour)
	txInputEarlyIntf, err := hLedger.SetUpTransactionInput(ctx, bookedAtEarly, []ledger.EntryInput{
		testEntryInput{
			address: ledgeraccount.NewAddressFromData(ledgeraccount.AddressData{SubAccountID: subAccountA, AccountType: ledger.AccountTypeCustomerFBO}),
			amount:  alpacadecimal.NewFromInt(100),
		},
		testEntryInput{
			address: ledgeraccount.NewAddressFromData(ledgeraccount.AddressData{SubAccountID: subAccountB, AccountType: ledger.AccountTypeCustomerFBO}),
			amount:  alpacadecimal.NewFromInt(-100),
		},
	})
	require.NoError(t, err)
	txEarly, err := env.repo.BookTransaction(ctx, models.NamespacedID{Namespace: namespace, ID: group.ID}, txInputEarlyIntf.(*ledgerhistorical.TransactionInput))
	require.NoError(t, err)

	bookedAtLate := time.Now().UTC().Add(-30 * time.Minute)
	txInputLateIntf, err := hLedger.SetUpTransactionInput(ctx, bookedAtLate, []ledger.EntryInput{
		testEntryInput{
			address: ledgeraccount.NewAddressFromData(ledgeraccount.AddressData{SubAccountID: subAccountA, AccountType: ledger.AccountTypeCustomerFBO}),
			amount:  alpacadecimal.NewFromInt(50),
		},
		testEntryInput{
			address: ledgeraccount.NewAddressFromData(ledgeraccount.AddressData{SubAccountID: subAccountC, AccountType: ledger.AccountTypeCustomerFBO}),
			amount:  alpacadecimal.NewFromInt(-50),
		},
	})
	require.NoError(t, err)
	_, err = env.repo.BookTransaction(ctx, models.NamespacedID{Namespace: namespace, ID: group.ID}, txInputLateIntf.(*ledgerhistorical.TransactionInput))
	require.NoError(t, err)

	sumUSD, err := env.repo.SumEntries(ctx, ledger.Query{
		Namespace: namespace,
		Filters: ledger.Filters{
			Dimensions: ledger.QueryDimensions{CurrencyID: currencyUSD},
		},
	})
	require.NoError(t, err)
	require.True(t, sumUSD.Equal(alpacadecimal.NewFromInt(50)))

	sumTaxA, err := env.repo.SumEntries(ctx, ledger.Query{
		Namespace: namespace,
		Filters: ledger.Filters{
			Dimensions: ledger.QueryDimensions{
				CurrencyID: currencyUSD,
				TaxCodeID:  &taxA,
			},
		},
	})
	require.NoError(t, err)
	require.True(t, sumTaxA.Equal(alpacadecimal.NewFromInt(150)))

	sumFeatureA, err := env.repo.SumEntries(ctx, ledger.Query{
		Namespace: namespace,
		Filters: ledger.Filters{
			Dimensions: ledger.QueryDimensions{
				CurrencyID: currencyUSD,
				FeatureIDs: []string{featureA},
			},
		},
	})
	require.NoError(t, err)
	require.True(t, sumFeatureA.Equal(alpacadecimal.NewFromInt(150)))

	creditPriority := 1
	sumPriority, err := env.repo.SumEntries(ctx, ledger.Query{
		Namespace: namespace,
		Filters: ledger.Filters{
			Dimensions: ledger.QueryDimensions{
				CurrencyID:     currencyUSD,
				CreditPriority: &creditPriority,
			},
		},
	})
	require.NoError(t, err)
	require.True(t, sumPriority.Equal(alpacadecimal.NewFromInt(150)))

	txID := txEarly.ID().ID
	sumTxID, err := env.repo.SumEntries(ctx, ledger.Query{
		Namespace: namespace,
		Filters: ledger.Filters{
			TransactionID: &txID,
			Dimensions:    ledger.QueryDimensions{CurrencyID: currencyUSD},
		},
	})
	require.NoError(t, err)
	require.True(t, sumTxID.Equal(alpacadecimal.NewFromInt(0)))

	from := bookedAtLate.Add(-1 * time.Minute)
	sumLate, err := env.repo.SumEntries(ctx, ledger.Query{
		Namespace: namespace,
		Filters: ledger.Filters{
			BookedAtPeriod: &timeutil.OpenPeriod{From: &from},
			Dimensions:     ledger.QueryDimensions{CurrencyID: currencyUSD},
		},
	})
	require.NoError(t, err)
	require.True(t, sumLate.Equal(alpacadecimal.NewFromInt(50)))
}

func TestSumEntriesQuery_SQL(t *testing.T) {
	bookedFrom := time.Now().UTC().Add(-1 * time.Hour)
	txID := "01TESTTXID1234567890123456"
	taxCodeID := "01TESTTAX1234567890123456"
	creditPriority := 7

	q := sumEntriesQuery{
		query: ledger.Query{
			Namespace: "ns-test",
			Filters: ledger.Filters{
				TransactionID: &txID,
				BookedAtPeriod: &timeutil.OpenPeriod{
					From: &bookedFrom,
				},
				Dimensions: ledger.QueryDimensions{
					CurrencyID:     "01TESTCUR1234567890123456",
					TaxCodeID:      &taxCodeID,
					FeatureIDs:     []string{"01TESTFEAT123456789012345"},
					CreditPriority: &creditPriority,
				},
			},
		},
	}

	sqlStr, args := q.SQL()

	require.Equal(t, `SELECT SUM("ledger_entries"."amount") AS "sum_amount" FROM "ledger_entries" WHERE (("ledger_entries"."namespace" = $1 AND "ledger_entries"."transaction_id" = $2) AND EXISTS (SELECT "ledger_transactions"."id" FROM "ledger_transactions" WHERE "ledger_entries"."transaction_id" = "ledger_transactions"."id" AND "ledger_transactions"."booked_at" >= $3)) AND EXISTS (SELECT "ledger_sub_accounts"."id" FROM "ledger_sub_accounts" WHERE ((("ledger_entries"."sub_account_id" = "ledger_sub_accounts"."id" AND "ledger_sub_accounts"."currency_dimension_id" = $4) AND "ledger_sub_accounts"."tax_code_dimension_id" = $5) AND "ledger_sub_accounts"."features_dimension_id" IN ($6)) AND EXISTS (SELECT "ledger_dimensions"."id" FROM "ledger_dimensions" WHERE ("ledger_sub_accounts"."credit_priority_dimension_id" = "ledger_dimensions"."id" AND "ledger_dimensions"."dimension_key" = $7) AND "ledger_dimensions"."dimension_value" = $8))`, sqlStr)
	require.Equal(t, []any{
		"ns-test",
		txID,
		bookedFrom,
		"01TESTCUR1234567890123456",
		taxCodeID,
		"01TESTFEAT123456789012345",
		string(ledger.DimensionKeyCreditPriority),
		"7",
	}, args)
}

type TestEnv struct {
	repo   ledgerhistorical.Repo
	client *entdb.Client
	db     *testutils.TestDB
}

func NewTestEnv(t *testing.T) *TestEnv {
	t.Helper()

	db := testutils.InitPostgresDB(t)
	client := db.EntDriver.Client()

	return &TestEnv{
		repo:   NewRepo(client),
		client: client,
		db:     db,
	}
}

func (e *TestEnv) DBSchemaMigrate(t *testing.T) {
	t.Helper()

	migrator, err := migrate.New(migrate.MigrateOptions{
		ConnectionString: e.db.URL,
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
}

func (e *TestEnv) Close(t *testing.T) {
	t.Helper()

	require.NoError(t, e.client.Close())
	require.NoError(t, e.db.EntDriver.Close())
	require.NoError(t, e.db.PGDriver.Close())
}

func (e *TestEnv) createSubAccount(t *testing.T, namespace string, accountID string) string {
	t.Helper()

	currencyID := e.createDimension(t, namespace, string(ledger.DimensionKeyCurrency), fmt.Sprintf("currency-%d", time.Now().UnixNano()), "USD")
	return e.createSubAccountWithDimensions(t, namespace, accountID, currencyID, nil, nil, nil)
}

func (e *TestEnv) createSubAccountWithDimensions(t *testing.T, namespace string, accountID string, currencyDimensionID string, taxCodeDimensionID *string, featuresDimensionID *string, creditPriorityDimensionID *string) string {
	t.Helper()

	account, err := e.client.LedgerAccount.Create().
		SetNamespace(namespace).
		SetID(accountID).
		SetAccountType(ledger.AccountTypeCustomerFBO).
		Save(t.Context())
	require.NoError(t, err)

	subAccount, err := e.client.LedgerSubAccount.Create().
		SetNamespace(namespace).
		SetAccountID(account.ID).
		SetCurrencyDimensionID(currencyDimensionID).
		SetNillableTaxCodeDimensionID(taxCodeDimensionID).
		SetNillableFeaturesDimensionID(featuresDimensionID).
		SetNillableCreditPriorityDimensionID(creditPriorityDimensionID).
		Save(t.Context())
	require.NoError(t, err)

	return subAccount.ID
}

func (e *TestEnv) createDimension(t *testing.T, namespace, key, value, displayValue string) string {
	t.Helper()

	dimension, err := e.client.LedgerDimension.Create().
		SetNamespace(namespace).
		SetDimensionKey(key).
		SetDimensionValue(value).
		SetDimensionDisplayValue(displayValue).
		Save(t.Context())
	require.NoError(t, err)

	return dimension.ID
}

func testNamespace() string {
	return fmt.Sprintf("ledger-historical-adapter-%d", time.Now().UnixNano())
}
