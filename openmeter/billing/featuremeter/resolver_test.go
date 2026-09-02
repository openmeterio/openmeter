package featuremeter

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/ref"
)

type featureServiceStub struct {
	listFeatures func(ctx context.Context, params feature.ListFeaturesParams) (pagination.Result[feature.Feature], error)
}

func (s featureServiceStub) ListFeatures(ctx context.Context, params feature.ListFeaturesParams) (pagination.Result[feature.Feature], error) {
	return s.listFeatures(ctx, params)
}

type meterServiceStub struct {
	listMeters func(ctx context.Context, params meter.ListMetersParams) (pagination.Result[meter.Meter], error)
}

func (s meterServiceStub) ListMeters(ctx context.Context, params meter.ListMetersParams) (pagination.Result[meter.Meter], error) {
	return s.listMeters(ctx, params)
}

func TestGetLastFeatures(t *testing.T) {
	tests := []struct {
		name     string
		features []feature.Feature
		expected map[string]string
	}{
		{
			name: "single active",
			features: []feature.Feature{
				{ID: "id-active", Key: "feature-active"},
			},
			expected: map[string]string{"feature-active": "id-active"},
		},
		{
			name: "single archived",
			features: []feature.Feature{
				{ID: "id-archived", ArchivedAt: lo.ToPtr(time.Now()), Key: "feature-archived"},
			},
			expected: map[string]string{"feature-archived": "id-archived"},
		},
		{
			name: "active preferred over archived",
			features: []feature.Feature{
				{ID: "id-archived", ArchivedAt: lo.ToPtr(time.Now()), Key: "feature"},
				{ID: "id-active", Key: "feature"},
			},
			expected: map[string]string{"feature": "id-active"},
		},
		{
			name: "most recently archived preferred",
			features: []feature.Feature{
				{ID: "id-archived-1", ArchivedAt: lo.ToPtr(time.Now()), Key: "feature"},
				{ID: "id-archived-2", ArchivedAt: lo.ToPtr(time.Now().Add(5 * time.Second)), Key: "feature"},
			},
			expected: map[string]string{"feature": "id-archived-2"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resolved := getLastFeatures(test.features)

			featureKeyToID := make(map[string]string, len(resolved))
			for key, featureEntity := range resolved {
				featureKeyToID[key] = featureEntity.ID
			}

			require.Equal(t, test.expected, featureKeyToID)
		})
	}
}

func TestResolver(t *testing.T) {
	const namespace = "namespace"

	archivedAt := time.Now()
	meterID := "meter-id"
	features := []feature.Feature{
		{ID: "feature-old", Key: "tokens", MeterID: &meterID, ArchivedAt: &archivedAt},
		{ID: "feature-new", Key: "tokens", MeterID: &meterID},
		{ID: "feature-other", Key: "requests"},
	}

	newResolver := func(t *testing.T, expectedFeatureIDsOrKeys, expectedMeterIDs []string) Resolver {
		t.Helper()

		resolver, err := New(Config{
			FeatureService: featureServiceStub{
				listFeatures: func(_ context.Context, params feature.ListFeaturesParams) (pagination.Result[feature.Feature], error) {
					require.Equal(t, namespace, params.Namespace)
					require.ElementsMatch(t, expectedFeatureIDsOrKeys, params.IDsOrKeys)
					require.True(t, params.IncludeArchived)

					return pagination.Result[feature.Feature]{Items: lo.Filter(features, func(featureEntity feature.Feature, _ int) bool {
						return lo.Contains(params.IDsOrKeys, featureEntity.ID) || lo.Contains(params.IDsOrKeys, featureEntity.Key)
					})}, nil
				},
			},
			MeterService: meterServiceStub{
				listMeters: func(_ context.Context, params meter.ListMetersParams) (pagination.Result[meter.Meter], error) {
					require.Equal(t, namespace, params.Namespace)
					require.NotNil(t, params.IDFilter)
					require.ElementsMatch(t, expectedMeterIDs, *params.IDFilter)
					require.True(t, params.IncludeDeleted)

					meters := []meter.Meter{{
						ManagedResource: models.ManagedResource{ID: meterID},
					}}

					return pagination.Result[meter.Meter]{Items: lo.Filter(meters, func(meterEntity meter.Meter, _ int) bool {
						return lo.Contains(*params.IDFilter, meterEntity.ID)
					})}, nil
				},
			},
			Logger: slog.Default(),
		})
		require.NoError(t, err)

		return resolver
	}

	t.Run("key resolves latest while explicit IDs remain addressable", func(t *testing.T) {
		resolver := newResolver(t, []string{"tokens", "feature-old"}, []string{meterID})

		resolved, err := resolver.Resolve(t.Context(), ResolveInput{
			Namespace: namespace,
			FeatureRefs: []FeatureMeterRef{
				{IDOrKey: ref.IDOrKey{Key: "tokens"}, RequireMeter: true},
				{IDOrKey: ref.IDOrKey{ID: "feature-old"}, RequireMeter: true},
			},
		})
		require.NoError(t, err)

		byKey, err := resolved.GetByKey("tokens", true)
		require.NoError(t, err)
		require.Equal(t, "feature-new", byKey.Feature.ID)
		require.Equal(t, meterID, byKey.Meter.ID)

		byArchivedID, err := resolved.GetByID("feature-old", true)
		require.NoError(t, err)
		require.Equal(t, "feature-old", byArchivedID.Feature.ID)
		require.Equal(t, meterID, byArchivedID.Meter.ID)
	})

	t.Run("missing feature validation is optional", func(t *testing.T) {
		refs := []FeatureMeterRef{
			{IDOrKey: ref.IDOrKey{Key: "missing-feature"}},
			{IDOrKey: ref.IDOrKey{Key: "requests"}, RequireMeter: true},
		}

		strictResolver := newResolver(t, []string{"missing-feature", "requests"}, nil)
		_, strictErr := strictResolver.Resolve(t.Context(), ResolveInput{
			Namespace:   namespace,
			FeatureRefs: refs,
		})
		require.True(t, models.IsGenericNotFoundError(strictErr))

		missingOnlyResolver := newResolver(t, []string{"missing-feature"}, nil)
		missingOnly, missingOnlyErr := missingOnlyResolver.Resolve(
			t.Context(),
			ResolveInput{
				Namespace:   namespace,
				FeatureRefs: refs[:1],
			},
			WithAllowMissingFeatures(),
		)
		require.NoError(t, missingOnlyErr)
		_, missingErr := missingOnly.GetByKey("missing-feature", false)
		require.True(t, models.IsGenericNotFoundError(missingErr))

		allowMissingResolver := newResolver(t, []string{"missing-feature", "requests"}, nil)
		_, allowMissingErr := allowMissingResolver.Resolve(
			t.Context(),
			ResolveInput{
				Namespace:   namespace,
				FeatureRefs: refs,
			},
			WithAllowMissingFeatures(),
		)
		require.Error(t, allowMissingErr)
		require.False(t, models.IsGenericNotFoundError(allowMissingErr))
		require.True(t, models.IsGenericValidationError(allowMissingErr))
		require.ErrorContains(t, allowMissingErr, "feature[requests] has no meter associated")
	})
}

func TestResolverValidatesInput(t *testing.T) {
	resolver, err := New(Config{
		FeatureService: featureServiceStub{
			listFeatures: func(context.Context, feature.ListFeaturesParams) (pagination.Result[feature.Feature], error) {
				t.Fatal("feature service must not be called for invalid input")

				return pagination.Result[feature.Feature]{}, nil
			},
		},
		MeterService: meterServiceStub{
			listMeters: func(context.Context, meter.ListMetersParams) (pagination.Result[meter.Meter], error) {
				t.Fatal("meter service must not be called for invalid input")

				return pagination.Result[meter.Meter]{}, nil
			},
		},
		Logger: slog.Default(),
	})
	require.NoError(t, err)

	// Given an input without the namespace required to scope catalog lookups.
	input := ResolveInput{
		FeatureRefs: []FeatureMeterRef{{IDOrKey: ref.IDOrKey{Key: "tokens"}}},
	}

	// When feature meters are resolved.
	_, err = resolver.Resolve(t.Context(), input)

	// Then validation fails before either backing service is called.
	require.True(t, models.IsGenericValidationError(err))
	require.ErrorContains(t, err, "namespace is required")
}
