package customerscredits

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	"github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
	ledgerhistorical "github.com/openmeterio/openmeter/openmeter/ledger/historical"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestCreditMovementFromTypeFilter_AdjustedReturnsEmpty(t *testing.T) {
	filter := api.BillingCreditTransactionTypeAdjusted

	movement, empty := creditMovementFromTypeFilter(&filter)

	require.Equal(t, ledger.ListTransactionsCreditMovementUnspecified, movement)
	require.True(t, empty)
}

func TestFBOAccountIDFromCustomerAccounts_ReturnsOnlyFBO(t *testing.T) {
	fbo := mustCustomerFBOAccount(t, "ns", "fbo-account")
	receivable := mustCustomerReceivableAccount(t, "ns", "receivable-account")
	accrued := mustCustomerAccruedAccount(t, "ns", "accrued-account")

	accountID := fboAccountIDFromCustomerAccounts(ledger.CustomerAccounts{
		FBOAccount:        fbo,
		ReceivableAccount: receivable,
		AccruedAccount:    accrued,
	})

	require.Equal(t, "fbo-account", accountID)
}

func TestCustomerFBOBalance_UsesCurrencyAndCursor(t *testing.T) {
	usd := currencyx.Code("USD")
	cursor := &ledger.TransactionCursor{
		BookedAt:  time.Date(2026, 4, 10, 9, 0, 0, 0, time.UTC),
		CreatedAt: time.Date(2026, 4, 10, 9, 0, 1, 0, time.UTC),
		ID: models.NamespacedID{
			Namespace: "ns",
			ID:        "tx-1",
		},
	}
	facade := &capturingBalanceFacade{
		balance: alpacadecimal.NewFromInt(42),
	}
	h := handler{
		balanceFacade: facade,
	}

	total, err := h.customerFBOBalance(t.Context(), ListCreditTransactionsRequest{
		Namespace:      "ns",
		CustomerID:     "customer-1",
		CurrencyFilter: &usd,
	}, usd, cursor)
	require.NoError(t, err)
	require.True(t, total.Equal(alpacadecimal.NewFromInt(42)))
	require.Equal(t, usd, facade.lastBalanceInput.Currency)
	require.Equal(t, cursor, facade.lastBalanceInput.After)
}

func TestMapCreditTransaction_UsesFBOEntry(t *testing.T) {
	usd := currencyx.Code("USD")
	tx := mustHistoricalTransaction(t, []ledgerhistorical.EntryData{
		mustEntryData(t, "entry-usd", ledger.AccountTypeCustomerFBO, usd, alpacadecimal.NewFromInt(-10)),
		mustEntryData(t, "entry-accrued", ledger.AccountTypeCustomerAccrued, usd, alpacadecimal.NewFromInt(10)),
	})

	item, err := mapCreditTransaction(tx)
	require.NoError(t, err)
	require.Equal(t, api.BillingCreditTransactionTypeConsumed, item.API.Type)
	require.Equal(t, api.BillingCurrencyCode("USD"), item.API.Currency)
	require.Equal(t, api.Numeric("-10"), item.API.Amount)
	require.True(t, item.Amount.Equal(alpacadecimal.NewFromInt(-10)))
}

func TestApplyCreditTransactionBalances(t *testing.T) {
	items := []mappedCreditTransaction{
		{
			API: api.BillingCreditTransaction{
				Amount: api.Numeric("-10"),
			},
			Amount: alpacadecimal.NewFromInt(-10),
		},
	}

	applyCreditTransactionBalances(items, alpacadecimal.NewFromInt(42))

	require.Equal(t, api.Numeric("42"), items[0].API.AvailableBalance.After)
	require.Equal(t, api.Numeric("52"), items[0].API.AvailableBalance.Before)
}

type capturingBalanceFacade struct {
	lastBalanceInput customerbalance.GetBalanceInput
	balance          alpacadecimal.Decimal
}

func (c *capturingBalanceFacade) GetBalance(_ context.Context, input customerbalance.GetBalanceInput) (alpacadecimal.Decimal, error) {
	c.lastBalanceInput = input
	return c.balance, nil
}

func (c *capturingBalanceFacade) GetBalances(_ context.Context, _ customerbalance.GetBalancesInput) ([]customerbalance.BalanceByCurrency, error) {
	return nil, nil
}

func mustCustomerFBOAccount(t *testing.T, namespace, id string) *ledgeraccount.CustomerFBOAccount {
	t.Helper()

	account := mustAccount(t, namespace, id, ledger.AccountTypeCustomerFBO)
	fbo, err := account.AsCustomerFBOAccount()
	require.NoError(t, err)

	return fbo
}

func mustCustomerReceivableAccount(t *testing.T, namespace, id string) *ledgeraccount.CustomerReceivableAccount {
	t.Helper()

	account := mustAccount(t, namespace, id, ledger.AccountTypeCustomerReceivable)
	receivable, err := account.AsCustomerReceivableAccount()
	require.NoError(t, err)

	return receivable
}

func mustCustomerAccruedAccount(t *testing.T, namespace, id string) *ledgeraccount.CustomerAccruedAccount {
	t.Helper()

	account := mustAccount(t, namespace, id, ledger.AccountTypeCustomerAccrued)
	accrued, err := account.AsCustomerAccruedAccount()
	require.NoError(t, err)

	return accrued
}

func mustAccount(t *testing.T, namespace, id string, accountType ledger.AccountType) *ledgeraccount.Account {
	t.Helper()

	account, err := ledgeraccount.NewAccountFromData(ledgeraccount.AccountData{
		ID: models.NamespacedID{
			Namespace: namespace,
			ID:        id,
		},
		AccountType: accountType,
	}, ledgeraccount.AccountLiveServices{})
	require.NoError(t, err)

	return account
}

func mustHistoricalTransaction(t *testing.T, entries []ledgerhistorical.EntryData) ledger.Transaction {
	t.Helper()

	tx, err := ledgerhistorical.NewTransactionFromData(ledgerhistorical.TransactionData{
		ID:        "tx-1",
		Namespace: "ns",
		CreatedAt: time.Now().UTC(),
		BookedAt:  time.Now().UTC(),
	}, entries)
	require.NoError(t, err)

	return tx
}

func mustEntryData(t *testing.T, id string, accountType ledger.AccountType, currency currencyx.Code, amount alpacadecimal.Decimal) ledgerhistorical.EntryData {
	t.Helper()

	route := ledger.Route{Currency: currency}
	key, err := ledger.BuildRoutingKey(ledger.RoutingKeyVersionV1, route)
	require.NoError(t, err)

	return ledgerhistorical.EntryData{
		ID:            id,
		Namespace:     "ns",
		CreatedAt:     time.Now().UTC(),
		SubAccountID:  id + "-subaccount",
		AccountType:   accountType,
		Route:         route,
		RouteID:       id + "-route",
		RouteKey:      key.Value(),
		RouteKeyVer:   key.Version(),
		Amount:        amount,
		TransactionID: "tx-1",
	}
}
