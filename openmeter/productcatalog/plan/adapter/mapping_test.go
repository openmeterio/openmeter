package adapter

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

func TestAsPlanRateCardRowValidatesFeatureReferenceForPersistence(t *testing.T) {
	featureID := "feature-id"
	featureKey := "feature-key"

	tests := []struct {
		name      string
		reference *productcatalog.FeatureReference
		wantError string
	}{
		{name: "missing identity", reference: &productcatalog.FeatureReference{}, wantError: "id or key is required"},
		{name: "id only", reference: productcatalog.NewFeatureReference(&featureID, nil), wantError: "must include both id and key"},
		{name: "key only", reference: productcatalog.NewFeatureReference(nil, &featureKey), wantError: "must include both id and key"},
		{name: "complete", reference: productcatalog.NewFeatureReference(&featureID, &featureKey)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rateCard := &productcatalog.FlatFeeRateCard{
				RateCardMeta: productcatalog.RateCardMeta{Feature: tt.reference},
			}

			row, err := asPlanRateCardRow(rateCard)
			if tt.wantError != "" {
				require.ErrorContains(t, err, tt.wantError)
				return
			}

			require.NoError(t, err)
			require.Equal(t, featureID, *row.FeatureID)
			require.Equal(t, featureKey, *row.FeatureKey)
		})
	}
}
