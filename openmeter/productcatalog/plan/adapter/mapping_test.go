package adapter

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

func TestAsPlanRateCardRowRejectsInvalidFeatureReference(t *testing.T) {
	rateCard := &productcatalog.FlatFeeRateCard{
		RateCardMeta: productcatalog.RateCardMeta{
			Feature: &productcatalog.FeatureReference{},
		},
	}

	_, err := asPlanRateCardRow(rateCard)
	require.ErrorContains(t, err, "id or key is required")
}
