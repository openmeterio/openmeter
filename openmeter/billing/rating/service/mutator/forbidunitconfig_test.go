package mutator

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/billing/rating/service/rate"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

func TestForbidUnitConfigMutator(t *testing.T) {
	unitPrice := productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(1)})

	newInput := func(unitConfig *productcatalog.UnitConfig) rate.PricerCalculateInput {
		return rate.PricerCalculateInput{
			ProgressiveBilledLineAccessor: unitConfigTestLine{
				StandardLineWithSplitLineHierarchy: billing.StandardLineWithSplitLineHierarchy{
					StandardLine: &billing.StandardLine{},
				},
				price:      unitPrice,
				unitConfig: unitConfig,
			},
			Usage: &rating.Usage{
				Quantity:              alpacadecimal.NewFromFloat(1400),
				PreLinePeriodQuantity: alpacadecimal.NewFromFloat(0),
			},
		}
	}

	t.Run("no unit_config is a no-op", func(t *testing.T) {
		out, err := (&ForbidUnitConfig{}).Mutate(newInput(nil))
		require.NoError(t, err)

		usage, err := out.GetUsage()
		require.NoError(t, err)
		require.Equal(t, float64(1400), usage.Quantity.InexactFloat64())
	})

	t.Run("unit_config present errors instead of billing raw", func(t *testing.T) {
		// With the feature disabled a config must never be silently dropped: rating
		// the raw quantity would under/over-bill, so surface it as an error.
		_, err := (&ForbidUnitConfig{}).Mutate(newInput(newUnitConfig(opDivide, 1000, roundCeiling)))
		require.ErrorIs(t, err, ErrUnitConfigDisabled)
	})
}
