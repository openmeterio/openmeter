package feature

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/ref"
)

type FeatureMeter struct {
	Feature Feature
	Meter   *meter.Meter
}

type FeatureMeters interface {
	Get(featureKey string, requireMeter bool) (FeatureMeter, error)
	GetByID(featureID string, requireMeter bool) (FeatureMeter, error)
	Resolve(ref FeatureMeterRef) (FeatureMeter, error)
}

type FeatureMeterCollection struct {
	ByKey map[string]FeatureMeter
	ByID  map[string]FeatureMeter
}

func (f FeatureMeterCollection) Get(featureKey string, requireMeter bool) (FeatureMeter, error) {
	featureMeter, exists := f.ByKey[featureKey]
	if !exists {
		return FeatureMeter{}, models.NewGenericNotFoundError(fmt.Errorf("feature[%s] not found", featureKey))
	}

	if requireMeter && featureMeter.Meter == nil {
		return FeatureMeter{}, models.NewGenericValidationError(fmt.Errorf("feature[%s] has no meter associated", featureMeter.Feature.Key))
	}

	return featureMeter, nil
}

func (f FeatureMeterCollection) GetByID(featureID string, requireMeter bool) (FeatureMeter, error) {
	featureMeter, exists := f.ByID[featureID]
	if !exists {
		return FeatureMeter{}, models.NewGenericNotFoundError(fmt.Errorf("feature[%s] not found", featureID))
	}

	if requireMeter && featureMeter.Meter == nil {
		return FeatureMeter{}, models.NewGenericValidationError(fmt.Errorf("feature[%s] has no meter associated", featureMeter.Feature.Key))
	}

	return featureMeter, nil
}

type FeatureMeterRef struct {
	IDOrKey      ref.IDOrKey
	RequireMeter bool
}

func (f FeatureMeterCollection) Resolve(r FeatureMeterRef) (FeatureMeter, error) {
	var featureMeter FeatureMeter
	var err error

	switch {
	case r.IDOrKey.Key != "" && r.IDOrKey.ID != "":
		return FeatureMeter{}, fmt.Errorf("feature reference must have either key or ID, not both")
	case r.IDOrKey.Key != "":
		featureMeter, err = f.Get(r.IDOrKey.Key, r.RequireMeter)
	case r.IDOrKey.ID != "":
		featureMeter, err = f.GetByID(r.IDOrKey.ID, r.RequireMeter)
	default:
		return FeatureMeter{}, fmt.Errorf("feature reference must have either key or ID")
	}

	return featureMeter, err
}

func (c *featureConnector) ResolveFeatureMeters(ctx context.Context, namespace string, featureRefs ...FeatureMeterRef) (FeatureMeters, error) {
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}

	if len(featureRefs) == 0 {
		return FeatureMeterCollection{
			ByKey: map[string]FeatureMeter{},
			ByID:  map[string]FeatureMeter{},
		}, nil
	}

	featuresToResolve := lo.Uniq(lo.FlatMap(featureRefs, func(featureRef FeatureMeterRef, _ int) []string {
		out := featureRef.IDOrKey.GetKeys()
		out = append(out, featureRef.IDOrKey.GetIDs()...)
		return out
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

	out := resolveFeatureMeters(features.Items)

	metersToResolve := lo.Uniq(
		lo.Filter(
			lo.Map(lo.Values(out.ByID), func(fm FeatureMeter, _ int) string {
				f := fm.Feature
				if f.MeterID == nil {
					return ""
				}

				return *f.MeterID
			}),
			func(meterID string, _ int) bool {
				return meterID != ""
			},
		),
	)

	meters, err := c.meterService.ListMeters(ctx, meter.ListMetersParams{
		IDFilter:       lo.ToPtr(metersToResolve),
		Namespace:      namespace,
		IncludeDeleted: true,
	})
	if err != nil {
		return nil, fmt.Errorf("listing meters: %w", err)
	}

	metersByID := lo.SliceToMap(meters.Items, func(m meter.Meter) (string, meter.Meter) {
		return m.ID, m
	})

	for featureID, featureMeter := range out.ByID {
		if featureMeter.Feature.MeterID == nil {
			out.ByID[featureID] = featureMeter
			continue
		}

		meter, exists := metersByID[*featureMeter.Feature.MeterID]
		if exists {
			featureMeter.Meter = &meter
		}

		out.ByID[featureID] = featureMeter
		if latest, ok := out.ByKey[featureMeter.Feature.Key]; ok && latest.Feature.ID == featureID {
			out.ByKey[featureMeter.Feature.Key] = featureMeter
		}
	}

	// Let's make sure that all the feature refs are available in the output.
	featureRefErrs := errors.Join(lo.Map(featureRefs, func(featureRef FeatureMeterRef, _ int) error {
		if _, err := out.Resolve(featureRef); err != nil {
			return err
		}
		return nil
	})...)
	if featureRefErrs != nil {
		return nil, featureRefErrs
	}

	return out, nil
}

func resolveFeatureMeters(features []Feature) FeatureMeterCollection {
	featuresByKey := getLastFeatures(features)

	out := FeatureMeterCollection{
		ByKey: make(map[string]FeatureMeter, len(featuresByKey)),
		ByID:  make(map[string]FeatureMeter, len(features)),
	}

	for _, feat := range features {
		out.ByID[feat.ID] = FeatureMeter{
			Feature: feat,
		}
	}

	for featureKey, feat := range featuresByKey {
		out.ByKey[featureKey] = out.ByID[feat.ID]
	}

	return out
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
