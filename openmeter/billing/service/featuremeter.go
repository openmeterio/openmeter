package billingservice

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/ref"
)

type linesFeatureGetter interface {
	GetReferencedFeatureKeys() ([]string, error)
}

func (s *Service) resolveFeatureMeters(ctx context.Context, namespace string, lines linesFeatureGetter) (feature.FeatureMeters, error) {
	keys, err := lines.GetReferencedFeatureKeys()
	if err != nil {
		return nil, fmt.Errorf("getting referenced feature keys: %w", err)
	}

	featureMeters, err := s.featureService.ResolveFeatureMeters(ctx, namespace, lo.Map(keys, func(key string, _ int) ref.IDOrKey {
		return ref.IDOrKey{Key: key}
	})...)
	if err != nil {
		return nil, fmt.Errorf("resolving feature meters: %w", err)
	}

	return featureMetersErrorWrapper{featureMeters}, nil
}

// featureMetersErrorWrapper is a wrapper around the feature meters that returns a ErrSnapshotInvalidDatabaseState if the feature meter is not found.
// This is useful to wrap the feature meters in a way that allows us to return a consistent error type for the billing service.
type featureMetersErrorWrapper struct {
	feature.FeatureMeters
}

func (w featureMetersErrorWrapper) Get(featureKey string, requireMeter bool) (feature.FeatureMeter, error) {
	featureMeter, err := w.FeatureMeters.Get(featureKey, requireMeter)
	if err != nil {
		return feature.FeatureMeter{}, &billing.ErrSnapshotInvalidDatabaseState{
			Err: err,
		}
	}

	return featureMeter, nil
}
