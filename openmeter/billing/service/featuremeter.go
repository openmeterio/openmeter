package billingservice

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
)

func (s *Service) resolveFeatureMeters(ctx context.Context, lines billing.StandardLines) (billing.FeatureMeters, error) {
	namespaces := lo.Uniq(lo.Map(lines, func(line *billing.StandardLine, _ int) string {
		return line.Namespace
	}))

	if len(namespaces) != 1 {
		return nil, fmt.Errorf("all lines must be in the same namespace")
	}

	namespace := namespaces[0]

	featuresToResolve := lo.Uniq(
		lo.Filter(
			lo.Map(lines, func(line *billing.StandardLine, _ int) string {
				return line.UsageBased.FeatureKey
			}),
			func(featureKey string, _ int) bool {
				return featureKey != ""
			},
		),
	)

	// Let's resolve the features
	features, err := s.featureService.ListFeatures(ctx, feature.ListFeaturesParams{
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
			lo.Map(lo.Values(featuresByKey), func(feature feature.Feature, _ int) string {
				if feature.MeterSlug == nil {
					return ""
				}

				return *feature.MeterSlug
			}),
			func(meterSlug string, _ int) bool {
				return meterSlug != ""
			},
		),
	)

	meters, err := s.meterService.ListMeters(ctx, meter.ListMetersParams{
		SlugFilter:     lo.ToPtr(metersToResolve),
		Namespace:      namespace,
		IncludeDeleted: true,
	})
	if err != nil {
		return nil, fmt.Errorf("listing meters: %w", err)
	}

	metersByKey := getLastMeters(meters.Items)

	out := make(billing.FeatureMeters, len(featuresByKey))
	for featureKey, feature := range featuresByKey {
		if feature.MeterSlug == nil {
			out[featureKey] = billing.FeatureMeter{
				Feature: feature,
			}

			continue
		}

		meter, exists := metersByKey[*feature.MeterSlug]
		if !exists {
			out[featureKey] = billing.FeatureMeter{
				Feature: feature,
			}

			continue
		}

		out[featureKey] = billing.FeatureMeter{
			Feature: feature,
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

var _ lastEntityAccessor[feature.Feature] = (*featureAccessor)(nil)

func (a featureAccessor) GetKey(feature feature.Feature) string {
	return feature.Key
}

func (a featureAccessor) GetDeletedAt(feature feature.Feature) *time.Time {
	return feature.ArchivedAt
}

func getLastFeatures(features []feature.Feature) map[string]feature.Feature {
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
