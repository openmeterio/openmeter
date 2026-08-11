package creditpurchase

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	chargecostbasis "github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestCostBasisVariants(t *testing.T) {
	t.Run("fiat", func(t *testing.T) {
		costBasis := NewCostBasis(FiatCostBasis{
			Rate: alpacadecimal.NewFromFloat(1.25),
		})

		require.NoError(t, costBasis.Validate())
		require.Equal(t, CostBasisTypeFiat, costBasis.Type())

		fiat, err := costBasis.AsFiat()
		require.NoError(t, err)
		require.Equal(t, 1.25, fiat.Rate.InexactFloat64())
		fiatCurrency, err := costBasis.GetFiatCurrency(currenciestestutils.NewFiatCurrency(t, "USD"))
		require.NoError(t, err)
		require.Equal(t, currencyx.FiatCode("USD"), fiatCurrency.GetFiatCode())

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

		intent, err := costBasis.AsCustomCurrency()
		require.NoError(t, err)
		require.Equal(t, chargecostbasis.ModeManual, intent.Kind())
		fiatCurrency, err := costBasis.GetFiatCurrency(currenciestestutils.NewCustomCurrency(t, "TOKENS", 2))
		require.NoError(t, err)
		require.Equal(t, currencyx.FiatCode("USD"), fiatCurrency.GetFiatCode())

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

func TestResolvedCostBasisValidate(t *testing.T) {
	require.ErrorContains(t, (ResolvedCostBasis{
		Rate: alpacadecimal.NewFromInt(1),
	}).Validate(), "fiat currency is required")

	usd, err := currencyx.NewFiatCurrency("USD")
	require.NoError(t, err)
	require.ErrorContains(t, (ResolvedCostBasis{
		FiatCurrency: usd,
		Rate:         alpacadecimal.Zero,
	}).Validate(), "rate must be positive")
}

func TestResolvedCostBasisFiatAmount(t *testing.T) {
	usd, err := currencyx.NewFiatCurrency("USD")
	require.NoError(t, err)

	amount := (ResolvedCostBasis{
		FiatCurrency: usd,
		Rate:         alpacadecimal.NewFromFloat(0.5),
	}).FiatAmount(alpacadecimal.NewFromFloat(100.1234))

	require.Equal(t, 50.06, amount.InexactFloat64())
}

func TestChargeBaseValidateCostBasis(t *testing.T) {
	usd, err := currencyx.NewFiatCurrency("USD")
	require.NoError(t, err)

	resolved := chargecostbasis.State{
		CostBasis:  alpacadecimal.NewFromFloat(0.5),
		ResolvedAt: time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC),
	}
	intent := Intent{CostBasis: lo.ToPtr(NewCostBasis(chargecostbasis.NewIntent(chargecostbasis.ManualIntent{
		FiatCurrency: usd,
		Rate:         alpacadecimal.NewFromFloat(0.5),
	})))}
	intent.Currency = currenciestestutils.NewCustomCurrency(t, "TOKENS", 2)
	intent.Settlement = NewInvoiceSettlement()

	require.NoError(t, (ChargeBase{Intent: intent, State: State{
		ResolvedCostBasis: &resolved,
	}}).validateCostBasis())

	require.Error(t, (ChargeBase{Intent: intent}).validateCostBasis())

	t.Run("fiat resolved state matches immutable intent", func(t *testing.T) {
		createdAt := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
		fiatIntent := Intent{
			Intent: meta.Intent{Currency: currenciestestutils.NewFiatCurrency(t, "USD")},
			IntentMutableFields: IntentMutableFields{
				Settlement: NewInvoiceSettlement(),
			},
			CostBasis: lo.ToPtr(NewCostBasis(FiatCostBasis{Rate: alpacadecimal.NewFromFloat(0.5)})),
		}
		validState := State{ResolvedCostBasis: &chargecostbasis.State{
			CostBasis:  alpacadecimal.NewFromFloat(0.5),
			ResolvedAt: createdAt,
		}}

		require.NoError(t, (ChargeBase{
			ManagedResource: meta.ManagedResource{ManagedModel: models.ManagedModel{CreatedAt: createdAt}},
			Intent:          fiatIntent,
			State:           validState,
		}).validateCostBasis())

		mismatchedRate := validState
		mismatchedRate.ResolvedCostBasis = lo.ToPtr(*validState.ResolvedCostBasis)
		mismatchedRate.ResolvedCostBasis.CostBasis = alpacadecimal.NewFromInt(1)
		require.ErrorContains(t, (ChargeBase{
			ManagedResource: meta.ManagedResource{ManagedModel: models.ManagedModel{CreatedAt: createdAt}},
			Intent:          fiatIntent,
			State:           mismatchedRate,
		}).validateCostBasis(), "must match the intent rate")

		withSource := validState
		withSource.ResolvedCostBasis = lo.ToPtr(*validState.ResolvedCostBasis)
		withSource.ResolvedCostBasis.CostBasisID = lo.ToPtr("currency-cost-basis-id")
		require.ErrorContains(t, (ChargeBase{
			ManagedResource: meta.ManagedResource{ManagedModel: models.ManagedModel{CreatedAt: createdAt}},
			Intent:          fiatIntent,
			State:           withSource,
		}).validateCostBasis(), "cannot reference a currency cost basis")

		require.ErrorContains(t, (ChargeBase{
			ManagedResource: meta.ManagedResource{ManagedModel: models.ManagedModel{CreatedAt: createdAt}},
			Intent:          fiatIntent,
		}).validateCostBasis(), "requires resolved cost-basis state")
	})
}
