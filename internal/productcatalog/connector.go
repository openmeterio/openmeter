package productcatalog

import (
	"context"
	"slices"

	"github.com/openmeterio/openmeter/internal/meter"
	"github.com/openmeterio/openmeter/pkg/models"
)

type FeatureConnector interface {
	// Feature Management
	CreateFeature(ctx context.Context, feature Feature) (Feature, error)
	// Should just use deletedAt, there's no real "archiving"
	ArchiveFeature(ctx context.Context, featureID NamespacedFeatureID) error
	ListFeatures(ctx context.Context, params ListFeaturesParams) ([]Feature, error)
	GetFeature(ctx context.Context, featureID NamespacedFeatureID) (Feature, error)
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

type FeatureDB interface {
	CreateFeature(ctx context.Context, feature Feature) (Feature, error)
	ArchiveFeature(ctx context.Context, featureID NamespacedFeatureID) error
	ListFeatures(ctx context.Context, params ListFeaturesParams) ([]Feature, error)
	FindByName(ctx context.Context, namespace string, name string, includeArchived bool) ([]Feature, error)
	GetByID(ctx context.Context, featureID NamespacedFeatureID) (Feature, error)
}

type featureConnector struct {
	db        FeatureDB
	meterRepo meter.Repository
}

func NewFeatureConnector(db FeatureDB) FeatureConnector {
	return &featureConnector{
		db: db,
	}
}

func (c *featureConnector) CreateFeature(ctx context.Context, feature Feature) (Feature, error) {
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

	err = c.checkGroupByFilters(feature, meter)
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

	return c.db.CreateFeature(ctx, feature)
}

func (c *featureConnector) ArchiveFeature(ctx context.Context, featureID NamespacedFeatureID) error {
	_, err := c.GetFeature(ctx, featureID)
	if err != nil {
		return err
	}
	return c.db.ArchiveFeature(ctx, featureID)
}

func (c *featureConnector) ListFeatures(ctx context.Context, params ListFeaturesParams) ([]Feature, error) {
	return c.db.ListFeatures(ctx, params)
}

func (c *featureConnector) GetFeature(ctx context.Context, featureID NamespacedFeatureID) (Feature, error) {
	feature, err := c.db.GetByID(ctx, featureID)
	if err != nil {
		return Feature{}, err
	}
	if feature.Namespace != featureID.Namespace {
		return Feature{}, &FeatureNotFoundError{ID: featureID.ID}
	}
	return feature, nil
}

func (c *featureConnector) checkGroupByFilters(feature Feature, meter models.Meter) error {
	if feature.MeterGroupByFilters == nil {
		return nil
	}

	for filterProp := range *feature.MeterGroupByFilters {
		if _, ok := meter.GroupBy[filterProp]; !ok {
			meterGroupByColumns := make([]string, 0, len(meter.GroupBy))
			for k := range meter.GroupBy {
				meterGroupByColumns = append(meterGroupByColumns, k)
			}
			return &FeatureInvalidFiltersError{
				RequestedFilters:    *feature.MeterGroupByFilters,
				MeterGroupByColumns: meterGroupByColumns,
			}
		}
	}

	return nil
}
