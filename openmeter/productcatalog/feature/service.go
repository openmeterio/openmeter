package feature

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

// Service is the main interface for feature management business logic.
type Service interface {
	// CreateFeature creates a new feature.
	CreateFeature(ctx context.Context, feature CreateFeatureInputs) (Feature, error)
	// ArchiveFeature archives (soft-deletes) a feature.
	ArchiveFeature(ctx context.Context, featureID models.NamespacedID) error
	// ListFeatures lists features with filtering and pagination.
	ListFeatures(ctx context.Context, params ListFeaturesParams) (pagination.Result[Feature], error)
	// GetFeature returns a feature by ID or key.
	GetFeature(ctx context.Context, namespace string, idOrKey string, includeArchived IncludeArchivedFeature) (*Feature, error)
	// ResolveFeatureMeters resolves the feature meters for a given namespace and feature keys, returning a map of feature key to feature meter.
	// The list contains either the active feature or the most recently archived feature.
	ResolveFeatureMeters(ctx context.Context, namespace string, featureKeys []string) (FeatureMeters, error)
}

// FeatureConnector is an alias for Service, kept for backward compatibility.
type FeatureConnector = Service

// CreateFeatureInputs contains the parameters for creating a feature.
type CreateFeatureInputs struct {
	Name                string              `json:"name"`
	Key                 string              `json:"key"`
	Namespace           string              `json:"namespace"`
	MeterSlug           *string             `json:"meterSlug"`
	MeterGroupByFilters MeterGroupByFilters `json:"meterGroupByFilters"`
	UnitCost            *UnitCost           `json:"unitCost"`
	Metadata            map[string]string   `json:"metadata"`
}

// IncludeArchivedFeature is a type for the include archived feature flag.
type IncludeArchivedFeature bool

const (
	IncludeArchivedFeatureTrue  IncludeArchivedFeature = true
	IncludeArchivedFeatureFalse IncludeArchivedFeature = false
)

// FeatureOrderBy is the order by clause for features.
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

// ListFeaturesParams contains the parameters for listing features.
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
