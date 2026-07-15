package transactions

import (
	"fmt"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestConvertCurrencyTemplate(t *testing.T) {
	t.Run("posts explicit fiat and source-aware custom legs", func(t *testing.T) {
		// given:
		// - a caller-materialized USD to ACME funding exchange
		// when:
		// - the exchange is resolved
		// then:
		// - receivable and brokerage receive opposite legs in each currency
		env := newTransactionsTestEnv(t)
		costBasis := alpacadecimal.NewFromFloat(0.25)
		inputs := env.resolve(
			t,
			ConvertCurrencyTemplate{
				At:             env.Now(),
				SourceAmount:   alpacadecimal.NewFromInt(25),
				TargetAmount:   alpacadecimal.NewFromInt(100),
				CostBasis:      costBasis,
				SourceCurrency: currencyx.Code("USD"),
				TargetCurrency: currencyx.Code("ACME"),
			},
		)
		require.Len(t, inputs, 1)
		require.Len(t, inputs[0].EntryInputs(), 4)

		expected := map[string]float64{
			fmt.Sprintf("%s/%s", ledger.AccountTypeCustomerReceivable, currencyx.Code("USD")):  -25,
			fmt.Sprintf("%s/%s", ledger.AccountTypeBrokerage, currencyx.Code("USD")):           25,
			fmt.Sprintf("%s/%s", ledger.AccountTypeCustomerReceivable, currencyx.Code("ACME")): 100,
			fmt.Sprintf("%s/%s", ledger.AccountTypeBrokerage, currencyx.Code("ACME")):          -100,
		}
		totals := map[currencyx.Code]alpacadecimal.Decimal{}

		for _, entry := range inputs[0].EntryInputs() {
			route := entry.PostingAddress().Route().Route()
			key := fmt.Sprintf("%s/%s", entry.PostingAddress().AccountType(), route.Currency)
			require.Equal(t, expected[key], entry.Amount().InexactFloat64())
			delete(expected, key)

			require.NotNil(t, route.CostBasis)
			require.Equal(t, costBasis.InexactFloat64(), route.CostBasis.InexactFloat64())
			require.Nil(t, entry.SourceChargeID())
			require.Nil(t, entry.SpendChargeID())
			if route.Currency == currencyx.Code("USD") {
				require.Nil(t, route.ExchangeSourceCurrency)
			} else {
				require.Equal(t, currencyx.Code("USD"), *route.ExchangeSourceCurrency)
			}

			totals[route.Currency] = totals[route.Currency].Add(entry.Amount())
		}

		require.Empty(t, expected)
		require.Equal(t, float64(0), totals[currencyx.Code("USD")].InexactFloat64())
		require.Equal(t, float64(0), totals[currencyx.Code("ACME")].InexactFloat64())
	})

	t.Run("validates one way funding inputs", func(t *testing.T) {
		valid := ConvertCurrencyTemplate{
			At:             time.Now(),
			SourceAmount:   alpacadecimal.NewFromInt(25),
			TargetAmount:   alpacadecimal.NewFromInt(100),
			CostBasis:      alpacadecimal.NewFromFloat(0.25),
			SourceCurrency: currencyx.Code("USD"),
			TargetCurrency: currencyx.Code("ACME"),
		}

		tests := []struct {
			name   string
			mutate func(*ConvertCurrencyTemplate)
		}{
			{
				name: "fiat to custom",
			},
			{
				name: "fiat to fiat",
				mutate: func(input *ConvertCurrencyTemplate) {
					input.TargetCurrency = currencyx.Code("EUR")
				},
			},
			{
				name: "custom to custom",
				mutate: func(input *ConvertCurrencyTemplate) {
					input.SourceCurrency = currencyx.Code("COIN")
				},
			},
			{
				name: "custom to fiat",
				mutate: func(input *ConvertCurrencyTemplate) {
					input.SourceCurrency = currencyx.Code("ACME")
					input.TargetCurrency = currencyx.Code("USD")
				},
			},
			{
				name: "equal currencies",
				mutate: func(input *ConvertCurrencyTemplate) {
					input.TargetCurrency = currencyx.Code("USD")
				},
			},
			{
				name: "zero source amount",
				mutate: func(input *ConvertCurrencyTemplate) {
					input.SourceAmount = alpacadecimal.Zero
				},
			},
			{
				name: "negative source amount",
				mutate: func(input *ConvertCurrencyTemplate) {
					input.SourceAmount = alpacadecimal.NewFromInt(-1)
				},
			},
			{
				name: "zero target amount",
				mutate: func(input *ConvertCurrencyTemplate) {
					input.TargetAmount = alpacadecimal.Zero
				},
			},
			{
				name: "negative target amount",
				mutate: func(input *ConvertCurrencyTemplate) {
					input.TargetAmount = alpacadecimal.NewFromInt(-1)
				},
			},
			{
				name: "zero cost basis",
				mutate: func(input *ConvertCurrencyTemplate) {
					input.CostBasis = alpacadecimal.Zero
				},
			},
			{
				name: "negative cost basis",
				mutate: func(input *ConvertCurrencyTemplate) {
					input.CostBasis = alpacadecimal.NewFromFloat(-0.25)
				},
			},
			{
				name: "inconsistent positive amounts and cost basis",
				mutate: func(input *ConvertCurrencyTemplate) {
					input.SourceAmount = alpacadecimal.NewFromInt(26)
				},
			},
			{
				name: "malformed target",
				mutate: func(input *ConvertCurrencyTemplate) {
					input.TargetCurrency = currencyx.Code("AC|ME")
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				input := valid
				if tt.mutate != nil {
					tt.mutate(&input)
				}

				err := input.Validate()
				if tt.mutate == nil {
					require.NoError(t, err)
					return
				}

				require.Error(t, err)
			})
		}
	})

	t.Run("uses source currency precision for amount consistency", func(t *testing.T) {
		err := (ConvertCurrencyTemplate{
			At:             time.Now(),
			SourceAmount:   alpacadecimal.NewFromInt(1),
			TargetAmount:   alpacadecimal.NewFromInt(3),
			CostBasis:      alpacadecimal.NewFromFloat(0.333),
			SourceCurrency: currencyx.Code("USD"),
			TargetCurrency: currencyx.Code("ACME"),
		}).Validate()
		require.NoError(t, err)
	})

	t.Run("collects all validation failures", func(t *testing.T) {
		err := (ConvertCurrencyTemplate{
			SourceAmount:   alpacadecimal.NewFromInt(-1),
			TargetAmount:   alpacadecimal.NewFromInt(-1),
			CostBasis:      alpacadecimal.NewFromInt(-1),
			SourceCurrency: currencyx.Code("AC|ME"),
			TargetCurrency: currencyx.Code("USD"),
		}).Validate()
		require.True(t, models.IsGenericValidationError(err))
		require.ErrorContains(t, err, "at is required")
		require.ErrorContains(t, err, "source amount:")
		require.ErrorContains(t, err, "target amount:")
		require.ErrorContains(t, err, "cost basis:")
		require.ErrorContains(t, err, "source currency:")
		require.ErrorContains(t, err, "target currency must be custom")
	})
}

func TestFiatToCustomFundingLifecycle(t *testing.T) {
	t.Run("full spend matches the credit only account lifecycle", func(t *testing.T) {
		// given:
		// - 100 ACME is issued and funded by a materialized 25 USD exchange
		// when:
		// - the USD payment settles and all ACME is consumed
		// then:
		// - receivable clears while brokerage retains the paired funding balances
		env := newTransactionsTestEnv(t)
		costBasis := alpacadecimal.NewFromFloat(0.25)
		inputs := fundingLifecycleInputs(t, env, costBasis, alpacadecimal.NewFromInt(100))

		_, err := env.Deps.HistoricalLedger.CommitGroup(t.Context(), GroupInputs(env.Namespace, nil, inputs...))
		require.NoError(t, err)

		customFBO := customerFBOSubAccount(t, env, currencyx.Code("ACME"), lo.ToPtr(currencyx.Code("USD")), &costBasis)
		customReceivable := customerReceivableSubAccount(t, env, currencyx.Code("ACME"), lo.ToPtr(currencyx.Code("USD")), &costBasis, ledger.TransactionAuthorizationStatusOpen)
		fiatReceivable := customerReceivableSubAccount(t, env, currencyx.Code("USD"), nil, &costBasis, ledger.TransactionAuthorizationStatusOpen)
		fiatAuthorizedReceivable := customerReceivableSubAccount(t, env, currencyx.Code("USD"), nil, &costBasis, ledger.TransactionAuthorizationStatusAuthorized)
		customBrokerage := businessSubAccount(t, env.BusinessAccounts.BrokerageAccount, currencyx.Code("ACME"), lo.ToPtr(currencyx.Code("USD")), &costBasis)
		fiatBrokerage := businessSubAccount(t, env.BusinessAccounts.BrokerageAccount, currencyx.Code("USD"), nil, &costBasis)
		fiatWash := businessSubAccount(t, env.BusinessAccounts.WashAccount, currencyx.Code("USD"), nil, &costBasis)
		customAccrued := customerAccruedSubAccount(t, env, currencyx.Code("ACME"), lo.ToPtr(currencyx.Code("USD")), &costBasis)

		require.Equal(t, float64(0), env.SumBalance(t, customFBO).InexactFloat64())
		require.Equal(t, float64(0), env.SumBalance(t, customReceivable).InexactFloat64())
		require.Equal(t, float64(0), env.SumBalance(t, fiatReceivable).InexactFloat64())
		require.Equal(t, float64(0), env.SumBalance(t, fiatAuthorizedReceivable).InexactFloat64())
		require.Equal(t, float64(-100), env.SumBalance(t, customBrokerage).InexactFloat64())
		require.Equal(t, float64(25), env.SumBalance(t, fiatBrokerage).InexactFloat64())
		require.Equal(t, float64(-25), env.SumBalance(t, fiatWash).InexactFloat64())
		require.Equal(t, float64(100), env.SumBalance(t, customAccrued).InexactFloat64())

		customCurrency := currencyx.Code("ACME")
		filtered, err := env.Deps.HistoricalLedger.ListTransactions(t.Context(), ledger.ListTransactionsInput{
			Namespace: env.Namespace,
			Limit:     10,
			Currency:  &customCurrency,
		})
		require.NoError(t, err)
		require.Len(t, filtered.Items, 3)
		for _, transaction := range filtered.Items {
			for _, entry := range transaction.Entries() {
				require.Equal(t, customCurrency, entry.PostingAddress().Route().Route().Currency)
			}
		}
	})

	t.Run("partial spend leaves unspent custom balance", func(t *testing.T) {
		// given:
		// - the same 100 ACME funding lifecycle
		// when:
		// - only 40 ACME is consumed
		// then:
		// - 60 ACME remains in FBO and funding balances do not change
		env := newTransactionsTestEnv(t)
		costBasis := alpacadecimal.NewFromFloat(0.25)
		inputs := fundingLifecycleInputs(t, env, costBasis, alpacadecimal.NewFromInt(40))

		_, err := env.Deps.HistoricalLedger.CommitGroup(
			t.Context(),
			GroupInputs(env.Namespace, nil, inputs...),
		)
		require.NoError(t, err)

		require.Equal(t, float64(60), env.SumBalance(t, customerFBOSubAccount(t, env, currencyx.Code("ACME"), lo.ToPtr(currencyx.Code("USD")), &costBasis)).InexactFloat64())
		require.Equal(t, float64(40), env.SumBalance(t, customerAccruedSubAccount(t, env, currencyx.Code("ACME"), lo.ToPtr(currencyx.Code("USD")), &costBasis)).InexactFloat64())
		require.Equal(t, float64(-100), env.SumBalance(t, businessSubAccount(t, env.BusinessAccounts.BrokerageAccount, currencyx.Code("ACME"), lo.ToPtr(currencyx.Code("USD")), &costBasis)).InexactFloat64())
		require.Equal(t, float64(25), env.SumBalance(t, businessSubAccount(t, env.BusinessAccounts.BrokerageAccount, currencyx.Code("USD"), nil, &costBasis)).InexactFloat64())
		require.Equal(t, float64(-25), env.SumBalance(t, businessSubAccount(t, env.BusinessAccounts.WashAccount, currencyx.Code("USD"), nil, &costBasis)).InexactFloat64())
	})
}

func fundingLifecycleInputs(
	t *testing.T,
	env *transactionsTestEnv,
	costBasis alpacadecimal.Decimal,
	spendAmount alpacadecimal.Decimal,
) []ledger.TransactionInput {
	t.Helper()

	customCurrency := currencyx.Code("ACME")
	fiatCurrency := currencyx.Code("USD")
	exchangeSourceCurrency := lo.ToPtr(fiatCurrency)
	fbo := customerFBOSubAccount(t, env, customCurrency, exchangeSourceCurrency, &costBasis)

	return env.resolve(
		t,
		IssueCustomerReceivableTemplate{
			At:                     env.Now(),
			Amount:                 alpacadecimal.NewFromInt(100),
			Currency:               customCurrency,
			ExchangeSourceCurrency: exchangeSourceCurrency,
			CostBasis:              &costBasis,
		},
		ConvertCurrencyTemplate{
			At:             env.Now(),
			SourceAmount:   alpacadecimal.NewFromInt(25),
			TargetAmount:   alpacadecimal.NewFromInt(100),
			CostBasis:      costBasis,
			SourceCurrency: fiatCurrency,
			TargetCurrency: customCurrency,
		},
		AuthorizeCustomerReceivablePaymentTemplate{
			At:        env.Now(),
			Amount:    alpacadecimal.NewFromInt(25),
			Currency:  fiatCurrency,
			CostBasis: &costBasis,
		},
		SettleCustomerReceivableFromPaymentTemplate{
			At:        env.Now(),
			Amount:    alpacadecimal.NewFromInt(25),
			Currency:  fiatCurrency,
			CostBasis: &costBasis,
		},
		TransferCustomerFBOToAccruedTemplate{
			At:       env.Now(),
			Currency: customCurrency,
			Sources: []PostingAmount{
				{
					Address: fbo.Address(),
					Amount:  spendAmount,
				},
			},
		},
	)
}

func customerFBOSubAccount(
	t *testing.T,
	env *transactionsTestEnv,
	currency currencyx.Code,
	exchangeSourceCurrency *currencyx.Code,
	costBasis *alpacadecimal.Decimal,
) ledger.SubAccount {
	t.Helper()

	subAccount, err := env.CustomerAccounts.FBOAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerFBORouteParams{
		Currency:               currency,
		ExchangeSourceCurrency: exchangeSourceCurrency,
		CreditPriority:         ledger.DefaultCustomerFBOPriority,
		CostBasis:              costBasis,
	})
	require.NoError(t, err)

	return subAccount
}

func customerReceivableSubAccount(
	t *testing.T,
	env *transactionsTestEnv,
	currency currencyx.Code,
	exchangeSourceCurrency *currencyx.Code,
	costBasis *alpacadecimal.Decimal,
	status ledger.TransactionAuthorizationStatus,
) ledger.SubAccount {
	t.Helper()

	subAccount, err := env.CustomerAccounts.ReceivableAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerReceivableRouteParams{
		Currency:                       currency,
		ExchangeSourceCurrency:         exchangeSourceCurrency,
		CostBasis:                      costBasis,
		TransactionAuthorizationStatus: status,
	})
	require.NoError(t, err)

	return subAccount
}

func customerAccruedSubAccount(
	t *testing.T,
	env *transactionsTestEnv,
	currency currencyx.Code,
	exchangeSourceCurrency *currencyx.Code,
	costBasis *alpacadecimal.Decimal,
) ledger.SubAccount {
	t.Helper()

	subAccount, err := env.CustomerAccounts.AccruedAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerAccruedRouteParams{
		Currency:               currency,
		ExchangeSourceCurrency: exchangeSourceCurrency,
		CostBasis:              costBasis,
	})
	require.NoError(t, err)

	return subAccount
}

func businessSubAccount(
	t *testing.T,
	account ledger.BusinessAccount,
	currency currencyx.Code,
	exchangeSourceCurrency *currencyx.Code,
	costBasis *alpacadecimal.Decimal,
) ledger.SubAccount {
	t.Helper()

	subAccount, err := account.GetSubAccountForRoute(t.Context(), ledger.BusinessRouteParams{
		Currency:               currency,
		ExchangeSourceCurrency: exchangeSourceCurrency,
		CostBasis:              costBasis,
	})
	require.NoError(t, err)

	return subAccount
}
