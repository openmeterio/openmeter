package customerbalance

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	ledgerhistorical "github.com/openmeterio/openmeter/openmeter/ledger/historical"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestLedgerCreditMovement_AdjustedReturnsEmpty(t *testing.T) {
	txType := CreditTransactionTypeAdjusted

	movement, empty, err := ledgerCreditMovement(&txType)

	require.NoError(t, err)
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

func TestCreditTransactionFromLedgerTransaction_UsesFBOEntry(t *testing.T) {
	usd := currencyx.Code("USD")
	tx := mustHistoricalTransaction(t, []ledgerhistorical.EntryData{
		mustEntryData(t, "entry-usd", ledger.AccountTypeCustomerFBO, usd, alpacadecimal.NewFromInt(-10)),
		mustEntryData(t, "entry-accrued", ledger.AccountTypeCustomerAccrued, usd, alpacadecimal.NewFromInt(10)),
	})

	item, err := creditTransactionFromLedgerTransaction(tx)
	require.NoError(t, err)
	require.Equal(t, CreditTransactionTypeConsumed, item.Type)
	require.Equal(t, currencyx.Code("USD"), item.Currency)
	require.True(t, item.Amount.Equal(alpacadecimal.NewFromInt(-10)))
}

func TestApplyCreditTransactionBalances(t *testing.T) {
	items := []CreditTransaction{
		{
			Amount: alpacadecimal.NewFromInt(-10),
		},
	}

	applyCreditTransactionBalances(items, alpacadecimal.NewFromInt(42))

	require.True(t, items[0].Balance.After.Equal(alpacadecimal.NewFromInt(42)))
	require.True(t, items[0].Balance.Before.Equal(alpacadecimal.NewFromInt(52)))
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
