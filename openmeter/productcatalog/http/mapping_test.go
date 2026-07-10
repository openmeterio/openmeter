package http

import (
	"testing"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
)

// FromRateCard is the single v1 serialization choke point for rate cards. It must fail closed on a
// unit_config-bearing card (the v1 types cannot represent it) so no v1 surface silently strips the
// conversion, while a plain card still serializes normally.
func TestFromRateCardRejectsUnitConfig(t *testing.T) {
	cadence := datetime.NewISODuration(0, 1, 0, 0, 0, 0, 0)

	t.Run("unit_config card is rejected with the typed code", func(t *testing.T) {
		rc := &productcatalog.UsageBasedRateCard{
			RateCardMeta: productcatalog.RateCardMeta{
				Key:        "feat-1",
				Name:       "Feature 1",
				FeatureKey: lo.ToPtr("feat-1"),
				Price:      productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: decimal.NewFromInt(1)}),
				UnitConfig: &productcatalog.UnitConfig{
					Operation:        productcatalog.UnitConfigOperationDivide,
					ConversionFactor: decimal.NewFromInt(1000),
				},
			},
			BillingCadence: cadence,
		}

		_, err := FromRateCard(rc)
		require.Error(t, err)

		issues, convErr := models.AsValidationIssues(err)
		require.NoError(t, convErr)
		require.Len(t, issues, 1)
		assert.Equal(t, productcatalog.ErrCodeUnitConfigNotRepresentable, issues[0].Code())
	})

	t.Run("plain card serializes without error", func(t *testing.T) {
		rc := &productcatalog.FlatFeeRateCard{
			RateCardMeta: productcatalog.RateCardMeta{
				Key:   "flat-1",
				Name:  "Flat 1",
				Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{Amount: decimal.NewFromInt(10), PaymentTerm: productcatalog.InArrearsPaymentTerm}),
			},
			BillingCadence: &cadence,
		}

		_, err := FromRateCard(rc)
		require.NoError(t, err)
	})
}
