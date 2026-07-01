package reconciler

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

// A subscription-created usage-based charge must snapshot the rate card's unit_config
// onto its intent; otherwise a plan using unit_config would rate the raw metered
// quantity (only direct charge-intent tests would exercise conversion). The intent →
// converted invoice amount is covered by the charges service integration suite.
func TestUsageBasedChargeIntentSnapshotsUnitConfig(t *testing.T) {
	unitConfig := &productcatalog.UnitConfig{
		Operation:        productcatalog.UnitConfigOperationDivide,
		ConversionFactor: alpacadecimal.NewFromInt(1000),
		Rounding:         productcatalog.UnitConfigRoundingModeCeiling,
	}

	t.Run("copies the rate card unit_config onto the usage-based intent", func(t *testing.T) {
		rateCard := newChargePatchTestUsageRateCard()
		rateCard.(*productcatalog.UsageBasedRateCard).UnitConfig = unitConfig

		target := newChargePatchTestTarget(t, productcatalog.CreditThenInvoiceSettlementMode, rateCard)

		intent, err := newUsageBasedChargeIntent(target)
		require.NoError(t, err)

		ubIntent, err := intent.AsUsageBasedIntent()
		require.NoError(t, err)
		require.NotNil(t, ubIntent.UnitConfig)
		require.True(t, unitConfig.Equal(ubIntent.UnitConfig))
		// Cloned, not aliased to the rate card's config.
		require.NotSame(t, unitConfig, ubIntent.UnitConfig)
	})

	t.Run("leaves the intent unit_config nil when the rate card has none", func(t *testing.T) {
		target := newChargePatchTestTarget(t, productcatalog.CreditThenInvoiceSettlementMode, newChargePatchTestUsageRateCard())

		intent, err := newUsageBasedChargeIntent(target)
		require.NoError(t, err)

		ubIntent, err := intent.AsUsageBasedIntent()
		require.NoError(t, err)
		require.Nil(t, ubIntent.UnitConfig)
	})
}
