package feature

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/meter"
)

type FeatureMeter struct {
	Feature Feature
	Meter   *meter.Meter
}

type FeatureMeters interface {
	Get(featureKey string, requireMeter bool) (FeatureMeter, error)
}

type FeatureMeterCollection map[string]FeatureMeter

func (f FeatureMeterCollection) Get(featureKey string, requireMeter bool) (FeatureMeter, error) {
	featureMeter, exists := f[featureKey]
	if !exists {
		return FeatureMeter{}, fmt.Errorf("feature[%s] not found", featureKey)
	}

	if requireMeter && featureMeter.Meter == nil {
		return FeatureMeter{}, fmt.Errorf("feature[%s] has no meter associated, but caller requires a meter", featureMeter.Feature.Key)
	}

	return featureMeter, nil
}

func (c *featureConnector) ResolveFeatureMeters(ctx context.Context, namespace string, featureKeys []string) (FeatureMeters, error) {
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}

	if len(featureKeys) == 0 {
		return FeatureMeterCollection{}, nil
	}

	featuresToResolve := lo.Uniq(lo.Filter(featureKeys, func(key string, _ int) bool {
		return key != ""
	}))

	// Let's resolve the features
	features, err := c.featureRepo.ListFeatures(ctx, ListFeaturesParams{
		IDsOrKeys:       featuresToResolve,
		Namespace:       namespace,
		IncludeArchived: true,
	})
	if err != nil {
		return nil, fmt.Errorf("listing features: %w", err)
	}

	featuresByKey := getLastFeatures(features.Items)

	metersToResolve := lo.Uniq(
		lo.Filter(
			lo.Map(lo.Values(featuresByKey), func(f Feature, _ int) string {
				if f.MeterSlug == nil {
					return ""
				}

				return *f.MeterSlug
			}),
			func(meterSlug string, _ int) bool {
				return meterSlug != ""
			},
		),
	)

	meters, err := c.meterService.ListMeters(ctx, meter.ListMetersParams{
		SlugFilter:     lo.ToPtr(metersToResolve),
		Namespace:      namespace,
		IncludeDeleted: true,
	})
	if err != nil {
		return nil, fmt.Errorf("listing meters: %w", err)
	}

	metersByKey := getLastMeters(meters.Items)

	out := make(FeatureMeterCollection, len(featuresByKey))
	for featureKey, feat := range featuresByKey {
		if feat.MeterSlug == nil {
			out[featureKey] = FeatureMeter{
				Feature: feat,
			}

			continue
		}

		meter, exists := metersByKey[*feat.MeterSlug]
		if !exists {
			out[featureKey] = FeatureMeter{
				Feature: feat,
			}

			continue
		}

		out[featureKey] = FeatureMeter{
			Feature: feat,
			Meter:   &meter,
		}
	}

	return out, nil
}

type lastEntityAccessor[T any] interface {
	GetKey(T) string
	GetDeletedAt(T) *time.Time
}

func getLastEntity[T any](entities []T, accessor lastEntityAccessor[T]) map[string]T {
	featuresByKey := lo.GroupBy(entities, func(entity T) string {
		return accessor.GetKey(entity)
	})

	out := make(map[string]T, len(featuresByKey))
	for key, features := range featuresByKey {
		// Let's try to find an unarchived feature
		out[key] = latestEntity(features, accessor)
	}

	return out
}

func latestEntity[T any](entities []T, accessor lastEntityAccessor[T]) T {
	for _, entity := range entities {
		if accessor.GetDeletedAt(entity) == nil {
			return entity
		}
	}

	// Otherwise, let's find the most recently archived feature:
	// - all entities have non-nil deleted at (or we would have returned already)
	// - and we have at least one entity due to the definition of the groupBy
	mostRecentlyArchivedFeature := entities[0]
	for _, entity := range entities {
		if accessor.GetDeletedAt(entity).After(*accessor.GetDeletedAt(mostRecentlyArchivedFeature)) {
			mostRecentlyArchivedFeature = entity
		}
	}

	return mostRecentlyArchivedFeature
}

type featureAccessor struct{}

var _ lastEntityAccessor[Feature] = (*featureAccessor)(nil)

func (a featureAccessor) GetKey(f Feature) string {
	return f.Key
}

func (a featureAccessor) GetDeletedAt(f Feature) *time.Time {
	return f.ArchivedAt
}

func getLastFeatures(features []Feature) map[string]Feature {
	return getLastEntity(features, featureAccessor{})
}

type meterAccessor struct{}

var _ lastEntityAccessor[meter.Meter] = (*meterAccessor)(nil)

func (a meterAccessor) GetKey(meter meter.Meter) string {
	return meter.Key
}

func (a meterAccessor) GetDeletedAt(meter meter.Meter) *time.Time {
	return meter.DeletedAt
}

func getLastMeters(meters []meter.Meter) map[string]meter.Meter {
	return getLastEntity(meters, meterAccessor{})
}
