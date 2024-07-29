package productcatalog

import (
	"context"
	"slices"

	"github.com/openmeterio/openmeter/internal/meter"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/slicesx"
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

type FeatureConnector interface {
	// Feature Management
	CreateFeature(ctx context.Context, feature CreateFeatureInputs) (Feature, error)
	// Should just use deletedAt, there's no real "archiving"
	ArchiveFeature(ctx context.Context, featureID models.NamespacedID) error
	ListFeatures(ctx context.Context, params ListFeaturesParams) (pagination.PagedResponse[Feature], error)
	GetFeature(ctx context.Context, namespace string, idOrKey string, includeArchived IncludeArchivedFeature) (*Feature, error)
}

type IncludeArchivedFeature bool

const (
	IncludeArchivedFeatureTrue  IncludeArchivedFeature = true
	IncludeArchivedFeatureFalse IncludeArchivedFeature = false
)

type FeatureOrderBy string

const (
	FeatureOrderByCreatedAt FeatureOrderBy = "created_at"
	FeatureOrderByUpdatedAt FeatureOrderBy = "updated_at"
)

func (f FeatureOrderBy) Values() []FeatureOrderBy {
	return []FeatureOrderBy{
		FeatureOrderByCreatedAt,
		FeatureOrderByUpdatedAt,
	}
}

func (f FeatureOrderBy) StrValues() []string {
	return slicesx.Map(f.Values(), func(v FeatureOrderBy) string {
		return string(v)
	})
}

type ListFeaturesParams struct {
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

type FeatureRepo interface {
	CreateFeature(ctx context.Context, feature CreateFeatureInputs) (Feature, error)
	ArchiveFeature(ctx context.Context, featureID models.NamespacedID) error
	ListFeatures(ctx context.Context, params ListFeaturesParams) (pagination.PagedResponse[Feature], error)
	HasActiveFeatureForMeter(ctx context.Context, namespace string, meterSlug string) (bool, error)

	GetByIdOrKey(ctx context.Context, namespace string, idOrKey string, includeArchived bool) (*Feature, error)
	entutils.TxCreator
	entutils.TxUser[FeatureRepo]
}

type featureConnector struct {
	featureRepo FeatureRepo
	meterRepo   meter.Repository

	validMeterAggregations []models.MeterAggregation
}

func NewFeatureConnector(
	featureRepo FeatureRepo,
	meterRepo meter.Repository,
) FeatureConnector {
	return &featureConnector{
		featureRepo: featureRepo,
		meterRepo:   meterRepo,

		validMeterAggregations: []models.MeterAggregation{
			models.MeterAggregationSum,
			models.MeterAggregationCount,
		},
	}
}

func (c *featureConnector) CreateFeature(ctx context.Context, feature CreateFeatureInputs) (Feature, error) {
	// validate meter configuration
	if feature.MeterSlug != nil {
		slug := *feature.MeterSlug
		meter, err := c.meterRepo.GetMeterByIDOrSlug(ctx, feature.Namespace, slug)
		if err != nil {
			return Feature{}, &models.MeterNotFoundError{MeterSlug: slug}
		}

		if !slices.Contains(c.validMeterAggregations, meter.Aggregation) {
			return Feature{}, &FeatureInvalidMeterAggregationError{Aggregation: meter.Aggregation, MeterSlug: meter.Slug, ValidAggregations: c.validMeterAggregations}
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

	// check key is not taken
	found, err := c.featureRepo.GetByIdOrKey(ctx, feature.Namespace, feature.Key, false)
	if err != nil {
		if _, ok := err.(*FeatureNotFoundError); !ok {
			return Feature{}, err
		}
	} else {
		return Feature{}, &FeatureWithNameAlreadyExistsError{Name: feature.Key, ID: found.ID}
	}

	return c.featureRepo.CreateFeature(ctx, feature)
}

func (c *featureConnector) ArchiveFeature(ctx context.Context, featureID models.NamespacedID) error {
	_, err := c.GetFeature(ctx, featureID.Namespace, featureID.ID, false)
	if err != nil {
		return err
	}
	return c.featureRepo.ArchiveFeature(ctx, featureID)
}

func (c *featureConnector) ListFeatures(ctx context.Context, params ListFeaturesParams) (pagination.PagedResponse[Feature], error) {
	return c.featureRepo.ListFeatures(ctx, params)
}

func (c *featureConnector) GetFeature(ctx context.Context, namespace string, idOrKey string, includeArchived IncludeArchivedFeature) (*Feature, error) {
	feature, err := c.featureRepo.GetByIdOrKey(ctx, namespace, idOrKey, bool(includeArchived))
	if err != nil {
		return nil, err
	}
	return feature, nil
}
