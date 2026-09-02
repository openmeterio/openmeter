package featuremeter

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/ref"
)

func TestFeatureMeterCollectionResolve(t *testing.T) {
	featureMeters := FeatureMeterCollection{
		ByKey: map[string]FeatureMeter{
			"tokens": {
				Feature: feature.Feature{ID: "feature-new", Key: "tokens"},
			},
			"requests": {
				Feature: feature.Feature{ID: "feature-other", Key: "requests"},
			},
		},
		ByID: map[string]FeatureMeter{
			"feature-old": {
				Feature: feature.Feature{ID: "feature-old", Key: "tokens"},
			},
			"feature-new": {
				Feature: feature.Feature{ID: "feature-new", Key: "tokens"},
			},
			"feature-other": {
				Feature: feature.Feature{ID: "feature-other", Key: "requests"},
			},
		},
	}

	t.Run("key and explicit ids are addressable", func(t *testing.T) {
		byKey, err := featureMeters.GetByKey("tokens", false)
		require.NoError(t, err)
		require.Equal(t, "feature-new", byKey.Feature.ID)
		require.True(t, featureMeters.HasFeatureKey("tokens"))
		require.False(t, featureMeters.HasFeatureKey("missing"))

		byLatestID, err := featureMeters.GetByID("feature-new", false)
		require.NoError(t, err)
		require.Equal(t, "feature-new", byLatestID.Feature.ID)
		require.True(t, featureMeters.HasFeatureID("feature-new"))
		require.False(t, featureMeters.HasFeatureID("missing"))

		byOldID, err := featureMeters.GetByID("feature-old", false)
		require.NoError(t, err)
		require.Equal(t, "feature-old", byOldID.Feature.ID)
		require.Equal(t, "tokens", byOldID.Feature.Key)
	})

	t.Run("references must resolve", func(t *testing.T) {
		// given:
		// - a feature meter collection containing current and historical features
		// when:
		// - an existing and a missing feature ID are resolved
		// then:
		// - the existing ID resolves and the missing ID returns a not-found error
		existing, existingErr := featureMeters.Resolve(FeatureMeterRef{
			IDOrKey: ref.IDOrKey{ID: "feature-old"},
		})
		_, missingErr := featureMeters.Resolve(FeatureMeterRef{
			IDOrKey: ref.IDOrKey{ID: "missing-feature"},
		})

		require.NoError(t, existingErr)
		require.Equal(t, "feature-old", existing.Feature.ID)
		require.Error(t, missingErr)
		require.True(t, models.IsGenericNotFoundError(missingErr))
		require.ErrorContains(t, missingErr, "missing-feature")
	})

	t.Run("required meter rejects a meterless feature", func(t *testing.T) {
		// given:
		// - a resolved feature without an associated meter
		// when:
		// - the feature is resolved with a meter requirement
		// then:
		// - resolution returns a validation error
		_, err := featureMeters.Resolve(FeatureMeterRef{
			IDOrKey:      ref.IDOrKey{Key: "requests"},
			RequireMeter: true,
		})

		require.Error(t, err)
		require.True(t, models.IsGenericValidationError(err))
		require.ErrorContains(t, err, "has no meter associated")
	})

	t.Run("reference with ID and key is rejected", func(t *testing.T) {
		_, err := featureMeters.Resolve(FeatureMeterRef{
			IDOrKey: ref.IDOrKey{ID: "feature-old", Key: "tokens"},
		})

		require.ErrorContains(t, err, "either key or ID, not both")
	})

	t.Run("reference without ID or key is rejected", func(t *testing.T) {
		_, err := featureMeters.Resolve(FeatureMeterRef{})

		require.ErrorContains(t, err, "either key or ID")
	})
}
