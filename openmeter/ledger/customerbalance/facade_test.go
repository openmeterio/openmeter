package customerbalance

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	pagepagination "github.com/openmeterio/openmeter/pkg/pagination"
)

func TestFacadeGetBalancesWithExplicitCurrencies(t *testing.T) {
	env := newTestEnv(t)

	env.bookFBOBalanceInCurrency(t, alpacadecimal.NewFromInt(100), "USD")
	env.bookFBOBalanceInCurrency(t, alpacadecimal.NewFromInt(200), "EUR")
	env.createFlatFeeChargeInCurrency(t, alpacadecimal.NewFromInt(30), productcatalog.CreditOnlySettlementMode, env.sp(), "USD")
	env.createFlatFeeChargeInCurrency(t, alpacadecimal.NewFromInt(70), productcatalog.CreditOnlySettlementMode, env.sp(), "EUR")

	facade, err := NewFacade(env.Service)
	require.NoError(t, err)

	balances, err := facade.GetBalances(t.Context(), GetBalancesInput{
		CustomerID: env.CustomerID,
		Currencies: CurrencyFilter{
			Codes: []currencyx.Code{"USD", "EUR"},
		},
	})
	require.NoError(t, err)
	require.Len(t, balances, 2)

	require.Equal(t, currencyx.Code("USD"), balances[0].Currency)
	require.True(t, balances[0].Balance.Settled().Equal(alpacadecimal.NewFromInt(100)))
	require.True(t, balances[0].Balance.Pending().Equal(alpacadecimal.NewFromInt(70)))

	require.Equal(t, currencyx.Code("EUR"), balances[1].Currency)
	require.True(t, balances[1].Balance.Settled().Equal(alpacadecimal.NewFromInt(200)))
	require.True(t, balances[1].Balance.Pending().Equal(alpacadecimal.NewFromInt(130)))
}

func TestFacadeGetBalancesWithDiscoveredCurrencies(t *testing.T) {
	env := newTestEnv(t)

	env.bookFBOBalanceInCurrency(t, alpacadecimal.NewFromInt(100), "USD")
	env.bookFBOBalanceInCurrency(t, alpacadecimal.NewFromInt(200), "EUR")
	env.createFlatFeeChargeInCurrency(t, alpacadecimal.NewFromInt(30), productcatalog.CreditOnlySettlementMode, env.sp(), "USD")
	env.createFlatFeeChargeInCurrency(t, alpacadecimal.NewFromInt(70), productcatalog.CreditOnlySettlementMode, env.sp(), "EUR")
	facade, err := NewFacade(env.Service)
	require.NoError(t, err)

	balances, err := facade.GetBalances(t.Context(), GetBalancesInput{
		CustomerID: env.CustomerID,
	})
	require.NoError(t, err)
	require.Len(t, balances, 2)

	var usdCount, eurCount int
	for _, balance := range balances {
		switch balance.Currency {
		case "USD":
			usdCount++
			require.True(t, balance.Balance.Settled().Equal(alpacadecimal.NewFromInt(100)))
			require.True(t, balance.Balance.Pending().Equal(alpacadecimal.NewFromInt(70)))
		case "EUR":
			eurCount++
			require.True(t, balance.Balance.Settled().Equal(alpacadecimal.NewFromInt(200)))
			require.True(t, balance.Balance.Pending().Equal(alpacadecimal.NewFromInt(130)))
		}
	}

	require.Equal(t, 1, usdCount)
	require.Equal(t, 1, eurCount)
}

func TestFacadeGetBalancesWithUnsupportedExplicitCurrency(t *testing.T) {
	env := newTestEnv(t)

	facade, err := NewFacade(env.Service)
	require.NoError(t, err)

	_, err = facade.GetBalances(t.Context(), GetBalancesInput{
		CustomerID: env.CustomerID,
		Currencies: CurrencyFilter{
			Codes: []currencyx.Code{"CUSTOM"},
		},
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "CUSTOM")
	require.ErrorContains(t, err, "not supported by ledger")
}

func TestFacadeGetBalanceAfterTransactionCursor(t *testing.T) {
	env := newTestEnv(t)
	facade, err := NewFacade(env.Service)
	require.NoError(t, err)

	firstBookedAt := time.Date(2026, 4, 10, 9, 0, 0, 0, time.UTC)
	secondBookedAt := firstBookedAt.Add(time.Minute)

	clock.SetTime(firstBookedAt)
	defer clock.ResetTime()
	env.bookFBOBalance(t, alpacadecimal.NewFromInt(100))

	clock.SetTime(secondBookedAt)
	env.bookFBOBalance(t, alpacadecimal.NewFromInt(20))

	fboAccount, ok := env.CustomerAccounts.FBOAccount.(*ledgeraccount.CustomerFBOAccount)
	require.True(t, ok)

	paged, err := env.Deps.HistoricalLedger.ListTransactionsByPage(t.Context(), ledger.ListTransactionsByPageInput{
		Page:       pagepagination.NewPage(1, 10),
		Namespace:  env.Namespace,
		AccountIDs: []string{fboAccount.ID().ID},
		Currency:   &env.Currency,
	})
	require.NoError(t, err)
	require.Len(t, paged.Items, 2)

	olderTx := paged.Items[1]
	balanceAfterOlderTx, err := facade.GetBalance(t.Context(), GetBalanceInput{
		CustomerID: env.CustomerID,
		Currency:   env.Currency,
		After:      lo.ToPtr(olderTx.Cursor()),
	})
	require.NoError(t, err)
	require.True(t, balanceAfterOlderTx.Equal(alpacadecimal.NewFromInt(100)))

	currentBalance, err := facade.GetBalance(t.Context(), GetBalanceInput{
		CustomerID: env.CustomerID,
		Currency:   env.Currency,
	})
	require.NoError(t, err)
	require.True(t, currentBalance.Equal(alpacadecimal.NewFromInt(120)))
}
