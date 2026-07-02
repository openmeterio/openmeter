package service

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/billing/rating/service/mutator"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

func TestGetPricerForUnitConfigRegistration(t *testing.T) {
	unitLine := &billing.StandardLine{
		UsageBased: &billing.UsageBasedLine{
			Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(1)}),
		},
	}

	t.Run("flag off registers ForbidUnitConfig before DiscountUsage", func(t *testing.T) {
		pm, err := getPricerFor(unitLine, rating.NewGenerateDetailedLinesOptions(), false)
		require.NoError(t, err)
		require.Len(t, pm.PreCalculation, 2)
		require.IsType(t, &mutator.ForbidUnitConfig{}, pm.PreCalculation[0])
		require.IsType(t, &mutator.DiscountUsage{}, pm.PreCalculation[1])
	})

	t.Run("flag on registers UnitConfig before DiscountUsage", func(t *testing.T) {
		pm, err := getPricerFor(unitLine, rating.NewGenerateDetailedLinesOptions(), true)
		require.NoError(t, err)
		require.Len(t, pm.PreCalculation, 2)
		require.IsType(t, &mutator.UnitConfig{}, pm.PreCalculation[0])
		require.IsType(t, &mutator.DiscountUsage{}, pm.PreCalculation[1])
	})

	t.Run("flat price has no pre-calculation mutators regardless of flag", func(t *testing.T) {
		flatLine := &billing.StandardLine{
			UsageBased: &billing.UsageBasedLine{
				Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{Amount: alpacadecimal.NewFromInt(1)}),
			},
		}

		pm, err := getPricerFor(flatLine, rating.NewGenerateDetailedLinesOptions(), true)
		require.NoError(t, err)
		require.Empty(t, pm.PreCalculation)
	})
}
