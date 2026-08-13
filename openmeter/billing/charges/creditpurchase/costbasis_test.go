package creditpurchase

import (
	"encoding/json"
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	chargecostbasis "github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestCostBasisVariants(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		costBasis := CostBasis{}

		require.True(t, costBasis.IsEmpty())
		require.Empty(t, costBasis.GetCustomCurrencyModeOrEmpty())
		require.Error(t, costBasis.Validate())
	})

	t.Run("fiat", func(t *testing.T) {
		costBasis := NewCostBasis(FiatCostBasis{
			Rate: alpacadecimal.NewFromFloat(1.25),
		})

		require.NoError(t, costBasis.Validate())
		require.Equal(t, CostBasisTypeFiat, costBasis.Type())
		require.Empty(t, costBasis.GetCustomCurrencyModeOrEmpty())

		fiat, err := costBasis.AsFiat()
		require.NoError(t, err)
		require.Equal(t, 1.25, fiat.Rate.InexactFloat64())
		_, err = costBasis.AsCustomCurrency()
		require.Error(t, err)
	})

	t.Run("custom currency keeps shared validation", func(t *testing.T) {
		usd, err := currencyx.NewFiatCurrency("USD")
		require.NoError(t, err)

		costBasis := NewCostBasis(chargecostbasis.NewIntent(chargecostbasis.ManualIntent{
			FiatCurrency: usd,
			Rate:         alpacadecimal.NewFromFloat(0.5),
		}))

		require.NoError(t, costBasis.Validate())
		require.Equal(t, CostBasisTypeCustomCurrency, costBasis.Type())
		require.Equal(t, chargecostbasis.ModeManual, costBasis.GetCustomCurrencyModeOrEmpty())

		intent, err := costBasis.AsCustomCurrency()
		require.NoError(t, err)
		require.Equal(t, chargecostbasis.ModeManual, intent.Kind())
		_, err = costBasis.AsFiat()
		require.Error(t, err)
	})

	t.Run("fiat rate must be positive", func(t *testing.T) {
		costBasis := NewCostBasis(FiatCostBasis{
			Rate: alpacadecimal.Zero,
		})
		require.Error(t, costBasis.Validate())
	})
}

func TestCostBasisJSONRoundTrip(t *testing.T) {
	usd, err := currencyx.NewFiatCurrency("USD")
	require.NoError(t, err)

	testCases := map[string]CostBasis{
		"fiat": NewCostBasis(FiatCostBasis{
			Rate: alpacadecimal.NewFromFloat(1.25),
		}),
		"custom currency dynamic": NewCostBasis(chargecostbasis.NewIntent(chargecostbasis.DynamicIntent{
			FiatCurrency: usd,
		})),
		"custom currency pinned": NewCostBasis(chargecostbasis.NewIntent(chargecostbasis.PinnedIntent{
			FiatCurrency:        usd,
			CurrencyCostBasisID: "01J00000000000000000000000",
		})),
		"custom currency manual": NewCostBasis(chargecostbasis.NewIntent(chargecostbasis.ManualIntent{
			FiatCurrency: usd,
			Rate:         alpacadecimal.NewFromFloat(0.5),
		})),
	}

	for name, costBasis := range testCases {
		t.Run(name, func(t *testing.T) {
			encoded, err := json.Marshal(costBasis)
			require.NoError(t, err)
			require.NotEqual(t, "{}", string(encoded))

			var decoded CostBasis
			require.NoError(t, json.Unmarshal(encoded, &decoded))
			require.NoError(t, decoded.Validate())
			require.Equal(t, costBasis.Type(), decoded.Type())

			reencoded, err := json.Marshal(decoded)
			require.NoError(t, err)
			require.JSONEq(t, string(encoded), string(reencoded))
		})
	}
}

func TestEmptyCostBasisJSONRoundTrip(t *testing.T) {
	encoded, err := json.Marshal(CostBasis{})
	require.NoError(t, err)
	require.JSONEq(t, "null", string(encoded))

	var decoded CostBasis
	require.NoError(t, json.Unmarshal(encoded, &decoded))
	require.True(t, decoded.IsEmpty())
}
