package costbasis

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestIntentClone(t *testing.T) {
	usd, err := currencyx.NewFiatCurrency("USD")
	require.NoError(t, err)

	eur, err := currencyx.NewFiatCurrency("EUR")
	require.NoError(t, err)

	t.Run("dynamic", func(t *testing.T) {
		original := NewIntent(DynamicIntent{FiatCurrency: usd})
		cloned := original.Clone()

		require.NotSame(t, original.dynamic, cloned.dynamic)
		cloned.dynamic.FiatCurrency = eur
		require.Same(t, usd, original.dynamic.FiatCurrency)
	})

	t.Run("pinned", func(t *testing.T) {
		original := NewIntent(PinnedIntent{
			FiatCurrency:        usd,
			CurrencyCostBasisID: "cost-basis-1",
		})
		cloned := original.Clone()

		require.NotSame(t, original.pinned, cloned.pinned)
		cloned.pinned.CurrencyCostBasisID = "cost-basis-2"
		require.Equal(t, "cost-basis-1", original.pinned.CurrencyCostBasisID)
	})

	t.Run("manual", func(t *testing.T) {
		original := NewIntent(ManualIntent{
			FiatCurrency: usd,
			Rate:         alpacadecimal.NewFromInt(2),
		})
		cloned := original.Clone()

		require.NotSame(t, original.manual, cloned.manual)
		cloned.manual.Rate = alpacadecimal.NewFromInt(3)
		require.Equal(t, float64(2), original.manual.Rate.InexactFloat64())
	})
}
