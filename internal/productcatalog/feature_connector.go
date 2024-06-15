package productcatalog

import (
	"context"
	"slices"

	"github.com/openmeterio/openmeter/internal/meter"
	"github.com/openmeterio/openmeter/pkg/models"
)

type CreateFeatureInputs struct {
	Name                string
	Namespace           string
	MeterSlug           string
	MeterGroupByFilters *map[string]string
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

type DBCreateFeatureInputs struct {
	Name                string
	Namespace           string
	MeterSlug           string
	MeterGroupByFilters *map[string]string
}

type FeatureDBConnector interface {
	CreateFeature(ctx context.Context, feature DBCreateFeatureInputs) (Feature, error)
	ArchiveFeature(ctx context.Context, featureID models.NamespacedID) error
	ListFeatures(ctx context.Context, params ListFeaturesParams) ([]Feature, error)
	FindByName(ctx context.Context, namespace string, name string, includeArchived bool) ([]Feature, error)
	GetByID(ctx context.Context, featureID models.NamespacedID) (Feature, error)
}

type featureConnector struct {
	db        FeatureDBConnector
	meterRepo meter.Repository
}

func NewFeatureConnector(db FeatureDBConnector) FeatureConnector {
	return &featureConnector{
		db: db,
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

	nameMatches, err := c.db.FindByName(ctx, feature.Namespace, feature.Name, false)
	if err != nil {
		return Feature{}, err
	}

	if len(nameMatches) > 0 {
		return Feature{}, &FeatureWithNameAlreadyExistsError{Name: feature.Name, ID: nameMatches[0].ID}
	}

	//nolint:staticcheck Interface might change
	return c.db.CreateFeature(ctx, DBCreateFeatureInputs{
		Name:                feature.Name,
		Namespace:           feature.Namespace,
		MeterSlug:           feature.MeterSlug,
		MeterGroupByFilters: feature.MeterGroupByFilters,
	})
}

func (c *featureConnector) ArchiveFeature(ctx context.Context, featureID models.NamespacedID) error {
	_, err := c.GetFeature(ctx, featureID)
	if err != nil {
		return err
	}
	return c.db.ArchiveFeature(ctx, featureID)
}

func (c *featureConnector) ListFeatures(ctx context.Context, params ListFeaturesParams) ([]Feature, error) {
	return c.db.ListFeatures(ctx, params)
}

func (c *featureConnector) GetFeature(ctx context.Context, featureID models.NamespacedID) (Feature, error) {
	feature, err := c.db.GetByID(ctx, featureID)
	if err != nil {
		return Feature{}, err
	}
	if feature.Namespace != featureID.Namespace {
		return Feature{}, &FeatureNotFoundError{ID: featureID.ID}
	}
	return feature, nil
}

func (c *featureConnector) checkGroupByFilters(filters *map[string]string, meter models.Meter) error {
	if filters == nil {
		return nil
	}

	for filterProp := range *filters {
		if _, ok := meter.GroupBy[filterProp]; !ok {
			meterGroupByColumns := make([]string, 0, len(meter.GroupBy))
			for k := range meter.GroupBy {
				meterGroupByColumns = append(meterGroupByColumns, k)
			}
			return &FeatureInvalidFiltersError{
				RequestedFilters:    *filters,
				MeterGroupByColumns: meterGroupByColumns,
			}
		}
	}

	return nil
}
