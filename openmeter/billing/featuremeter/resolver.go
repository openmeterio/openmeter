package featuremeter

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// Resolver resolves the feature and meter data needed by billing operations.
type Resolver interface {
	Resolve(ctx context.Context, namespace string, featureRefs []FeatureMeterRef, opts ...ResolveFeatureMetersOption) (FeatureMeters, error)
}

type FeatureService interface {
	ListFeatures(ctx context.Context, params feature.ListFeaturesParams) (pagination.Result[feature.Feature], error)
}

type MeterService interface {
	ListMeters(ctx context.Context, params meter.ListMetersParams) (pagination.Result[meter.Meter], error)
}

type Config struct {
	FeatureService FeatureService
	MeterService   MeterService
}

func (c Config) Validate() error {
	var errs []error

	if c.FeatureService == nil {
		errs = append(errs, errors.New("feature service is required"))
	}

	if c.MeterService == nil {
		errs = append(errs, errors.New("meter service is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func New(config Config) (Resolver, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &resolver{
		featureService: config.FeatureService,
		meterService:   config.MeterService,
	}, nil
}

type resolver struct {
	featureService FeatureService
	meterService   MeterService
}

var _ Resolver = (*resolver)(nil)

type ResolveFeatureMetersOptions struct {
	AllowMissingFeatures bool
}

type ResolveFeatureMetersOption func(*ResolveFeatureMetersOptions)

func NewResolveFeatureMetersOptions(opts ...ResolveFeatureMetersOption) ResolveFeatureMetersOptions {
	var options ResolveFeatureMetersOptions

	for _, opt := range opts {
		opt(&options)
	}

	return options
}

func WithAllowMissingFeatures() ResolveFeatureMetersOption {
	return func(options *ResolveFeatureMetersOptions) {
		options.AllowMissingFeatures = true
	}
}

func (r *resolver) Resolve(ctx context.Context, namespace string, featureRefs []FeatureMeterRef, opts ...ResolveFeatureMetersOption) (FeatureMeters, error) {
	if namespace == "" {
		return nil, errors.New("namespace is required")
	}

	options := NewResolveFeatureMetersOptions(opts...)

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

	features, err := r.featureService.ListFeatures(ctx, feature.ListFeaturesParams{
		IDsOrKeys:       featuresToResolve,
		Namespace:       namespace,
		IncludeArchived: true,
	})
	if err != nil {
		return nil, fmt.Errorf("listing features: %w", err)
	}

	resolved := resolveFeatureMeters(features.Items)

	metersToResolve := lo.Uniq(
		lo.Filter(
			lo.Map(lo.Values(resolved.ByID), func(featureMeter FeatureMeter, _ int) string {
				if featureMeter.Feature.MeterID == nil {
					return ""
				}

				return *featureMeter.Feature.MeterID
			}),
			func(meterID string, _ int) bool {
				return meterID != ""
			},
		),
	)

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

	if err := validateReferences(resolved, featureRefs, options); err != nil {
		return nil, err
	}

	return resolved, nil
}

func validateReferences(featureMeters FeatureMeters, featureRefs []FeatureMeterRef, options ResolveFeatureMetersOptions) error {
	// Missing features take precedence over other validation failures so a mixed
	// strict batch preserves the not-found error category at the API boundary.
	var notFoundErrs, validationErrs []error
	for _, featureRef := range featureRefs {
		if _, err := featureMeters.Resolve(featureRef); err != nil {
			if models.IsGenericNotFoundError(err) {
				if options.AllowMissingFeatures {
					continue
				}

				notFoundErrs = append(notFoundErrs, err)
			} else {
				validationErrs = append(validationErrs, err)
			}
		}
	}

	if err := errors.Join(notFoundErrs...); err != nil {
		return err
	}

	return errors.Join(validationErrs...)
}

func resolveFeatureMeters(features []feature.Feature) FeatureMeterCollection {
	featuresByKey := getLastFeatures(features)

	resolved := FeatureMeterCollection{
		ByKey: make(map[string]FeatureMeter, len(featuresByKey)),
		ByID:  make(map[string]FeatureMeter, len(features)),
	}

	for _, featureEntity := range features {
		resolved.ByID[featureEntity.ID] = FeatureMeter{
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
