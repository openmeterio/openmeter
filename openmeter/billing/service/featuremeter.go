package billingservice

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	billingfeaturemeter "github.com/openmeterio/openmeter/openmeter/billing/featuremeter"
	"github.com/openmeterio/openmeter/pkg/ref"
)

type linesFeatureGetter interface {
	GetReferencedFeatureKeys() ([]string, error)
}

func (s *Service) resolveFeatureMeters(ctx context.Context, namespace string, lines linesFeatureGetter) (billingfeaturemeter.FeatureMeters, error) {
	keys, err := lines.GetReferencedFeatureKeys()
	if err != nil {
		return nil, fmt.Errorf("getting referenced feature keys: %w", err)
	}

	featureMeters, err := s.featureMeterResolver.Resolve(ctx, namespace, lo.Map(keys, func(key string, _ int) billingfeaturemeter.FeatureMeterRef {
		return billingfeaturemeter.FeatureMeterRef{
			IDOrKey: ref.IDOrKey{Key: key},
		}
	}))
	if err != nil {
		return nil, fmt.Errorf("resolving feature meters: %w", err)
	}

	return featureMetersErrorWrapper{featureMeters}, nil
}

// featureMetersErrorWrapper identifies a persisted feature whose required
// meter association is missing while preserving unrelated resolver errors.
type featureMetersErrorWrapper struct {
	billingfeaturemeter.FeatureMeters
}

func (w featureMetersErrorWrapper) GetByKey(featureKey string, requireMeter bool) (billingfeaturemeter.FeatureMeter, error) {
	featureMeter, err := w.FeatureMeters.GetByKey(featureKey, false)
	if err != nil {
		return billingfeaturemeter.FeatureMeter{}, err
	}

	if requireMeter && featureMeter.Meter == nil {
		return billingfeaturemeter.FeatureMeter{}, &billing.ErrSnapshotFeatureHasNoMeter{
			FeatureKey: featureKey,
		}
	}

	return featureMeter, nil
}
