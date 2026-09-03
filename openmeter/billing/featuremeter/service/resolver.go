package featuremeterservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/samber/lo"

	billingfeaturemeter "github.com/openmeterio/openmeter/openmeter/billing/featuremeter"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/ref"
)

type FeatureService interface {
	ListFeatures(ctx context.Context, params feature.ListFeaturesParams) (pagination.Result[feature.Feature], error)
}

type MeterService interface {
	ListMeters(ctx context.Context, params meter.ListMetersParams) (pagination.Result[meter.Meter], error)
}

type Config struct {
	FeatureService FeatureService
	MeterService   MeterService
	Logger         *slog.Logger
}

func (c Config) Validate() error {
	var errs []error

	if c.FeatureService == nil {
		errs = append(errs, errors.New("feature service is required"))
	}

	if c.MeterService == nil {
		errs = append(errs, errors.New("meter service is required"))
	}

	if c.Logger == nil {
		errs = append(errs, errors.New("logger is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func New(config Config) (*Resolver, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Resolver{
		featureService: config.FeatureService,
		meterService:   config.MeterService,
		logger:         config.Logger,
	}, nil
}

// Resolver resolves the feature and meter data needed by billing operations.
type Resolver struct {
	featureService FeatureService
	meterService   MeterService
	logger         *slog.Logger
}

// Resolve extracts and resolves the feature and meter dependencies of billing entities.
//
// When target validation fails, Resolve returns any feature-meter data it could
// resolve alongside the error. Callers that can continue with partial data must
// split the error with billing.ToValidationIssues and may continue only when the
// returned system error is nil.
func (r Resolver) Resolve[T billingfeaturemeter.FeatureReferenceGetter](ctx context.Context, namespace string, targets ...T) (billingfeaturemeter.FeatureMeters, error) {
	if namespace == "" {
		return nil, models.NewGenericValidationError(errors.New("namespace is required"))
	}

	featureRefs := collectUniqueFeatureMeterRefs(targets)

	if len(featureRefs) == 0 {
		return FeatureMeterCollection{
			ByKey: map[string]billingfeaturemeter.FeatureMeter{},
			ByID:  map[string]billingfeaturemeter.FeatureMeter{},
		}, nil
	}

	featuresToResolve := lo.Uniq(lo.FlatMap(featureRefs, func(featureRef billingfeaturemeter.FeatureMeterRef, _ int) []string {
		out := featureRef.IDOrKey.GetKeys()
		out = append(out, featureRef.IDOrKey.GetIDs()...)

		return out
	}))

	features, err := r.featureService.ListFeatures(ctx, feature.ListFeaturesParams{
		IDsOrKeys:       featuresToResolve,
		Namespace:       namespace,
		IncludeArchived: true,
	})
	if err != nil {
		return nil, fmt.Errorf("listing features: %w", err)
	}

	resolved := resolveFeatureMeters(features.Items)

	metersToResolve := lo.Uniq(lo.FilterMap(lo.Values(resolved.ByID), func(featureMeter billingfeaturemeter.FeatureMeter, _ int) (string, bool) {
		if featureMeter.Feature.MeterID == nil {
			return "", false
		}

		return *featureMeter.Feature.MeterID, true
	}))

	meters, err := r.meterService.ListMeters(ctx, meter.ListMetersParams{
		IDFilter:       lo.ToPtr(metersToResolve),
		Namespace:      namespace,
		IncludeDeleted: true,
	})
	if err != nil {
		return nil, fmt.Errorf("listing meters: %w", err)
	}

	metersByID := lo.SliceToMap(meters.Items, func(meterEntity meter.Meter) (string, meter.Meter) {
		return meterEntity.ID, meterEntity
	})

	for featureID, featureMeter := range resolved.ByID {
		if featureMeter.Feature.MeterID == nil {
			continue
		}

		meterEntity, exists := metersByID[*featureMeter.Feature.MeterID]
		if exists {
			featureMeter.Meter = &meterEntity
		}

		resolved.ByID[featureID] = featureMeter
		if latest, ok := resolved.ByKey[featureMeter.Feature.Key]; ok && latest.Feature.ID == featureID {
			resolved.ByKey[featureMeter.Feature.Key] = featureMeter
		}
	}

	if err := validateReferences(resolved, targets); err != nil {
		return resolved, err
	}

	return resolved, nil
}

func collectUniqueFeatureMeterRefs[T billingfeaturemeter.FeatureReferenceGetter](references []T) []billingfeaturemeter.FeatureMeterRef {
	featureRefs := lo.FilterMap(references, func(reference T, _ int) (billingfeaturemeter.FeatureMeterRef, bool) {
		featureRef := reference.GetFeatureMeterRef()
		if featureRef == nil {
			return billingfeaturemeter.FeatureMeterRef{}, false
		}

		return *featureRef, true
	})

	featureRefs = lo.MapToSlice(
		lo.GroupBy(featureRefs, func(featureRef billingfeaturemeter.FeatureMeterRef) ref.IDOrKey {
			return featureRef.IDOrKey
		}),
		func(idOrKey ref.IDOrKey, references []billingfeaturemeter.FeatureMeterRef) billingfeaturemeter.FeatureMeterRef {
			return billingfeaturemeter.FeatureMeterRef{
				IDOrKey: idOrKey,
				RequireMeter: lo.SomeBy(references, func(reference billingfeaturemeter.FeatureMeterRef) bool {
					return reference.RequireMeter
				}),
			}
		},
	)

	return featureRefs
}

func validateReferences[T billingfeaturemeter.FeatureReferenceGetter](featureMeters billingfeaturemeter.FeatureMeters, targets []T) error {
	errs := lo.FilterMap(targets, func(target T, _ int) (error, bool) {
		if target.GetFeatureMeterRef() == nil {
			return nil, false
		}

		_, err := featureMeters.Get(target)
		return err, err != nil
	})

	return errors.Join(errs...)
}

func resolveFeatureMeters(features []feature.Feature) FeatureMeterCollection {
	featuresByKey := getLastFeatures(features)

	resolved := FeatureMeterCollection{
		ByKey: make(map[string]billingfeaturemeter.FeatureMeter, len(featuresByKey)),
		ByID:  make(map[string]billingfeaturemeter.FeatureMeter, len(features)),
	}

	for _, featureEntity := range features {
		resolved.ByID[featureEntity.ID] = billingfeaturemeter.FeatureMeter{
			Feature: featureEntity,
		}
	}

	for featureKey, featureEntity := range featuresByKey {
		resolved.ByKey[featureKey] = resolved.ByID[featureEntity.ID]
	}

	return resolved
}

type lastEntityAccessor[T any] interface {
	GetKey(T) string
	GetDeletedAt(T) *time.Time
}

func getLastEntity[T any](entities []T, accessor lastEntityAccessor[T]) map[string]T {
	entitiesByKey := lo.GroupBy(entities, func(entity T) string {
		return accessor.GetKey(entity)
	})

	latestByKey := make(map[string]T, len(entitiesByKey))
	for key, entities := range entitiesByKey {
		latestByKey[key] = latestEntity(entities, accessor)
	}

	return latestByKey
}

func latestEntity[T any](entities []T, accessor lastEntityAccessor[T]) T {
	for _, entity := range entities {
		if accessor.GetDeletedAt(entity) == nil {
			return entity
		}
	}

	mostRecentlyArchived := entities[0]
	for _, entity := range entities {
		if accessor.GetDeletedAt(entity).After(*accessor.GetDeletedAt(mostRecentlyArchived)) {
			mostRecentlyArchived = entity
		}
	}

	return mostRecentlyArchived
}

type featureAccessor struct{}

var _ lastEntityAccessor[feature.Feature] = (*featureAccessor)(nil)

func (featureAccessor) GetKey(featureEntity feature.Feature) string {
	return featureEntity.Key
}

func (featureAccessor) GetDeletedAt(featureEntity feature.Feature) *time.Time {
	return featureEntity.ArchivedAt
}

func getLastFeatures(features []feature.Feature) map[string]feature.Feature {
	return getLastEntity(features, featureAccessor{})
}
