package feature

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/ref"
)

func TestGetLastFeatures(t *testing.T) {
	tcs := []struct {
		name     string
		features []Feature
		expected map[string]string
	}{
		{
			name: "single-active",
			features: []Feature{
				{ID: "id-active", ArchivedAt: nil, Key: "feature-1-active"},
			},
			expected: map[string]string{"feature-1-active": "id-active"},
		},
		{
			name: "single-archived",
			features: []Feature{
				{ID: "id-archived", ArchivedAt: lo.ToPtr(time.Now()), Key: "feature-1-archived"},
			},
			expected: map[string]string{"feature-1-archived": "id-archived"},
		},
		{
			name: "multi-archived",
			features: []Feature{
				{ID: "id-archived", ArchivedAt: lo.ToPtr(time.Now()), Key: "feature-1"},
				{ID: "id-active", ArchivedAt: nil, Key: "feature-1"},
			},
			expected: map[string]string{"feature-1": "id-active"},
		},
		{
			name: "archived-ordering",
			features: []Feature{
				{ID: "id-archived-1", ArchivedAt: lo.ToPtr(time.Now()), Key: "feature-1"},
				{ID: "id-archived-2", ArchivedAt: lo.ToPtr(time.Now().Add(5 * time.Second)), Key: "feature-1"},
			},
			expected: map[string]string{"feature-1": "id-archived-2"},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			out := getLastFeatures(tc.features)

			featureKeyToID := map[string]string{}
			for key, feat := range out {
				featureKeyToID[key] = feat.ID
			}

			require.Equal(t, tc.expected, featureKeyToID)
		})
	}
}

func TestResolveFeatureMeters(t *testing.T) {
	archivedAt := time.Now()

	features := []Feature{
		{ID: "feature-old", Key: "tokens", ArchivedAt: &archivedAt},
		{ID: "feature-new", Key: "tokens", ArchivedAt: nil},
		{ID: "feature-other", Key: "requests", ArchivedAt: nil},
	}

	t.Run("key resolves latest while explicit ids remain addressable", func(t *testing.T) {
		out := resolveFeatureMeters(features)

		byKey, err := out.Get("tokens", false)
		require.NoError(t, err)
		require.Equal(t, "feature-new", byKey.Feature.ID)

		byLatestID, err := out.GetByID("feature-new", false)
		require.NoError(t, err)
		require.Equal(t, "feature-new", byLatestID.Feature.ID)

		byArchivedID, err := out.GetByID("feature-old", false)
		require.NoError(t, err)
		require.Equal(t, "feature-old", byArchivedID.Feature.ID)
		require.Equal(t, "tokens", byArchivedID.Feature.Key)
	})

	t.Run("references must resolve", func(t *testing.T) {
		// given:
		// - a feature meter collection containing current and archived features
		// when:
		// - an existing and a missing feature ID are resolved
		// then:
		// - the existing ID resolves and the missing ID returns a not-found error
		out := resolveFeatureMeters(features)

		_, existingErr := out.Resolve(FeatureMeterRef{
			IDOrKey: ref.IDOrKey{ID: "feature-old"},
		})
		_, missingErr := out.Resolve(FeatureMeterRef{
			IDOrKey: ref.IDOrKey{ID: "missing-feature"},
		})

		require.NoError(t, existingErr)
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
		out := resolveFeatureMeters(features)

		_, err := out.Resolve(FeatureMeterRef{
			IDOrKey:      ref.IDOrKey{Key: "requests"},
			RequireMeter: true,
		})

		require.Error(t, err)
		require.True(t, models.IsGenericValidationError(err))
		require.ErrorContains(t, err, "has no meter associated")
	})
}
