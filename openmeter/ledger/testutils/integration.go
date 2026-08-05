package testutils

import (
	"crypto/sha256"
	"fmt"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	ledgerentrydb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgerentry"
	ledgertransactiondb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	omtestutils "github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type IntegrationEnv struct {
	Namespace  string
	CustomerID customer.CustomerID
	Currency   currencyx.Code
	// CustomCurrency overrides the resolved custom currency used by routes.
	// Tests that only set Currency get a deterministic definition by default.
	CustomCurrency   *currencies.Currency
	DB               *entdb.Client
	CustomerAccounts ledger.CustomerAccounts
	BusinessAccounts ledger.BusinessAccounts
	Deps             Deps
}

func (e *IntegrationEnv) CustomCurrencyForRoute(currency currencyx.Code) currencies.CurrencyReference {
	if e.CustomCurrency != nil {
		return e.CustomCurrency.Reference()
	}
	if !currency.IsCustom() {
		return currencies.NewCurrencyReference(currency)
	}

	id := fmt.Sprintf("%x", sha256.Sum256([]byte(currency)))[:26]
	resolved, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeCustom).
		WithCode(currency).
		WithName(currency.String()).
		WithPrecision(4).
		Build()
	if err != nil {
		panic(err)
	}

	return (currencies.Currency{
		NamespacedID: models.NamespacedID{
			Namespace: e.Namespace,
			ID:        id,
		},
		Currency: resolved,
	}).Reference()
}

func (e *IntegrationEnv) CurrencyReference() currencies.CurrencyReference {
	return e.CustomCurrencyForRoute(e.Currency)
}

func NewIntegrationEnv(t *testing.T, namespacePrefix string) *IntegrationEnv {
	t.Helper()

	now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	t.Cleanup(clock.UnFreeze)

	testDB := omtestutils.InitPostgresDB(t, omtestutils.PostgresDBStateAtlasMigrated)
	t.Cleanup(func() {
		require.NoError(t, testDB.EntDriver.Close())
		require.NoError(t, testDB.PGDriver.Close())
	})

	namespace := fmt.Sprintf("%s-%d", namespacePrefix, clock.Now().UnixNano())

	db := testDB.EntDriver.Client()
	deps, err := InitDeps(db, omtestutils.NewDiscardLogger(t))
	require.NoError(t, err)

	customerID := customer.CustomerID{
		Namespace: namespace,
		ID:        ulid.Make().String(),
	}

	_, err = db.Customer.Create().
		SetNamespace(namespace).
		SetID(customerID.ID).
		SetName("Test Customer").
		Save(t.Context())
	require.NoError(t, err)

	customerAccounts, err := deps.ResolversService.CreateCustomerAccounts(t.Context(), customerID)
	require.NoError(t, err)

	businessAccounts, err := deps.ResolversService.EnsureBusinessAccounts(t.Context(), namespace)
	require.NoError(t, err)

	return &IntegrationEnv{
		Namespace:        namespace,
		CustomerID:       customerID,
		Currency:         currencyx.Code("USD"),
		DB:               db,
		CustomerAccounts: customerAccounts,
		BusinessAccounts: businessAccounts,
		Deps:             deps,
	}
}

func (e *IntegrationEnv) Now() time.Time {
	return clock.Now().UTC()
}

func (e *IntegrationEnv) FBOSubAccount(t *testing.T, priority int) ledger.SubAccount {
	t.Helper()

	subAccount, err := e.CustomerAccounts.FBOAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerFBORouteParams{
		Currency:       e.CurrencyReference(),
		CreditPriority: priority,
	})
	require.NoError(t, err)

	return subAccount
}

func (e *IntegrationEnv) ReceivableSubAccount(t *testing.T) ledger.SubAccount {
	t.Helper()

	return e.ReceivableSubAccountWithCostBasisAndStatus(t, nil, ledger.TransactionAuthorizationStatusOpen)
}

func (e *IntegrationEnv) ReceivableSubAccountWithStatus(t *testing.T, status ledger.TransactionAuthorizationStatus) ledger.SubAccount {
	t.Helper()

	return e.ReceivableSubAccountWithCostBasisAndStatus(t, nil, status)
}

func (e *IntegrationEnv) ReceivableSubAccountWithCostBasis(t *testing.T, costBasis *alpacadecimal.Decimal) ledger.SubAccount {
	t.Helper()

	return e.ReceivableSubAccountWithCostBasisAndStatus(t, costBasis, ledger.TransactionAuthorizationStatusOpen)
}

func (e *IntegrationEnv) ReceivableSubAccountWithCostBasisAndStatus(t *testing.T, costBasis *alpacadecimal.Decimal, status ledger.TransactionAuthorizationStatus) ledger.SubAccount {
	t.Helper()

	subAccount, err := e.CustomerAccounts.ReceivableAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerReceivableRouteParams{
		Currency:                       e.CurrencyReference(),
		CostBasis:                      costBasis,
		TransactionAuthorizationStatus: status,
	})
	require.NoError(t, err)

	return subAccount
}

func (e *IntegrationEnv) AccruedSubAccount(t *testing.T) ledger.SubAccount {
	t.Helper()

	return e.AccruedSubAccountWithCostBasis(t, nil)
}

func (e *IntegrationEnv) AccruedSubAccountWithCostBasis(t *testing.T, costBasis *alpacadecimal.Decimal) ledger.SubAccount {
	t.Helper()

	return e.AccruedSubAccountWithCostBasisAndTaxCode(t, costBasis, nil)
}

func (e *IntegrationEnv) AccruedSubAccountWithCostBasisAndTaxCode(t *testing.T, costBasis *alpacadecimal.Decimal, taxCode *string) ledger.SubAccount {
	t.Helper()

	subAccount, err := e.CustomerAccounts.AccruedAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerAccruedRouteParams{
		Currency:  e.CurrencyReference(),
		TaxCode:   taxCode,
		CostBasis: costBasis,
	})
	require.NoError(t, err)

	return subAccount
}

func (e *IntegrationEnv) WashSubAccount(t *testing.T) ledger.SubAccount {
	t.Helper()

	return e.WashSubAccountWithCostBasis(t, nil)
}

func (e *IntegrationEnv) WashSubAccountWithCostBasis(t *testing.T, costBasis *alpacadecimal.Decimal) ledger.SubAccount {
	t.Helper()

	subAccount, err := e.BusinessAccounts.WashAccount.GetSubAccountForRoute(t.Context(), ledger.BusinessRouteParams{
		Currency:  e.CurrencyReference(),
		CostBasis: costBasis,
	})
	require.NoError(t, err)

	return subAccount
}

func (e *IntegrationEnv) EarningsSubAccount(t *testing.T) ledger.SubAccount {
	t.Helper()

	return e.EarningsSubAccountWithCostBasis(t, nil)
}

func (e *IntegrationEnv) EarningsSubAccountWithCostBasis(t *testing.T, costBasis *alpacadecimal.Decimal) ledger.SubAccount {
	t.Helper()

	return e.EarningsSubAccountWithCostBasisAndTaxCode(t, costBasis, nil)
}

func (e *IntegrationEnv) EarningsSubAccountWithCostBasisAndTaxCode(t *testing.T, costBasis *alpacadecimal.Decimal, taxCode *string) ledger.SubAccount {
	t.Helper()

	subAccount, err := e.BusinessAccounts.EarningsAccount.GetSubAccountForRoute(t.Context(), ledger.BusinessRouteParams{
		Currency:  e.CurrencyReference(),
		CostBasis: costBasis,
		TaxCode:   taxCode,
	})
	require.NoError(t, err)

	return subAccount
}

func (e *IntegrationEnv) BrokerageSubAccount(t *testing.T) ledger.SubAccount {
	t.Helper()

	subAccount, err := e.BusinessAccounts.BrokerageAccount.GetSubAccountForRoute(t.Context(), ledger.BusinessRouteParams{
		Currency: e.CurrencyReference(),
	})
	require.NoError(t, err)

	return subAccount
}

func (e *IntegrationEnv) BreakageSubAccountWithCostBasis(t *testing.T, costBasis *alpacadecimal.Decimal) ledger.SubAccount {
	t.Helper()

	subAccount, err := e.BusinessAccounts.BreakageAccount.GetSubAccountForRoute(t.Context(), ledger.BusinessRouteParams{
		Currency:  e.CurrencyReference(),
		CostBasis: costBasis,
	})
	require.NoError(t, err)

	return subAccount
}

func (e *IntegrationEnv) SumBalance(t *testing.T, subAccount ledger.SubAccount) alpacadecimal.Decimal {
	t.Helper()

	sum, err := e.Deps.HistoricalLedger.GetSubAccountBalance(t.Context(), subAccount, ledger.BalanceQuery{})
	require.NoError(t, err)

	return sum
}

// TransactionGroupEntries returns every ledger entry posted by the
// transactions in a committed group, ordered by creation.
func (e *IntegrationEnv) TransactionGroupEntries(t *testing.T, groupID string) []*entdb.LedgerEntry {
	t.Helper()

	ledgerTransactions, err := e.DB.LedgerTransaction.Query().
		Where(
			ledgertransactiondb.Namespace(e.Namespace),
			ledgertransactiondb.GroupID(groupID),
		).
		Order(
			ledgertransactiondb.ByCreatedAt(),
			ledgertransactiondb.ByID(),
		).
		All(t.Context())
	require.NoError(t, err)
	require.NotEmpty(t, ledgerTransactions)

	transactionIDs := make([]string, 0, len(ledgerTransactions))
	for _, transaction := range ledgerTransactions {
		transactionIDs = append(transactionIDs, transaction.ID)
	}

	entries, err := e.DB.LedgerEntry.Query().
		Where(
			ledgerentrydb.Namespace(e.Namespace),
			ledgerentrydb.TransactionIDIn(transactionIDs...),
		).
		Order(
			ledgerentrydb.ByCreatedAt(),
			ledgerentrydb.ByID(),
		).
		All(t.Context())
	require.NoError(t, err)
	require.NotEmpty(t, entries)

	return entries
}

// EntriesForSubAccount filters a set of ledger entries down to the ones
// posted against a specific sub-account.
func (e *IntegrationEnv) EntriesForSubAccount(entries []*entdb.LedgerEntry, subAccount ledger.SubAccount) []*entdb.LedgerEntry {
	out := make([]*entdb.LedgerEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.SubAccountID == subAccount.Address().SubAccountID() {
			out = append(out, entry)
		}
	}

	return out
}

// FBOSubAccountForCurrency is the explicit-currency counterpart of
// FBOSubAccount, for tests that operate on a currency other than the env's
// own (e.g. a custom currency alongside its fiat cost-basis currency).
func (e *IntegrationEnv) FBOSubAccountForCurrency(t *testing.T, currency currencies.CurrencyReference, costBasisCurrency *currencyx.Code, costBasis alpacadecimal.Decimal) ledger.SubAccount {
	t.Helper()

	subAccount, err := e.CustomerAccounts.FBOAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerFBORouteParams{
		Currency:          currency,
		CostBasisCurrency: costBasisCurrency,
		CostBasis:         &costBasis,
		CreditPriority:    ledger.DefaultCustomerFBOPriority,
	})
	require.NoError(t, err)

	return subAccount
}

// AccruedSubAccountForCurrency is the explicit-currency counterpart of
// AccruedSubAccountWithCostBasisAndTaxCode.
func (e *IntegrationEnv) AccruedSubAccountForCurrency(t *testing.T, currency currencies.CurrencyReference, costBasisCurrency *currencyx.Code, costBasis *alpacadecimal.Decimal, taxCode *string) ledger.SubAccount {
	t.Helper()

	subAccount, err := e.CustomerAccounts.AccruedAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerAccruedRouteParams{
		Currency:          currency,
		CostBasisCurrency: costBasisCurrency,
		TaxCode:           taxCode,
		CostBasis:         costBasis,
	})
	require.NoError(t, err)

	return subAccount
}

// ReceivableSubAccountForCurrency is the explicit-currency, open-status
// counterpart of ReceivableSubAccountWithCostBasis.
func (e *IntegrationEnv) ReceivableSubAccountForCurrency(t *testing.T, currency currencies.CurrencyReference, costBasisCurrency *currencyx.Code, costBasis alpacadecimal.Decimal) ledger.SubAccount {
	t.Helper()

	subAccount, err := e.CustomerAccounts.ReceivableAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerReceivableRouteParams{
		Currency:                       currency,
		CostBasisCurrency:              costBasisCurrency,
		CostBasis:                      &costBasis,
		TransactionAuthorizationStatus: ledger.TransactionAuthorizationStatusOpen,
	})
	require.NoError(t, err)

	return subAccount
}

// BrokerageSubAccountForCurrency is the explicit-currency counterpart of
// BrokerageSubAccount. Passing a nil costBasisCurrency serves a fiat
// brokerage route; passing the fiat cost-basis currency alongside a custom
// currency reference serves that custom currency's brokerage route.
func (e *IntegrationEnv) BrokerageSubAccountForCurrency(t *testing.T, currency currencies.CurrencyReference, costBasisCurrency *currencyx.Code, costBasis alpacadecimal.Decimal) ledger.SubAccount {
	t.Helper()

	subAccount, err := e.BusinessAccounts.BrokerageAccount.GetSubAccountForRoute(t.Context(), ledger.BusinessRouteParams{
		Currency:          currency,
		CostBasisCurrency: costBasisCurrency,
		CostBasis:         &costBasis,
	})
	require.NoError(t, err)

	return subAccount
}
