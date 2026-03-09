package service

import (
	"context"
	"fmt"
	"slices"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"

	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var validMeterAggregations = []meterpkg.MeterAggregation{
	meterpkg.MeterAggregationSum,
	meterpkg.MeterAggregationCount,
	meterpkg.MeterAggregationUniqueCount,
	meterpkg.MeterAggregationLatest,
}

type service struct {
	adapter      feature.Adapter
	meterService meterpkg.Service
	publisher    eventbus.Publisher
}

// New creates a new feature service.
func New(adapter feature.Adapter, meterService meterpkg.Service, publisher eventbus.Publisher) feature.Service {
	return &service{
		adapter:      adapter,
		meterService: meterService,
		publisher:    publisher,
	}
}

// CreateFeature creates a new feature.
func (s *service) CreateFeature(ctx context.Context, feat feature.CreateFeatureInputs) (feature.Feature, error) {
	// Validate meter configuration
	var resolvedMeter *meterpkg.Meter

	if feat.MeterSlug != nil {
		slug := *feat.MeterSlug

		// nosemgrep: trailofbits.go.invalid-usage-of-modified-variable.invalid-usage-of-modified-variable
		meter, err := s.meterService.GetMeterByIDOrSlug(ctx, meterpkg.GetMeterInput{
			Namespace: feat.Namespace,
			IDOrSlug:  slug,
		})
		if err != nil {
			return feature.Feature{}, meterpkg.NewMeterNotFoundError(slug)
		}

		resolvedMeter = &meter

		if !slices.Contains(validMeterAggregations, meter.Aggregation) {
			return feature.Feature{}, &feature.FeatureInvalidMeterAggregationError{Aggregation: meter.Aggregation, MeterSlug: meter.Key, ValidAggregations: validMeterAggregations}
		}

		if feat.MeterGroupByFilters != nil {
			err = feat.MeterGroupByFilters.Validate(meter)
			if err != nil {
				return feature.Feature{}, err
			}
		}
		if err != nil {
			return feature.Feature{}, err
		}
	}

	// Validate unit cost
	if feat.UnitCost != nil {
		if err := feat.UnitCost.Validate(); err != nil {
			return feature.Feature{}, models.NewGenericValidationError(err)
		}

		if feat.UnitCost.Type == feature.UnitCostTypeLLM {
			if resolvedMeter == nil {
				return feature.Feature{}, models.NewGenericValidationError(
					fmt.Errorf("LLM unit cost requires a meter to be associated with the feature"),
				)
			}

			if err := feat.UnitCost.ValidateWithMeter(*resolvedMeter); err != nil {
				return feature.Feature{}, models.NewGenericValidationError(err)
			}
		}
	}

	// Validate feature key
	if _, err := ulid.Parse(feat.Key); err == nil {
		return feature.Feature{}, models.NewGenericValidationError(fmt.Errorf("feature key cannot be a valid ULID"))
	}

	// Check key is not taken
	found, err := s.adapter.GetByIdOrKey(ctx, feat.Namespace, feat.Key, false)
	if err != nil {
		if _, ok := err.(*feature.FeatureNotFoundError); !ok {
			return feature.Feature{}, err
		}
	} else {
		return feature.Feature{}, &feature.FeatureWithNameAlreadyExistsError{Name: feat.Key, ID: found.ID}
	}

	// Create the feature
	createdFeature, err := s.adapter.CreateFeature(ctx, feat)
	if err != nil {
		return feature.Feature{}, err
	}

	// Publish the feature created event
	featureCreatedEvent := feature.NewFeatureCreateEvent(ctx, &createdFeature)
	if err := s.publisher.Publish(ctx, featureCreatedEvent); err != nil {
		return createdFeature, fmt.Errorf("failed to publish feature created event: %w", err)
	}

	return createdFeature, nil
}

// ArchiveFeature archives a feature.
func (s *service) ArchiveFeature(ctx context.Context, featureID models.NamespacedID) error {
	// Get the feature
	feat, err := s.GetFeature(ctx, featureID.Namespace, featureID.ID, false)
	if err != nil {
		return err
	}

	archivedAt := lo.ToPtr(clock.Now())

	// Archive the feature
	err = s.adapter.ArchiveFeature(ctx, feature.ArchiveFeatureInput{
		Namespace: feat.Namespace,
		ID:        feat.ID,
		At:        archivedAt,
	})
	if err != nil {
		return err
	}

	feat.ArchivedAt = archivedAt

	// Publish the feature archived event
	featureArchivedEvent := feature.NewFeatureArchiveEvent(ctx, feat)
	if err := s.publisher.Publish(ctx, featureArchivedEvent); err != nil {
		return fmt.Errorf("failed to publish feature archived event: %w", err)
	}

	return nil
}

// ListFeatures lists features.
func (s *service) ListFeatures(ctx context.Context, params feature.ListFeaturesParams) (pagination.Result[feature.Feature], error) {
	if !params.Page.IsZero() {
		if err := params.Page.Validate(); err != nil {
			return pagination.Result[feature.Feature]{}, err
		}
	}
	return s.adapter.ListFeatures(ctx, params)
}

// GetFeature gets a feature.
func (s *service) GetFeature(ctx context.Context, namespace string, idOrKey string, includeArchived feature.IncludeArchivedFeature) (*feature.Feature, error) {
	feat, err := s.adapter.GetByIdOrKey(ctx, namespace, idOrKey, bool(includeArchived))
	if err != nil {
		return nil, err
	}
	return feat, nil
}

// ResolveFeatureMeters resolves the feature meters for a given namespace and feature keys.
func (s *service) ResolveFeatureMeters(ctx context.Context, namespace string, featureKeys []string) (feature.FeatureMeters, error) {
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}

	if len(featureKeys) == 0 {
		return feature.FeatureMeterCollection{}, nil
	}

	featuresToResolve := lo.Uniq(lo.Filter(featureKeys, func(key string, _ int) bool {
		return key != ""
	}))

	// Let's resolve the features
	features, err := s.adapter.ListFeatures(ctx, feature.ListFeaturesParams{
		IDsOrKeys:       featuresToResolve,
		Namespace:       namespace,
		IncludeArchived: true,
	})
	if err != nil {
		return nil, fmt.Errorf("listing features: %w", err)
	}

	featuresByKey := feature.GetLastFeatures(features.Items)

	metersToResolve := lo.Uniq(
		lo.Filter(
			lo.Map(lo.Values(featuresByKey), func(f feature.Feature, _ int) string {
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

	meters, err := s.meterService.ListMeters(ctx, meterpkg.ListMetersParams{
		SlugFilter:     lo.ToPtr(metersToResolve),
		Namespace:      namespace,
		IncludeDeleted: true,
	})
	if err != nil {
		return nil, fmt.Errorf("listing meters: %w", err)
	}

	metersByKey := feature.GetLastMeters(meters.Items)

	out := make(feature.FeatureMeterCollection, len(featuresByKey))
	for featureKey, feat := range featuresByKey {
		if feat.MeterSlug == nil {
			out[featureKey] = feature.FeatureMeter{
				Feature: feat,
			}

			continue
		}

		m, exists := metersByKey[*feat.MeterSlug]
		if !exists {
			out[featureKey] = feature.FeatureMeter{
				Feature: feat,
			}

			continue
		}

		out[featureKey] = feature.FeatureMeter{
			Feature: feat,
			Meter:   &m,
		}
	}

	return out, nil
}
