package feature

import (
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

// GetLastFeatures returns a map of feature key to the most recent feature (active preferred, then most recently archived).
func GetLastFeatures(features []Feature) map[string]Feature {
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

// GetLastMeters returns a map of meter key to the most recent meter (active preferred, then most recently deleted).
func GetLastMeters(meters []meter.Meter) map[string]meter.Meter {
	return getLastEntity(meters, meterAccessor{})
}
