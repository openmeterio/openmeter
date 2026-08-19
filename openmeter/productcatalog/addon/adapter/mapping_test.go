package adapter

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

func TestAsAddonRateCardRowRejectsInvalidFeatureReference(t *testing.T) {
	rateCard := &productcatalog.FlatFeeRateCard{
		RateCardMeta: productcatalog.RateCardMeta{
			Feature: &productcatalog.FeatureReference{},
		},
	}

	_, err := asAddonRateCardRow(rateCard)
	require.ErrorContains(t, err, "id or key is required")
}
