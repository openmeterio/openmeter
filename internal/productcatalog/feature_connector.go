package productcatalog

import (
	"context"
	"slices"

	"github.com/openmeterio/openmeter/internal/meter"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

type CreateFeatureInputs struct {
	Name                string            `json:"name"`
	Key                 string            `json:"key"`
	Namespace           string            `json:"namespace"`
	MeterSlug           string            `json:"meterSlug"`
	MeterGroupByFilters map[string]string `json:"meterGroupByFilters"`
}

type FeatureConnector interface {
	// Feature Management
	CreateFeature(ctx context.Context, feature CreateFeatureInputs) (Feature, error)
	// Should just use deletedAt, there's no real "archiving"
	ArchiveFeature(ctx context.Context, featureID models.NamespacedID) error
	ListFeatures(ctx context.Context, params ListFeaturesParams) ([]Feature, error)
	GetFeature(ctx context.Context, featureID models.NamespacedID) (Feature, error)
}

type FeatureOrderBy string

const (
	FeatureOrderByCreatedAt FeatureOrderBy = "created_at"
	FeatureOrderByUpdatedAt FeatureOrderBy = "updated_at"
)

type ListFeaturesParams struct {
	Namespace       string
	IncludeArchived bool
	Offset          int
	Limit           int
	OrderBy         FeatureOrderBy
}

type FeatureRepoCreateFeatureInputs struct {
	Name                string
	Key                 string
	Namespace           string
	MeterSlug           string
	MeterGroupByFilters map[string]string
}

type FeatureRepo interface {
	CreateFeature(ctx context.Context, feature FeatureRepoCreateFeatureInputs) (Feature, error)
	ArchiveFeature(ctx context.Context, featureID models.NamespacedID) error
	ListFeatures(ctx context.Context, params ListFeaturesParams) ([]Feature, error)
	FindByKey(ctx context.Context, namespace string, key string, includeArchived bool) (*Feature, error)
	GetByID(ctx context.Context, featureID models.NamespacedID) (Feature, error)

	entutils.TxCreator
	entutils.TxUser[FeatureRepo]
}

type featureConnector struct {
	featureRepo FeatureRepo
	meterRepo   meter.Repository
}

func NewFeatureConnector(
	featureRepo FeatureRepo,
	meterRepo meter.Repository,
) FeatureConnector {
	return &featureConnector{
		featureRepo: featureRepo,
		meterRepo:   meterRepo,
	}
}

func (c *featureConnector) CreateFeature(ctx context.Context, feature CreateFeatureInputs) (Feature, error) {
	meter, err := c.meterRepo.GetMeterByIDOrSlug(ctx, feature.Namespace, feature.MeterSlug)
	if err != nil {
		return Feature{}, &models.MeterNotFoundError{MeterSlug: feature.MeterSlug}
	}

	validAggregations := []models.MeterAggregation{
		models.MeterAggregationSum,
		models.MeterAggregationCount,
	}
	if !slices.Contains(validAggregations, meter.Aggregation) {
		return Feature{}, &FeatureInvalidMeterAggregationError{Aggregation: meter.Aggregation, MeterSlug: meter.Slug, ValidAggregations: validAggregations}
	}

	err = c.checkGroupByFilters(feature.MeterGroupByFilters, meter)
	if err != nil {
		return Feature{}, err
	}

	found, err := c.featureRepo.FindByKey(ctx, feature.Namespace, feature.Name, false)
	if err != nil {
		if _, ok := err.(*FeatureNotFoundError); !ok {
			return Feature{}, err
		}
	} else {
		return Feature{}, &FeatureWithNameAlreadyExistsError{Name: feature.Name, ID: found.ID}
	}

	return c.featureRepo.CreateFeature(ctx, FeatureRepoCreateFeatureInputs(feature))
}

func (c *featureConnector) ArchiveFeature(ctx context.Context, featureID models.NamespacedID) error {
	_, err := c.GetFeature(ctx, featureID)
	if err != nil {
		return err
	}
	return c.featureRepo.ArchiveFeature(ctx, featureID)
}

func (c *featureConnector) ListFeatures(ctx context.Context, params ListFeaturesParams) ([]Feature, error) {
	return c.featureRepo.ListFeatures(ctx, params)
}

func (c *featureConnector) GetFeature(ctx context.Context, featureID models.NamespacedID) (Feature, error) {
	feature, err := c.featureRepo.GetByID(ctx, featureID)
	if err != nil {
		return Feature{}, err
	}
	if feature.Namespace != featureID.Namespace {
		return Feature{}, &FeatureNotFoundError{ID: featureID.ID}
	}
	return feature, nil
}

func (c *featureConnector) checkGroupByFilters(filters map[string]string, meter models.Meter) error {
	if filters == nil {
		return nil
	}

	for filterProp := range filters {
		if _, ok := meter.GroupBy[filterProp]; !ok {
			meterGroupByColumns := make([]string, 0, len(meter.GroupBy))
			for k := range meter.GroupBy {
				meterGroupByColumns = append(meterGroupByColumns, k)
			}
			return &FeatureInvalidFiltersError{
				RequestedFilters:    filters,
				MeterGroupByColumns: meterGroupByColumns,
			}
		}
	}

	return nil
}
