package featuremeterservice

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/featuremeter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/ref"
)

type featureMeterReference struct {
	featuremeter.FeatureMeterRef
}

func (r featureMeterReference) GetFeatureMeterRef() *featuremeter.FeatureMeterRef {
	return &r.FeatureMeterRef
}

type identifiedFeatureMeterRef struct {
	featuremeter.FeatureMeterRef
	identity featuremeter.FeatureReferenceIdentity
}

func (r identifiedFeatureMeterRef) GetFeatureMeterRef() *featuremeter.FeatureMeterRef {
	return &r.FeatureMeterRef
}

func (r identifiedFeatureMeterRef) GetFeatureMeterOwner() featuremeter.FeatureReferenceIdentity {
	return r.identity
}

func featureMeterRef(reference featuremeter.FeatureMeterRef) featureMeterReference {
	return featureMeterReference{FeatureMeterRef: reference}
}

var (
	_ featuremeter.FeatureReferenceGetter = featureMeterReference{}
	_ featuremeter.FeatureReferenceGetter = identifiedFeatureMeterRef{}
	_ featuremeter.FeatureReferenceOwner  = identifiedFeatureMeterRef{}
)

func TestFeatureMeterCollectionGet(t *testing.T) {
	featureMeters := FeatureMeterCollection{
		ByKey: map[string]featuremeter.FeatureMeter{
			"tokens": {
				Feature: feature.Feature{ID: "feature-new", Key: "tokens"},
			},
			"requests": {
				Feature: feature.Feature{ID: "feature-other", Key: "requests"},
			},
		},
		ByID: map[string]featuremeter.FeatureMeter{
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
		byKeyRef := featureMeterRef(featuremeter.FeatureMeterRef{IDOrKey: ref.IDOrKey{Key: "tokens"}})
		byKey, err := featureMeters.Get(byKeyRef)
		require.NoError(t, err)
		require.Equal(t, "feature-new", byKey.Feature.ID)
		require.True(t, featureMeters.Has(byKeyRef))
		require.False(t, featureMeters.Has(featureMeterRef(featuremeter.FeatureMeterRef{IDOrKey: ref.IDOrKey{Key: "missing"}})))

		byLatestIDRef := featureMeterRef(featuremeter.FeatureMeterRef{IDOrKey: ref.IDOrKey{ID: "feature-new"}})
		byLatestID, err := featureMeters.Get(byLatestIDRef)
		require.NoError(t, err)
		require.Equal(t, "feature-new", byLatestID.Feature.ID)
		require.True(t, featureMeters.Has(byLatestIDRef))
		require.False(t, featureMeters.Has(featureMeterRef(featuremeter.FeatureMeterRef{IDOrKey: ref.IDOrKey{ID: "missing"}})))

		byOldID, err := featureMeters.Get(featureMeterRef(featuremeter.FeatureMeterRef{IDOrKey: ref.IDOrKey{ID: "feature-old"}}))
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
		// - the existing ID resolves and the missing ID returns a validation issue without claiming caller ownership
		existing, existingErr := featureMeters.Get(featureMeterRef(featuremeter.FeatureMeterRef{
			IDOrKey: ref.IDOrKey{ID: "feature-old"},
		}))
		_, missingErr := featureMeters.Get(featureMeterRef(featuremeter.FeatureMeterRef{
			IDOrKey: ref.IDOrKey{ID: "missing-feature"},
		}))

		require.NoError(t, existingErr)
		require.Equal(t, "feature-old", existing.Feature.ID)
		require.ErrorIs(t, missingErr, billing.ErrInvoiceLineFeatureNotFound)
		issues, systemErr := billing.ToValidationIssues(missingErr)
		require.NoError(t, systemErr)
		require.Equal(t, billing.ValidationIssues{{
			Severity: billing.ValidationIssueSeverityCritical,
			Code:     billing.ErrInvoiceLineFeatureNotFound.Code,
			Message:  "feature[missing-feature]: invoice line: feature not found",
		}}, issues)
	})

	t.Run("identified missing feature returns a line validation issue", func(t *testing.T) {
		// given:
		// - a line-scoped reference to a feature absent from the resolved collection
		// when:
		// - the reference is resolved
		// then:
		// - the missing feature is returned as a validation issue on that line without claiming caller ownership
		_, err := featureMeters.Get(identifiedFeatureMeterRef{
			FeatureMeterRef: featuremeter.FeatureMeterRef{
				IDOrKey: ref.IDOrKey{Key: "missing-feature"},
			},
			identity: featuremeter.FeatureReferenceIdentity{
				Kind: featuremeter.FeatureReferenceKindLines,
				ID:   "line-id",
			},
		})

		require.ErrorIs(t, err, billing.ErrInvoiceLineFeatureNotFound)
		issues, systemErr := billing.ToValidationIssues(err)
		require.NoError(t, systemErr)
		require.Equal(t, billing.ValidationIssues{{
			Severity: billing.ValidationIssueSeverityCritical,
			Code:     billing.ErrInvoiceLineFeatureNotFound.Code,
			Message:  "feature[missing-feature]: invoice line: feature not found",
			Path:     "/lines/line-id",
		}}, issues)
	})

	t.Run("required meter rejects a meterless feature", func(t *testing.T) {
		// given:
		// - a resolved feature without an associated meter
		// when:
		// - the feature is resolved with a meter requirement
		// then:
		// - resolution returns a validation issue without claiming caller ownership
		featureMeter, err := featureMeters.Get(featureMeterRef(featuremeter.FeatureMeterRef{
			IDOrKey:      ref.IDOrKey{Key: "requests"},
			RequireMeter: true,
		}))

		require.ErrorIs(t, err, billing.ErrInvoiceLineFeatureHasNoMeters)
		issues, systemErr := billing.ToValidationIssues(err)
		require.NoError(t, systemErr)
		require.Equal(t, billing.ValidationIssues{{
			Severity: billing.ValidationIssueSeverityCritical,
			Code:     billing.ErrInvoiceLineFeatureHasNoMeters.Code,
			Message:  "feature[requests]: usage based invoice line: feature has no meters",
		}}, issues)
		require.Equal(t, "feature-other", featureMeter.Feature.ID)
	})

	t.Run("identified meterless feature returns the feature and a charge validation issue", func(t *testing.T) {
		// given:
		// - a charge-scoped reference that requires the meterless requests feature
		// when:
		// - the reference is resolved
		// then:
		// - the feature remains usable and the missing meter is attached to the charge
		featureMeter, err := featureMeters.Get(identifiedFeatureMeterRef{
			FeatureMeterRef: featuremeter.FeatureMeterRef{
				IDOrKey:      ref.IDOrKey{Key: "requests"},
				RequireMeter: true,
			},
			identity: featuremeter.FeatureReferenceIdentity{
				Kind: featuremeter.FeatureReferenceKindCharges,
				ID:   "charge-id",
			},
		})

		require.ErrorIs(t, err, billing.ErrInvoiceLineFeatureHasNoMeters)
		issues, systemErr := billing.ToValidationIssues(err)
		require.NoError(t, systemErr)
		require.Equal(t, "feature-other", featureMeter.Feature.ID)
		require.Equal(t, billing.ValidationIssues{{
			Severity: billing.ValidationIssueSeverityCritical,
			Code:     billing.ErrInvoiceLineFeatureHasNoMeters.Code,
			Message:  "feature[requests]: usage based invoice line: feature has no meters",
			Path:     "/charges/charge-id",
		}}, issues)
	})

	t.Run("ID takes precedence over key", func(t *testing.T) {
		featureRef := featureMeterRef(featuremeter.FeatureMeterRef{
			IDOrKey: ref.IDOrKey{ID: "feature-old", Key: "requests"},
		})

		featureMeter, err := featureMeters.Get(featureRef)

		require.NoError(t, err)
		require.Equal(t, "feature-old", featureMeter.Feature.ID)
		require.True(t, featureMeters.Has(featureRef))
		require.False(t, featureMeters.Has(featureMeterRef(featuremeter.FeatureMeterRef{
			IDOrKey: ref.IDOrKey{ID: "missing", Key: "tokens"},
		})))
	})

	t.Run("empty reference is required", func(t *testing.T) {
		_, nilErr := featureMeters.Get(nil)
		_, emptyErr := featureMeters.Get(featureMeterRef(featuremeter.FeatureMeterRef{}))

		require.True(t, models.IsGenericValidationError(nilErr))
		require.ErrorContains(t, nilErr, "feature reference is required")
		require.True(t, models.IsGenericValidationError(emptyErr))
		require.ErrorContains(t, emptyErr, "feature reference is required")
		require.False(t, featureMeters.Has(nil))
		require.False(t, featureMeters.Has(featureMeterRef(featuremeter.FeatureMeterRef{})))
	})
}
