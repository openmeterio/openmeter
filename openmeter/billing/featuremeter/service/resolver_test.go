package featuremeterservice

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/featuremeter"
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

	newResolver := func(t *testing.T, expectedFeatureIDsOrKeys, expectedMeterIDs []string) *Resolver {
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

		resolved, err := resolver.Resolve(
			t.Context(),
			namespace,
			[]featureMeterReference{
				featureMeterRef(featuremeter.FeatureMeterRef{IDOrKey: ref.IDOrKey{Key: "tokens"}, RequireMeter: true}),
				featureMeterRef(featuremeter.FeatureMeterRef{IDOrKey: ref.IDOrKey{ID: "feature-old"}, RequireMeter: true}),
			}...,
		)
		require.NoError(t, err)

		byKey, err := resolved.Get(featureMeterRef(featuremeter.FeatureMeterRef{
			IDOrKey:      ref.IDOrKey{Key: "tokens"},
			RequireMeter: true,
		}))
		require.NoError(t, err)
		require.Equal(t, "feature-new", byKey.Feature.ID)
		require.Equal(t, meterID, byKey.Meter.ID)

		byArchivedID, err := resolved.Get(featureMeterRef(featuremeter.FeatureMeterRef{
			IDOrKey:      ref.IDOrKey{ID: "feature-old"},
			RequireMeter: true,
		}))
		require.NoError(t, err)
		require.Equal(t, "feature-old", byArchivedID.Feature.ID)
		require.Equal(t, meterID, byArchivedID.Meter.ID)
	})

	t.Run("returns partial results alongside validation errors", func(t *testing.T) {
		// given:
		// - one resolvable feature and one missing feature
		targets := []featureMeterReference{
			featureMeterRef(featuremeter.FeatureMeterRef{IDOrKey: ref.IDOrKey{Key: "tokens"}, RequireMeter: true}),
			featureMeterRef(featuremeter.FeatureMeterRef{IDOrKey: ref.IDOrKey{Key: "missing-feature"}}),
		}
		resolver := newResolver(t, []string{"tokens", "missing-feature"}, []string{meterID})

		// when:
		// - both feature dependencies are resolved
		resolved, err := resolver.Resolve(t.Context(), namespace, targets...)

		// then:
		// - the resolved feature remains usable and the error contains validation issues only
		require.NotNil(t, resolved)
		resolvedFeature, getErr := resolved.Get(targets[0])
		require.NoError(t, getErr)
		require.Equal(t, "feature-new", resolvedFeature.Feature.ID)

		issues, systemErr := billing.ToValidationIssues(err)
		require.NoError(t, systemErr)
		require.Equal(t, billing.ValidationIssues{{
			Severity: billing.ValidationIssueSeverityCritical,
			Code:     billing.ErrInvoiceLineFeatureNotFound.Code,
			Message:  "feature[missing-feature]: invoice line: feature not found",
		}}, issues)
	})

	t.Run("returns meterless feature alongside validation error", func(t *testing.T) {
		// given:
		// - a feature exists but has no meter
		resolver := newResolver(t, []string{"requests"}, nil)
		target := featureMeterRef(featuremeter.FeatureMeterRef{
			IDOrKey:      ref.IDOrKey{Key: "requests"},
			RequireMeter: true,
		})

		// when:
		// - the target requires a meter
		resolved, err := resolver.Resolve(t.Context(), namespace, target)

		// then:
		// - the feature remains addressable and the missing meter is a validation-only error
		require.NotNil(t, resolved)
		resolvedFeature, getErr := resolved.Get(featureMeterRef(featuremeter.FeatureMeterRef{IDOrKey: target.IDOrKey}))
		require.NoError(t, getErr)
		require.Equal(t, "feature-other", resolvedFeature.Feature.ID)

		issues, systemErr := billing.ToValidationIssues(err)
		require.NoError(t, systemErr)
		require.Equal(t, billing.ValidationIssues{{
			Severity: billing.ValidationIssueSeverityCritical,
			Code:     billing.ErrInvoiceLineFeatureHasNoMeters.Code,
			Message:  "feature[requests]: usage based invoice line: feature has no meters",
		}}, issues)
	})

	t.Run("deduplicated lookup validates every original identified reference", func(t *testing.T) {
		// given:
		// - two gathering lines reference the same missing feature
		resolver := newResolver(t, []string{"missing-feature"}, nil)
		references := []identifiedFeatureMeterRef{
			{
				FeatureMeterRef: featuremeter.FeatureMeterRef{IDOrKey: ref.IDOrKey{Key: "missing-feature"}},
				identity: featuremeter.FeatureReferenceIdentity{
					Kind: featuremeter.FeatureReferenceKindLines,
					ID:   "line-1",
				},
			},
			{
				FeatureMeterRef: featuremeter.FeatureMeterRef{IDOrKey: ref.IDOrKey{Key: "missing-feature"}},
				identity: featuremeter.FeatureReferenceIdentity{
					Kind: featuremeter.FeatureReferenceKindLines,
					ID:   "line-2",
				},
			},
		}

		// when:
		// - feature meters are resolved
		resolved, err := resolver.Resolve(t.Context(), namespace, references...)

		// then:
		// - the catalog lookup is deduplicated while each original line receives an issue
		require.NotNil(t, resolved)
		issues, systemErr := billing.ToValidationIssues(err)
		require.NoError(t, systemErr)
		require.ElementsMatch(t, []string{"/lines/line-1", "/lines/line-2"}, lo.Map(issues, func(issue billing.ValidationIssue, _ int) string {
			return issue.Path
		}))
		for _, issue := range issues {
			require.Equal(t, billing.ErrInvoiceLineFeatureNotFound.Code, issue.Code)
			require.Empty(t, issue.Component)
		}
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

	// Given a feature reference without the namespace required to scope catalog lookups.
	featureRefs := []featureMeterReference{featureMeterRef(featuremeter.FeatureMeterRef{IDOrKey: ref.IDOrKey{Key: "tokens"}})}

	// When feature meters are resolved.
	_, err = resolver.Resolve(t.Context(), "", featureRefs...)

	// Then validation fails before either backing service is called.
	require.True(t, models.IsGenericValidationError(err))
	require.ErrorContains(t, err, "namespace is required")
}
