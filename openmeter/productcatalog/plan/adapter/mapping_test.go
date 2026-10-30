package adapter

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
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

func TestFromPlanRateCardRowHydratesFeatureReference(t *testing.T) {
	featureID := "feature-id"
	featureKey := "feature-key"
	now := time.Date(2026, time.August, 24, 0, 0, 0, 0, time.UTC)

	newRow := func(feature *entdb.Feature) entdb.PlanRateCard {
		return entdb.PlanRateCard{
			Key:        "rate-card",
			Name:       "Rate card",
			Type:       productcatalog.FlatFeeRateCardType,
			FeatureID:  &featureID,
			FeatureKey: &featureKey,
			Edges: entdb.PlanRateCardEdges{
				Features: feature,
			},
		}
	}

	t.Run("matching feature", func(t *testing.T) {
		feature := &entdb.Feature{
			ID:        featureID,
			Namespace: "default",
			Name:      "Feature",
			Key:       featureKey,
			CreatedAt: now,
			UpdatedAt: now,
		}

		rateCard, err := fromPlanRateCardRow(newRow(feature))
		require.NoError(t, err)

		reference := rateCard.AsMeta().Feature
		require.NotNil(t, reference)
		require.True(t, reference.IsResolved())

		resolvedFeature, ok := reference.Feature()
		require.True(t, ok)
		require.Equal(t, featureID, resolvedFeature.ID)
		require.Equal(t, featureKey, resolvedFeature.Key)
	})

	t.Run("conflicting feature", func(t *testing.T) {
		feature := &entdb.Feature{
			ID:        "other-feature-id",
			Namespace: "default",
			Name:      "Feature",
			Key:       featureKey,
			CreatedAt: now,
			UpdatedAt: now,
		}

		_, err := fromPlanRateCardRow(newRow(feature))
		require.ErrorContains(t, err, "invalid resolved feature reference")
		require.ErrorContains(t, err, "id mismatch")
	})
}

func TestRateCardRowsReadTaxFromColumns(t *testing.T) {
	const taxCodeID = "01JTAXCODE0000000000000000"

	// Old rows may still hold a Stripe code in the legacy JSON; only the columns count.
	legacyJSON := &productcatalog.TaxConfig{
		Stripe:    &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
		TaxCodeID: lo.ToPtr(taxCodeID),
	}

	tests := []struct {
		name        string
		taxCodeID   *string
		taxBehavior *productcatalog.TaxBehavior
		want        *productcatalog.TaxCodeConfig
	}{
		{name: "no columns set", want: nil},
		{name: "tax code ID", taxCodeID: lo.ToPtr(taxCodeID), want: &productcatalog.TaxCodeConfig{TaxCodeID: taxCodeID}},
		{
			name:        "behavior only",
			taxBehavior: lo.ToPtr(productcatalog.InclusiveTaxBehavior),
			want:        &productcatalog.TaxCodeConfig{Behavior: lo.ToPtr(productcatalog.InclusiveTaxBehavior)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			planRC, err := fromPlanRateCardRow(entdb.PlanRateCard{
				Key:         "rate-card",
				Name:        "Rate card",
				Type:        productcatalog.FlatFeeRateCardType,
				TaxConfig:   legacyJSON,
				TaxCodeID:   tt.taxCodeID,
				TaxBehavior: tt.taxBehavior,
			})
			require.NoError(t, err)
			require.Equal(t, tt.want, planRC.AsMeta().TaxConfig)

			addonRC, err := FromAddonRateCardRow(entdb.AddonRateCard{
				Key:         "rate-card",
				Name:        "Rate card",
				Type:        productcatalog.FlatFeeRateCardType,
				TaxConfig:   legacyJSON,
				TaxCodeID:   tt.taxCodeID,
				TaxBehavior: tt.taxBehavior,
			})
			require.NoError(t, err)
			require.Equal(t, tt.want, addonRC.AsMeta().TaxConfig)
		})
	}
}

func TestAsPlanRateCardRowWritesTaxColumnsAndLegacyCopy(t *testing.T) {
	const taxCodeID = "01JTAXCODE0000000000000000"

	tests := []struct {
		name string
		cfg  *productcatalog.TaxCodeConfig
	}{
		{name: "tax code ID only", cfg: &productcatalog.TaxCodeConfig{TaxCodeID: taxCodeID}},
		{name: "behavior only", cfg: &productcatalog.TaxCodeConfig{Behavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior)}},
		{
			name: "tax code ID and behavior",
			cfg:  &productcatalog.TaxCodeConfig{TaxCodeID: taxCodeID, Behavior: lo.ToPtr(productcatalog.InclusiveTaxBehavior)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			row, err := asPlanRateCardRow(&productcatalog.FlatFeeRateCard{
				RateCardMeta: productcatalog.RateCardMeta{TaxConfig: tt.cfg},
			})
			require.NoError(t, err)

			require.Equal(t, lo.EmptyableToPtr(tt.cfg.TaxCodeID), row.TaxCodeID)
			require.Equal(t, tt.cfg.Behavior, row.TaxBehavior)
			// The DB checks compare the JSON with the columns, so they must agree exactly.
			require.Equal(t, &productcatalog.TaxConfig{
				TaxCodeID: lo.EmptyableToPtr(tt.cfg.TaxCodeID),
				Behavior:  tt.cfg.Behavior,
			}, row.TaxConfig)
		})
	}

	t.Run("no tax config", func(t *testing.T) {
		row, err := asPlanRateCardRow(&productcatalog.FlatFeeRateCard{})
		require.NoError(t, err)

		require.Nil(t, row.TaxConfig)
		require.Nil(t, row.TaxCodeID)
		require.Nil(t, row.TaxBehavior)
	})
}
