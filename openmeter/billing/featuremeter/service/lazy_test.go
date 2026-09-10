package featuremeterservice

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/featuremeter"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/ref"
)

type mutableIdentifiedFeatureMeterReference struct {
	reference featuremeter.FeatureMeterRef
	identity  featuremeter.FeatureReferenceIdentity
}

func (r *mutableIdentifiedFeatureMeterReference) GetFeatureMeterRef() *featuremeter.FeatureMeterRef {
	return &r.reference
}

func (r *mutableIdentifiedFeatureMeterReference) GetFeatureMeterOwner() featuremeter.FeatureReferenceIdentity {
	return r.identity
}

func TestResolverResolveLazy(t *testing.T) {
	const namespace = "namespace"

	t.Run("resolves once when first queried", func(t *testing.T) {
		// given:
		// - an unresolved feature reference and concurrent consumers
		// when:
		// - the lazy collection is queried for the first time
		// then:
		// - the feature is resolved once and reused by every consumer
		var featureCalls atomic.Int32
		var featureIDsOrKeys []string
		resolver, err := New(Config{
			FeatureService: featureServiceStub{
				listFeatures: func(_ context.Context, params feature.ListFeaturesParams) (pagination.Result[feature.Feature], error) {
					featureCalls.Add(1)
					featureIDsOrKeys = params.IDsOrKeys

					return pagination.Result[feature.Feature]{Items: []feature.Feature{{ID: "feature-id", Key: "tokens"}}}, nil
				},
			},
			MeterService: meterServiceStub{
				listMeters: func(context.Context, meter.ListMetersParams) (pagination.Result[meter.Meter], error) {
					return pagination.Result[meter.Meter]{}, nil
				},
			},
			Logger: slog.Default(),
		})
		require.NoError(t, err)

		target := featureMeterRef(featuremeter.FeatureMeterRef{IDOrKey: ref.IDOrKey{Key: "tokens"}})
		resolved := resolver.ResolveLazy(t.Context(), namespace, target)
		require.Zero(t, featureCalls.Load())

		var waitGroup sync.WaitGroup
		hasResults := make(chan bool, 10)
		for range 10 {
			waitGroup.Add(1)
			go func() {
				defer waitGroup.Done()
				hasResults <- resolved.Has(target)
			}()
		}
		waitGroup.Wait()
		close(hasResults)
		for has := range hasResults {
			require.True(t, has)
		}

		featureMeter, err := resolved.Get(target)
		require.NoError(t, err)
		require.Equal(t, "feature-id", featureMeter.Feature.ID)
		require.Equal(t, []string{"tokens"}, featureIDsOrKeys)
		require.Equal(t, int32(1), featureCalls.Load())
	})

	t.Run("passes resolution failures through", func(t *testing.T) {
		// given:
		// - feature resolution returns a catalog error
		// when:
		// - the lazy collection is queried repeatedly
		// then:
		// - the original error is cached without querying meters
		resolutionFailure := errors.New("catalog unavailable")
		resolver, err := New(Config{
			FeatureService: featureServiceStub{
				listFeatures: func(context.Context, feature.ListFeaturesParams) (pagination.Result[feature.Feature], error) {
					return pagination.Result[feature.Feature]{}, resolutionFailure
				},
			},
			MeterService: meterServiceStub{
				listMeters: func(context.Context, meter.ListMetersParams) (pagination.Result[meter.Meter], error) {
					t.Fatal("meter service must not be called after feature resolution fails")

					return pagination.Result[meter.Meter]{}, nil
				},
			},
			Logger: slog.Default(),
		})
		require.NoError(t, err)

		target := featureMeterRef(featuremeter.FeatureMeterRef{IDOrKey: ref.IDOrKey{Key: "tokens"}})
		resolved := resolver.ResolveLazy(t.Context(), namespace, target)
		_, getErr := resolved.Get(target)
		require.ErrorIs(t, getErr, resolutionFailure)
		require.False(t, resolved.Has(target))

		_, repeatedGetErr := resolved.Get(target)
		require.Same(t, getErr, repeatedGetErr)
	})

	t.Run("snapshots references and usable identities", func(t *testing.T) {
		// given:
		// - a mutable reference with a complete owner identity
		// when:
		// - a snapshot is taken and the source reference changes
		// then:
		// - the snapshot retains the original reference and owner path
		target := &mutableIdentifiedFeatureMeterReference{
			reference: featuremeter.FeatureMeterRef{IDOrKey: ref.IDOrKey{Key: "original"}},
			identity: featuremeter.FeatureReferenceIdentity{
				Kind: featuremeter.FeatureReferenceKindCharges,
				ID:   "charge-id",
			},
		}

		snapshot := snapshotFeatureMeterReference(target)
		target.reference.IDOrKey.Key = "mutated"
		target.identity.ID = "mutated"

		require.Equal(t, "original", snapshot.GetFeatureMeterRef().IDOrKey.Key)
		_, err := (FeatureMeterCollection{}).Get(snapshot)
		issues, systemErr := billing.ToValidationIssues(err)
		require.NoError(t, systemErr)
		require.Equal(t, "/charges/charge-id", issues[0].Path)
	})

	t.Run("drops incomplete identities", func(t *testing.T) {
		// given:
		// - a reference whose owner identity has no ID
		// when:
		// - the reference is snapshotted
		// then:
		// - the incomplete owner identity is omitted
		target := &mutableIdentifiedFeatureMeterReference{
			reference: featuremeter.FeatureMeterRef{IDOrKey: ref.IDOrKey{Key: "tokens"}},
			identity:  featuremeter.FeatureReferenceIdentity{Kind: featuremeter.FeatureReferenceKindCharges},
		}

		snapshot := snapshotFeatureMeterReference(target)
		_, ok := snapshot.(featuremeter.FeatureReferenceOwner)
		require.False(t, ok)
	})
}
