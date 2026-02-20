package feature

import (
	"context"
	"fmt"
	"slices"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"

	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type CreateFeatureInputs struct {
	Name                string              `json:"name"`
	Key                 string              `json:"key"`
	Namespace           string              `json:"namespace"`
	MeterSlug           *string             `json:"meterSlug"`
	MeterGroupByFilters MeterGroupByFilters `json:"meterGroupByFilters"`
	Metadata            map[string]string   `json:"metadata"`
}

// TODO: refactor to service pattern
type FeatureConnector interface {
	// Feature Management
	CreateFeature(ctx context.Context, feature CreateFeatureInputs) (Feature, error)
	// Should just use deletedAt, there's no real "archiving"
	ArchiveFeature(ctx context.Context, featureID models.NamespacedID) error
	ListFeatures(ctx context.Context, params ListFeaturesParams) (pagination.Result[Feature], error)
	GetFeature(ctx context.Context, namespace string, idOrKey string, includeArchived IncludeArchivedFeature) (*Feature, error)

	// ResolveFeatureMeters resolves the feature meters for a given namespace and feature keys, returning a map of feature key to feature meter.
	// The list contains either the active feature or the most recently archived feature.
	ResolveFeatureMeters(ctx context.Context, namespace string, featureKeys []string) (FeatureMeters, error)
}

type IncludeArchivedFeature bool

const (
	IncludeArchivedFeatureTrue  IncludeArchivedFeature = true
	IncludeArchivedFeatureFalse IncludeArchivedFeature = false
)

// FeatureOrderBy is the order by clause for features
type FeatureOrderBy string

const (
	FeatureOrderByKey       FeatureOrderBy = "key"
	FeatureOrderByName      FeatureOrderBy = "name"
	FeatureOrderByCreatedAt FeatureOrderBy = "created_at"
	FeatureOrderByUpdatedAt FeatureOrderBy = "updated_at"
)

func (f FeatureOrderBy) Values() []FeatureOrderBy {
	return []FeatureOrderBy{
		FeatureOrderByKey,
		FeatureOrderByName,
		FeatureOrderByCreatedAt,
		FeatureOrderByUpdatedAt,
	}
}

type ListFeaturesParams struct {
	IDsOrKeys       []string
	Namespace       string
	MeterSlugs      []string
	IncludeArchived bool
	Page            pagination.Page
	OrderBy         FeatureOrderBy
	Order           sortx.Order
	// will be deprecated
	Limit int
	// will be deprecated
	Offset int
}

type featureConnector struct {
	featureRepo  FeatureRepo
	meterService meterpkg.Service
	publisher    eventbus.Publisher

	validMeterAggregations []meterpkg.MeterAggregation
}

func NewFeatureConnector(
	featureRepo FeatureRepo,
	meterService meterpkg.Service,
	publisher eventbus.Publisher,
) FeatureConnector {
	return &featureConnector{
		featureRepo:  featureRepo,
		meterService: meterService,
		publisher:    publisher,

		validMeterAggregations: []meterpkg.MeterAggregation{
			meterpkg.MeterAggregationSum,
			meterpkg.MeterAggregationCount,
			meterpkg.MeterAggregationUniqueCount,
			meterpkg.MeterAggregationLatest,
		},
	}
}

// CreateFeature creates a new feature
func (c *featureConnector) CreateFeature(ctx context.Context, feature CreateFeatureInputs) (Feature, error) {
	// Validate meter configuration
	if feature.MeterSlug != nil {
		slug := *feature.MeterSlug

		// nosemgrep: trailofbits.go.invalid-usage-of-modified-variable.invalid-usage-of-modified-variable
		meter, err := c.meterService.GetMeterByIDOrSlug(ctx, meterpkg.GetMeterInput{
			Namespace: feature.Namespace,
			IDOrSlug:  slug,
		})
		if err != nil {
			return Feature{}, meterpkg.NewMeterNotFoundError(slug)
		}

		if !slices.Contains(c.validMeterAggregations, meter.Aggregation) {
			return Feature{}, &FeatureInvalidMeterAggregationError{Aggregation: meter.Aggregation, MeterSlug: meter.Key, ValidAggregations: c.validMeterAggregations}
		}

		if feature.MeterGroupByFilters != nil {
			err = feature.MeterGroupByFilters.Validate(meter)
			if err != nil {
				return Feature{}, err
			}
		}
		if err != nil {
			return Feature{}, err
		}
	}

	// Validate feature key
	if _, err := ulid.Parse(feature.Key); err == nil {
		return Feature{}, models.NewGenericValidationError(fmt.Errorf("Feature key cannot be a valid ULID"))
	}

	// Check key is not taken
	found, err := c.featureRepo.GetByIdOrKey(ctx, feature.Namespace, feature.Key, false)
	if err != nil {
		if _, ok := err.(*FeatureNotFoundError); !ok {
			return Feature{}, err
		}
	} else {
		return Feature{}, &FeatureWithNameAlreadyExistsError{Name: feature.Key, ID: found.ID}
	}

	// Create the feature
	createdFeature, err := c.featureRepo.CreateFeature(ctx, feature)
	if err != nil {
		return Feature{}, err
	}

	// Publish the feature created event
	featureCreatedEvent := NewFeatureCreateEvent(ctx, &createdFeature)
	if err := c.publisher.Publish(ctx, featureCreatedEvent); err != nil {
		return createdFeature, fmt.Errorf("failed to publish feature created event: %w", err)
	}

	return createdFeature, nil
}

// ArchiveFeature archives a feature
func (c *featureConnector) ArchiveFeature(ctx context.Context, featureID models.NamespacedID) error {
	// Get the feature
	feat, err := c.GetFeature(ctx, featureID.Namespace, featureID.ID, false)
	if err != nil {
		return err
	}

	archivedAt := lo.ToPtr(clock.Now())

	// Archive the feature
	err = c.featureRepo.ArchiveFeature(ctx, ArchiveFeatureInput{
		Namespace: feat.Namespace,
		ID:        feat.ID,
		At:        archivedAt,
	})
	if err != nil {
		return err
	}

	feat.ArchivedAt = archivedAt

	// Publish the feature archived event
	featureArchivedEvent := NewFeatureArchiveEvent(ctx, feat)
	if err := c.publisher.Publish(ctx, featureArchivedEvent); err != nil {
		return fmt.Errorf("failed to publish feature archived event: %w", err)
	}

	return nil
}

// ListFeatures lists features
func (c *featureConnector) ListFeatures(ctx context.Context, params ListFeaturesParams) (pagination.Result[Feature], error) {
	if !params.Page.IsZero() {
		if err := params.Page.Validate(); err != nil {
			return pagination.Result[Feature]{}, err
		}
	}
	return c.featureRepo.ListFeatures(ctx, params)
}

// GetFeature gets a feature
func (c *featureConnector) GetFeature(ctx context.Context, namespace string, idOrKey string, includeArchived IncludeArchivedFeature) (*Feature, error) {
	feature, err := c.featureRepo.GetByIdOrKey(ctx, namespace, idOrKey, bool(includeArchived))
	if err != nil {
		return nil, err
	}
	return feature, nil
}
