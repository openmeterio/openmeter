package creditpurchase

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	chargecostbasis "github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/pkg/currencyx"
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
		SchemaLevel:       SchemaLevelLegacy,
		ResolvedCostBasis: &resolved,
	}}).validateCostBasis())

	require.Error(t, (ChargeBase{Intent: intent, State: State{
		SchemaLevel:       SchemaLevelCostBasis,
		ResolvedCostBasis: &resolved,
	}}).validateCostBasis())

	costBasisID := "01J00000000000000000000000"
	require.NoError(t, (ChargeBase{Intent: intent, State: State{
		SchemaLevel:       SchemaLevelCostBasis,
		CostBasisID:       &costBasisID,
		ResolvedCostBasis: &resolved,
	}}).validateCostBasis())
}
