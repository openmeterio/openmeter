package rate_test

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

// TestTieredOrderIndependence is the control for the storage-order defect in
// GraduatedTiered.TieredPriceCalculator.
//
// The tier walk reads in.TieredPrice.WithSortedTiers().Tiers; the zero-quantity
// fallback reads in.TieredPrice.Tiers[0] directly while labelling the range
// TierIndex: 0. TieredPrice.Validate() never checks tier order, and
// WithSortedTiers() exists precisely because order is not an invariant, so a
// zero-usage period must bill the lowest tier's mandatory flat fee regardless
// of the order the tiers are stored in.
//
// The fixture is the same four tiers as TestTieredPriceCalculator's testIn.
func TestTieredOrderIndependence(t *testing.T) {
	ascending := []productcatalog.PriceTier{
		{
			UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(5)),
			FlatPrice:  &productcatalog.PriceTierFlatPrice{Amount: alpacadecimal.NewFromFloat(100)},
		},
		{
			UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(10)),
			FlatPrice:  &productcatalog.PriceTierFlatPrice{Amount: alpacadecimal.NewFromFloat(50)},
		},
		{
			UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(15)),
			UnitPrice:  &productcatalog.PriceTierUnitPrice{Amount: alpacadecimal.NewFromFloat(10)},
		},
		{
			UnitPrice: &productcatalog.PriceTierUnitPrice{Amount: alpacadecimal.NewFromFloat(5)},
		},
	}

	reversed := make([]productcatalog.PriceTier, len(ascending))
	for i, tier := range ascending {
		reversed[len(ascending)-1-i] = tier
	}

	mk := func(tiers []productcatalog.PriceTier) productcatalog.TieredPrice {
		return productcatalog.TieredPrice{
			Mode:  productcatalog.GraduatedTieredPrice,
			Tiers: tiers,
		}
	}

	// If this fails, tier order IS an invariant and the premise is wrong.
	asc := mk(ascending)
	rev := mk(reversed)
	require.NoError(t, asc.Validate(), "ascending must validate")
	require.NoError(t, rev.Validate(),
		"REVERSED MUST ALSO VALIDATE: if this fails, order is enforced and there is no defect")

	zero := alpacadecimal.NewFromFloat(0)

	t.Run("zero usage, tiers stored ascending", func(t *testing.T) {
		got := getTotalAmountForGraduatedTieredPrice(t, zero, mk(ascending))
		require.Equal(t, alpacadecimal.NewFromFloat(100).String(), got.String())
	})

	t.Run("zero usage, tiers stored in reverse order", func(t *testing.T) {
		got := getTotalAmountForGraduatedTieredPrice(t, zero, mk(reversed))
		require.Equal(t, alpacadecimal.NewFromFloat(100).String(), got.String(),
			"the mandatory first-bucket flat fee must not depend on storage order")
	})
}
