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

	featureMeters, err := s.featureService.ResolveFeatureMeters(ctx, namespace, lo.Map(keys, func(key string, _ int) feature.FeatureMeterRef {
		return feature.FeatureMeterRef{
			IDOrKey: ref.IDOrKey{Key: key},
		}
	})...)
	if err != nil {
		return nil, fmt.Errorf("resolving feature meters: %w", err)
	}

	return featureMetersErrorWrapper{featureMeters}, nil
}

// featureMetersErrorWrapper identifies a persisted feature whose required
// meter association is missing while preserving unrelated resolver errors.
type featureMetersErrorWrapper struct {
	feature.FeatureMeters
}

func (w featureMetersErrorWrapper) Get(featureKey string, requireMeter bool) (feature.FeatureMeter, error) {
	featureMeter, err := w.FeatureMeters.Get(featureKey, false)
	if err != nil {
		return feature.FeatureMeter{}, err
	}

	if requireMeter && featureMeter.Meter == nil {
		return feature.FeatureMeter{}, &billing.ErrSnapshotFeatureHasNoMeter{
			FeatureKey: featureKey,
		}
	}

	return featureMeter, nil
}
