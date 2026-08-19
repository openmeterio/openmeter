package feature_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/filter"
)

func TestFeatureReferenceValidate(t *testing.T) {
	tests := []struct {
		name      string
		reference feature.FeatureReference
		wantError bool
	}{
		{name: "missing identity", reference: feature.FeatureReference{}, wantError: true},
		{name: "empty id", reference: feature.FeatureReference{ID: lo.ToPtr("")}, wantError: true},
		{name: "empty key", reference: feature.FeatureReference{Key: lo.ToPtr("")}, wantError: true},
		{name: "id only", reference: feature.FeatureReference{ID: lo.ToPtr("feature-id")}},
		{name: "key only", reference: feature.FeatureReference{Key: lo.ToPtr("feature-key")}},
		{name: "both", reference: feature.FeatureReference{ID: lo.ToPtr("feature-id"), Key: lo.ToPtr("feature-key")}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.reference.Validate()
			if test.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestFeatureReferenceWithFeature(t *testing.T) {
	resolved := newValidFeature()

	for _, reference := range []feature.FeatureReference{
		{ID: lo.ToPtr(resolved.ID)},
		{Key: lo.ToPtr(resolved.Key)},
		{ID: lo.ToPtr(resolved.ID), Key: lo.ToPtr(resolved.Key)},
	} {
		hydrated, err := reference.WithFeature(&resolved)
		require.NoError(t, err)
		require.True(t, hydrated.IsResolved())
		require.Equal(t, resolved.ID, *hydrated.ID)
		require.Equal(t, resolved.Key, *hydrated.Key)
		actual, ok := hydrated.Feature()
		require.True(t, ok)
		require.Same(t, &resolved, actual)
	}

	_, err := (feature.FeatureReference{ID: lo.ToPtr("other-id")}).WithFeature(&resolved)
	require.ErrorContains(t, err, "id mismatch")

	_, err = (feature.FeatureReference{Key: lo.ToPtr("other-key")}).WithFeature(&resolved)
	require.ErrorContains(t, err, "key mismatch")

	_, err = (feature.FeatureReference{Key: lo.ToPtr(resolved.Key)}).WithFeature(nil)
	require.ErrorContains(t, err, "feature is required")
}

func TestFeatureReferenceCompatible(t *testing.T) {
	complete := feature.FeatureReference{ID: lo.ToPtr("feature-id"), Key: lo.ToPtr("feature-key")}

	assert.True(t, complete.Compatible(feature.FeatureReference{ID: lo.ToPtr("feature-id")}))
	assert.True(t, complete.Compatible(feature.FeatureReference{Key: lo.ToPtr("feature-key")}))
	assert.True(t, complete.Compatible(complete))
	assert.False(t, complete.Compatible(feature.FeatureReference{ID: lo.ToPtr("other-id")}))
	assert.False(t, complete.Compatible(feature.FeatureReference{Key: lo.ToPtr("other-key")}))
}

func TestFeatureReferenceIdentityAndClone(t *testing.T) {
	resolved := newValidFeature()
	reference, err := (feature.FeatureReference{Key: lo.ToPtr(resolved.Key)}).WithFeature(&resolved)
	require.NoError(t, err)

	unresolved := feature.FeatureReference{ID: lo.ToPtr(resolved.ID), Key: lo.ToPtr(resolved.Key)}
	require.True(t, reference.Equal(unresolved), "runtime hydration must not change reference identity")

	clone := reference.Clone()
	clonedFeature, ok := clone.Feature()
	require.True(t, ok)
	clonedFeature.Metadata["owner"] = "clone"
	*clonedFeature.MeterGroupByFilters["model"].Eq = "clone-model"
	*clone.ID = "clone-id"

	originalFeature, ok := reference.Feature()
	require.True(t, ok)
	assert.Equal(t, "original", originalFeature.Metadata["owner"])
	assert.Equal(t, "original-model", *originalFeature.MeterGroupByFilters["model"].Eq)
	assert.Equal(t, resolved.ID, *reference.ID)
}

func TestFeatureReferenceJSONOmitsResolvedFeature(t *testing.T) {
	reference := newValidFeature().Reference()

	data, err := json.Marshal(reference)
	require.NoError(t, err)
	assert.JSONEq(t, `{"id":"feature-id","key":"feature-key"}`, string(data))
}

func newValidFeature() feature.Feature {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)

	return feature.Feature{
		Namespace: "namespace",
		ID:        "feature-id",
		Name:      "Feature",
		Key:       "feature-key",
		MeterGroupByFilters: feature.MeterGroupByFilters{
			"model": filter.FilterString{Eq: lo.ToPtr("original-model")},
		},
		Metadata:  map[string]string{"owner": "original"},
		CreatedAt: now,
		UpdatedAt: now,
	}
}
