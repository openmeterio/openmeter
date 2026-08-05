package currencyresolver_test

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	productcatalogtestutils "github.com/openmeterio/openmeter/openmeter/productcatalog/testutils"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription/currencyresolver"
)

func TestResolveCurrenciesForSubscriptionSpec(t *testing.T) {
	env := productcatalogtestutils.NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})

	const (
		namespace      = "subscription-currency-resolver"
		otherNamespace = "subscription-currency-resolver-other"
	)

	credits, err := env.Currency.CreateCurrency(t.Context(), currenciestestutils.NewCreateCurrencyInput(
		namespace,
		"CREDITS",
		"Credits",
		"CR",
	))
	require.NoError(t, err)

	tokens, err := env.Currency.CreateCurrency(t.Context(), currenciestestutils.NewCreateCurrencyInput(
		namespace,
		"TOKENS",
		"Tokens",
		"T",
	))
	require.NoError(t, err)

	otherCredits, err := env.Currency.CreateCurrency(t.Context(), currenciestestutils.NewCreateCurrencyInput(
		otherNamespace,
		"CREDITS",
		"Other credits",
		"OCR",
	))
	require.NoError(t, err)

	resolver := env.CurrencyResolver.WithNamespace(namespace)

	t.Run("resolves custom currency by code", func(t *testing.T) {
		spec := newSubscriptionSpecWithCurrency(currencies.NewCurrencyReference(credits.GetCode()))

		err := currencyresolver.ResolveCurrenciesForSubscriptionSpec(t.Context(), resolver, &spec)
		require.NoError(t, err)

		resolved := spec.Phases["default"].ItemsByKey["fee"][0].RateCard.AsMeta().Currency
		require.NotNil(t, resolved)
		require.True(t, resolved.IsResolved())
		require.NotNil(t, resolved.CustomCurrencyID)
		require.Equal(t, credits.ID, *resolved.CustomCurrencyID)
	})

	t.Run("resolves matching custom currency code and id", func(t *testing.T) {
		currencyID := credits.ID
		spec := newSubscriptionSpecWithCurrency(currencies.CurrencyReference{
			Code:             credits.GetCode(),
			CustomCurrencyID: &currencyID,
		})

		err := currencyresolver.ResolveCurrenciesForSubscriptionSpec(t.Context(), resolver, &spec)
		require.NoError(t, err)

		resolved := spec.Phases["default"].ItemsByKey["fee"][0].RateCard.AsMeta().Currency
		require.NotNil(t, resolved)
		require.True(t, resolved.IsResolved())
		currency, ok := resolved.CustomCurrency()
		require.True(t, ok)
		require.Equal(t, namespace, currency.Namespace)
		require.Equal(t, credits.ID, currency.ID)
	})

	t.Run("rejects mismatched custom currency code and id", func(t *testing.T) {
		currencyID := tokens.ID
		spec := newSubscriptionSpecWithCurrency(currencies.CurrencyReference{
			Code:             credits.GetCode(),
			CustomCurrencyID: &currencyID,
		})

		err := currencyresolver.ResolveCurrenciesForSubscriptionSpec(t.Context(), resolver, &spec)
		require.ErrorIs(t, err, productcatalog.ErrCurrencyNotFound)
	})

	t.Run("rejects custom currency from another namespace", func(t *testing.T) {
		currencyID := otherCredits.ID
		spec := newSubscriptionSpecWithCurrency(currencies.CurrencyReference{
			Code:             otherCredits.GetCode(),
			CustomCurrencyID: &currencyID,
		})

		err := currencyresolver.ResolveCurrenciesForSubscriptionSpec(t.Context(), resolver, &spec)
		require.ErrorIs(t, err, productcatalog.ErrCurrencyNotFound)
	})

	t.Run("rejects nil spec", func(t *testing.T) {
		err := currencyresolver.ResolveCurrenciesForSubscriptionSpec(t.Context(), resolver, nil)
		require.EqualError(t, err, "subscription spec is required")
	})
}

func newSubscriptionSpecWithCurrency(reference currencies.CurrencyReference) subscription.SubscriptionSpec {
	rateCard := &productcatalog.FlatFeeRateCard{
		RateCardMeta: productcatalog.RateCardMeta{
			Key:      "fee",
			Name:     "Fee",
			Currency: &reference,
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount: alpacadecimal.NewFromInt(1),
			}),
		},
	}

	return subscription.SubscriptionSpec{
		Phases: map[string]*subscription.SubscriptionPhaseSpec{
			"default": {
				ItemsByKey: map[string][]*subscription.SubscriptionItemSpec{
					"fee": {
						{
							CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
								CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
									PhaseKey: "default",
									ItemKey:  "fee",
									RateCard: rateCard,
								},
							},
						},
					},
				},
			},
		},
	}
}
