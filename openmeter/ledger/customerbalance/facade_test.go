package customerbalance

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestFacadeGetBalancesWithExplicitCurrencies(t *testing.T) {
	env := newTestEnv(t)

	env.bookFBOBalanceInCurrency(t, alpacadecimal.NewFromInt(100), "USD")
	env.fundOpenReceivableInCurrency(t, alpacadecimal.NewFromInt(100), "USD")
	env.bookFBOBalanceInCurrency(t, alpacadecimal.NewFromInt(200), "EUR")
	env.fundOpenReceivableInCurrency(t, alpacadecimal.NewFromInt(200), "EUR")
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
	require.True(t, balances[0].Balance.Live().Equal(alpacadecimal.NewFromInt(70)))

	require.Equal(t, currencyx.Code("EUR"), balances[1].Currency)
	require.True(t, balances[1].Balance.Settled().Equal(alpacadecimal.NewFromInt(200)))
	require.True(t, balances[1].Balance.Live().Equal(alpacadecimal.NewFromInt(130)))
}

func TestFacadeGetBalancesHistoricalLiveBalanceIsZero(t *testing.T) {
	env := newTestEnv(t)

	env.bookFBOBalance(t, alpacadecimal.NewFromInt(100))
	env.fundOpenReceivable(t, alpacadecimal.NewFromInt(100))
	env.createFlatFeeCharge(t, alpacadecimal.NewFromInt(30), productcatalog.CreditOnlySettlementMode, env.sp())

	facade, err := NewFacade(env.Service)
	require.NoError(t, err)

	// given:
	// - the current balance includes an open charge impact
	currentBalances, err := facade.GetBalances(t.Context(), GetBalancesInput{
		CustomerID: env.CustomerID,
		Currencies: CurrencyFilter{Codes: []currencyx.Code{env.Currency}},
	})
	require.NoError(t, err)
	require.Len(t, currentBalances, 1)
	require.Equal(t, float64(70), currentBalances[0].Balance.Live().InexactFloat64())

	// when:
	// - the same balance is queried with an explicit historical cutoff
	asOf := clock.Now()
	historicalBalances, err := facade.GetBalances(t.Context(), GetBalancesInput{
		CustomerID: env.CustomerID,
		Currencies: CurrencyFilter{Codes: []currencyx.Code{env.Currency}},
		AsOf:       &asOf,
	})
	require.NoError(t, err)
	require.Len(t, historicalBalances, 1)

	// then:
	// - settled is calculated at the cutoff, but live is not calculated
	require.Equal(t, float64(100), historicalBalances[0].Balance.Settled().InexactFloat64())
	require.Equal(t, float64(0), historicalBalances[0].Balance.Live().InexactFloat64())
}

func TestFacadeGetBalancesWithDiscoveredCurrencies(t *testing.T) {
	env := newTestEnv(t)

	env.bookFBOBalanceInCurrency(t, alpacadecimal.NewFromInt(100), "USD")
	env.fundOpenReceivableInCurrency(t, alpacadecimal.NewFromInt(100), "USD")
	env.bookFBOBalanceInCurrency(t, alpacadecimal.NewFromInt(200), "EUR")
	env.fundOpenReceivableInCurrency(t, alpacadecimal.NewFromInt(200), "EUR")
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
			require.True(t, balance.Balance.Live().Equal(alpacadecimal.NewFromInt(70)))
		case "EUR":
			eurCount++
			require.True(t, balance.Balance.Settled().Equal(alpacadecimal.NewFromInt(200)))
			require.True(t, balance.Balance.Live().Equal(alpacadecimal.NewFromInt(130)))
		}
	}

	require.Equal(t, 1, usdCount)
	require.Equal(t, 1, eurCount)
}

func TestFacadeGetBalancesDoesNotDiscoverCreditThenInvoiceOverageAsBalance(t *testing.T) {
	env := newTestEnv(t)
	env.createFlatFeeCharge(t, alpacadecimal.NewFromInt(30), productcatalog.CreditThenInvoiceSettlementMode, env.sp())

	facade, err := NewFacade(env.Service)
	require.NoError(t, err)

	// given:
	// - credit_then_invoice usage has no prepaid credit or pending grant
	// when:
	// - balances are listed without a currency filter
	balances, err := facade.GetBalances(t.Context(), GetBalancesInput{
		CustomerID: env.CustomerID,
	})
	require.NoError(t, err)

	// then:
	// - invoice overage is not exposed as a credit balance
	require.Empty(t, balances)
}

func TestFacadeGetBalancesDiscoversPendingGrantCurrencies(t *testing.T) {
	env := newTestEnv(t)
	facade, err := NewFacade(env.Service)
	require.NoError(t, err)

	env.createPendingInvoiceCreditGrant(t, alpacadecimal.NewFromInt(40), currencyx.Code("EUR"))

	balances, err := facade.GetBalances(t.Context(), GetBalancesInput{
		CustomerID: env.CustomerID,
	})
	require.NoError(t, err)
	require.Len(t, balances, 1)

	require.Equal(t, currencyx.Code("EUR"), balances[0].Currency)
	require.True(t, balances[0].Balance.Settled().Equal(alpacadecimal.Zero))
	require.True(t, balances[0].Balance.Live().Equal(alpacadecimal.Zero))
	require.True(t, balances[0].Balance.Pending().Equal(alpacadecimal.NewFromInt(40)))
}

func TestFacadeGetBalancesWithInvalidExplicitCurrency(t *testing.T) {
	env := newTestEnv(t)

	facade, err := NewFacade(env.Service)
	require.NoError(t, err)

	_, err = facade.GetBalances(t.Context(), GetBalancesInput{
		CustomerID: env.CustomerID,
		Currencies: CurrencyFilter{
			Codes: []currencyx.Code{"X"},
		},
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "X")
	require.ErrorContains(t, err, "not supported by ledger")
}

func TestFacadeGetBalancesWithCatalogCustomCurrencyWithoutCustomerState(t *testing.T) {
	env := newTestEnv(t)
	customCurrency, err := env.Currencies.CreateCurrency(t.Context(), currenciestestutils.NewCreateCurrencyInput(
		env.Namespace,
		"CUSTOM",
		"Custom credits",
		"C",
	))
	require.NoError(t, err)

	facade, err := NewFacade(env.Service)
	require.NoError(t, err)

	// given:
	// - CUSTOM exists in the namespace catalog but the customer has no state in it
	// when:
	// - the balance is explicitly requested by code
	balances, err := facade.GetBalances(t.Context(), GetBalancesInput{
		CustomerID: env.CustomerID,
		Currencies: CurrencyFilter{Codes: []currencyx.Code{"CUSTOM"}},
	})
	require.NoError(t, err)
	require.Len(t, balances, 1)

	// then:
	// - the zero row carries the catalog's immutable custom-currency identity
	require.Equal(t, currencyx.Code("CUSTOM"), balances[0].Currency)
	require.NotNil(t, balances[0].CustomCurrencyID)
	require.Equal(t, customCurrency.ID, *balances[0].CustomCurrencyID)
	require.Equal(t, float64(0), balances[0].Balance.Settled().InexactFloat64())
	require.Equal(t, float64(0), balances[0].Balance.Live().InexactFloat64())
	require.Equal(t, float64(0), balances[0].Balance.Pending().InexactFloat64())
}

func TestFacadeGetBalancesRejectsUnknownCustomCurrency(t *testing.T) {
	env := newTestEnv(t)
	facade, err := NewFacade(env.Service)
	require.NoError(t, err)

	// given:
	// - MISSING is syntactically valid but has neither a catalog definition nor
	//   historical customer state
	// when:
	// - the balance is explicitly requested by code
	_, err = facade.GetBalances(t.Context(), GetBalancesInput{
		CustomerID: env.CustomerID,
		Currencies: CurrencyFilter{Codes: []currencyx.Code{"MISSING"}},
	})

	// then:
	// - the filter is rejected instead of fabricating an identity-less zero row
	require.Error(t, err)
	require.True(t, models.IsGenericValidationError(err))
	require.ErrorContains(t, err, `custom currency "MISSING" is not defined`)
}

func TestFacadeGetBalancesExplicitCurrencyRatesOnlyMatchingUsageOnce(t *testing.T) {
	env := newTestEnv(t)
	env.addUsage(30, clock.Now().Add(-30*time.Minute))
	env.createUsageBasedChargeInCurrency(t, alpacadecimal.NewFromInt(1), productcatalog.CreditOnlySettlementMode, env.sp(), "EUR")
	env.createUsageBasedChargeInCurrency(t, alpacadecimal.NewFromInt(2), productcatalog.CreditOnlySettlementMode, env.sp(), "GBP")

	spy := &countingUsageBasedTotalsService{delegate: env.Service.UsageBasedService}
	env.Service.UsageBasedService = spy
	t.Cleanup(func() {
		env.Service.UsageBasedService = spy.delegate
	})

	facade, err := NewFacade(env.Service)
	require.NoError(t, err)

	// given:
	// - one matching and one unrelated live usage-based charge
	// when:
	// - EUR is explicitly requested
	balances, err := facade.GetBalances(t.Context(), GetBalancesInput{
		CustomerID: env.CustomerID,
		Currencies: CurrencyFilter{Codes: []currencyx.Code{"EUR"}},
	})
	require.NoError(t, err)
	require.Len(t, balances, 1)

	// then:
	// - discovery does not rate either charge, and balance calculation rates the
	//   matching charge exactly once
	require.Equal(t, 1, spy.calls)
	require.Equal(t, currencyx.Code("EUR"), balances[0].Currency)
	require.Equal(t, float64(-30), balances[0].Balance.Live().InexactFloat64())
}

func TestFacadeGetBalancesNoopPreservesExplicitFiatZeroRows(t *testing.T) {
	facade, err := NewFacade(NewNoopService())
	require.NoError(t, err)

	balances, err := facade.GetBalances(t.Context(), GetBalancesInput{
		CustomerID: customer.CustomerID{Namespace: "ns", ID: "customer-id"},
		Currencies: CurrencyFilter{Codes: []currencyx.Code{"USD", "USD", "CUSTOM"}},
	})
	require.NoError(t, err)
	require.Len(t, balances, 1)
	require.Equal(t, currencyx.Code("USD"), balances[0].Currency)
	require.Nil(t, balances[0].CustomCurrencyID)
	require.Equal(t, float64(0), balances[0].Balance.Settled().InexactFloat64())
}

type countingUsageBasedTotalsService struct {
	delegate usageBasedTotalsService
	calls    int
}

func (s *countingUsageBasedTotalsService) GetCurrentTotals(ctx context.Context, input usagebased.GetCurrentTotalsInput) (usagebased.GetCurrentTotalsResult, error) {
	s.calls++

	return s.delegate.GetCurrentTotals(ctx, input)
}

func TestFacadeGetBalancesKeepsCustomCurrencyIdentitiesSeparate(t *testing.T) {
	env := newTestEnv(t)
	alpha := currenciestestutils.NewCustomCurrency(t, "CUSTOM", 2)
	beta := currenciestestutils.NewCustomCurrency(t, "CUSTOM", 2)
	current, err := env.Currencies.CreateCurrency(t.Context(), currenciestestutils.NewCreateCurrencyInput(
		env.Namespace,
		"CUSTOM",
		"Current custom credits",
		"C",
	))
	require.NoError(t, err)

	// given:
	// - two historical managed currencies reuse one display code
	// - the catalog contains a third, current identity with no customer state
	// - each has its own settled customer balance
	env.bookFBOBalanceInCurrencyReferenceWithFeatures(t, alpacadecimal.NewFromInt(40), alpha.Reference(), nil)
	env.fundOpenReceivableInCurrencyReferenceWithFeatures(t, alpacadecimal.NewFromInt(40), alpha.Reference(), nil)
	env.bookFBOBalanceInCurrencyReferenceWithFeatures(t, alpacadecimal.NewFromInt(60), beta.Reference(), nil)
	env.fundOpenReceivableInCurrencyReferenceWithFeatures(t, alpacadecimal.NewFromInt(60), beta.Reference(), nil)

	facade, err := NewFacade(env.Service)
	require.NoError(t, err)

	// when:
	// - balances are filtered by their shared code
	balances, err := facade.GetBalances(t.Context(), GetBalancesInput{
		CustomerID: env.CustomerID,
		Currencies: CurrencyFilter{
			Codes: []currencyx.Code{"CUSTOM"},
		},
	})
	require.NoError(t, err)
	require.Len(t, balances, 3)

	// then:
	// - each immutable custom-currency ID has an independent balance row
	settledByID := make(map[string]float64, len(balances))
	for _, balance := range balances {
		reference := balance.CurrencyReference()
		require.Equal(t, currencyx.Code("CUSTOM"), balance.Currency)
		require.NotNil(t, reference.CustomCurrencyID)
		settledByID[*reference.CustomCurrencyID] = balance.Balance.Settled().InexactFloat64()
	}
	require.Equal(t, map[string]float64{
		alpha.ID:   40,
		beta.ID:    60,
		current.ID: 0,
	}, settledByID)
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
	env.fundOpenReceivable(t, alpacadecimal.NewFromInt(100))

	pagedBeforeSecondIssue, err := env.Deps.HistoricalLedger.ListTransactions(t.Context(), ledger.ListTransactionsInput{
		Namespace:  env.Namespace,
		Limit:      10,
		AccountIDs: []string{env.CustomerAccounts.ReceivableAccount.ID().ID},
		Currency:   &env.Currency,
	})
	require.NoError(t, err)
	require.Len(t, pagedBeforeSecondIssue.Items, 3)

	cursorAfterFunding := pagedBeforeSecondIssue.Items[0].Cursor()

	clock.SetTime(secondBookedAt)
	env.bookFBOBalance(t, alpacadecimal.NewFromInt(20))
	env.fundOpenReceivable(t, alpacadecimal.NewFromInt(20))

	balanceAfterOlderTx, err := facade.GetBalance(t.Context(), GetBalanceInput{
		CustomerID: env.CustomerID,
		Currency:   env.Currency,
		After:      &cursorAfterFunding,
	})
	require.NoError(t, err)
	require.True(t, balanceAfterOlderTx.Equal(alpacadecimal.NewFromInt(100)), "balance after older tx: %s", balanceAfterOlderTx)

	currentBalance, err := facade.GetBalance(t.Context(), GetBalanceInput{
		CustomerID: env.CustomerID,
		Currency:   env.Currency,
	})
	require.NoError(t, err)
	require.True(t, currentBalance.Equal(alpacadecimal.NewFromInt(120)))
}

func TestFacadeGetBalanceAsOfIncludesBreakageExpiry(t *testing.T) {
	env := newTestEnv(t)
	facade, err := NewFacade(env.Service)
	require.NoError(t, err)

	issuedAt := time.Date(2026, 4, 10, 9, 0, 0, 0, time.UTC)
	expiresAt := issuedAt.Add(time.Hour)
	beforeExpiry := expiresAt.Add(-time.Nanosecond)

	clock.FreezeTime(issuedAt)
	defer clock.UnFreeze()
	defer clock.ResetTime()

	amount := alpacadecimal.NewFromInt(100)
	env.bookFBOBalance(t, amount)
	env.fundOpenReceivable(t, amount)

	fbo := env.FBOSubAccount(t, ledger.DefaultCustomerFBOPriority)
	breakage := env.BreakageSubAccountWithCostBasis(t, nil)

	inputs, err := transactions.ResolveTransactions(
		t.Context(),
		transactions.ResolverDependencies{
			AccountService: env.Deps.ResolversService,
			AccountCatalog: env.Deps.AccountService,
			BalanceQuerier: env.Deps.HistoricalLedger,
		},
		transactions.ResolutionScope{
			CustomerID: env.CustomerID,
			Namespace:  env.Namespace,
		},
		transactions.PlanCustomerFBOBreakageTemplate{
			At:              expiresAt,
			Amount:          amount,
			FBOAddress:      fbo.Address(),
			BreakageAddress: breakage.Address(),
		},
	)
	require.NoError(t, err)

	_, err = env.Deps.HistoricalLedger.CommitGroup(t.Context(), transactions.GroupInputs(env.Namespace, nil, inputs...))
	require.NoError(t, err)

	balanceBeforeExpiry, err := facade.GetBalance(t.Context(), GetBalanceInput{
		CustomerID: env.CustomerID,
		Currency:   env.Currency,
		AsOf:       &beforeExpiry,
	})
	require.NoError(t, err)
	require.True(t, balanceBeforeExpiry.Equal(amount), "balance before expiry: %s", balanceBeforeExpiry)

	balanceAtExpiry, err := facade.GetBalance(t.Context(), GetBalanceInput{
		CustomerID: env.CustomerID,
		Currency:   env.Currency,
		AsOf:       &expiresAt,
	})
	require.NoError(t, err)
	require.True(t, balanceAtExpiry.Equal(alpacadecimal.Zero), "balance at expiry: %s", balanceAtExpiry)

	clock.FreezeTime(beforeExpiry)
	currentBeforeExpiry, err := facade.GetBalance(t.Context(), GetBalanceInput{
		CustomerID: env.CustomerID,
		Currency:   env.Currency,
	})
	require.NoError(t, err)
	require.True(t, currentBeforeExpiry.Equal(amount), "current balance before expiry: %s", currentBeforeExpiry)

	clock.FreezeTime(expiresAt)
	currentAtExpiry, err := facade.GetBalance(t.Context(), GetBalanceInput{
		CustomerID: env.CustomerID,
		Currency:   env.Currency,
	})
	require.NoError(t, err)
	require.True(t, currentAtExpiry.Equal(alpacadecimal.Zero), "current balance at expiry: %s", currentAtExpiry)
}

func TestFacadeGetBalanceAsOf(t *testing.T) {
	env := newTestEnv(t)
	facade, err := NewFacade(env.Service)
	require.NoError(t, err)

	firstBookedAt := time.Date(2026, 4, 10, 9, 0, 0, 0, time.UTC)
	secondBookedAt := firstBookedAt.Add(time.Minute)

	clock.FreezeTime(firstBookedAt)
	defer clock.UnFreeze()
	defer clock.ResetTime()
	env.bookFBOBalance(t, alpacadecimal.NewFromInt(100))
	env.fundOpenReceivable(t, alpacadecimal.NewFromInt(100))

	clock.FreezeTime(secondBookedAt)
	env.bookFBOBalance(t, alpacadecimal.NewFromInt(20))
	env.fundOpenReceivable(t, alpacadecimal.NewFromInt(20))

	balanceAtFirstBooking, err := facade.GetBalance(t.Context(), GetBalanceInput{
		CustomerID: env.CustomerID,
		Currency:   env.Currency,
		AsOf:       &firstBookedAt,
	})
	require.NoError(t, err)
	require.True(t, balanceAtFirstBooking.Equal(alpacadecimal.NewFromInt(100)), "balance as of first booking: %s", balanceAtFirstBooking)

	currentBalance, err := facade.GetBalance(t.Context(), GetBalanceInput{
		CustomerID: env.CustomerID,
		Currency:   env.Currency,
	})
	require.NoError(t, err)
	require.True(t, currentBalance.Equal(alpacadecimal.NewFromInt(120)))
}
