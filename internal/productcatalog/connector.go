package productcatalog

import "context"

type Connector interface {
	// Feature Management
	CreateFeature(ctx context.Context, feature Feature) (Feature, error)
	DeleteFeature(ctx context.Context, featureID NamespacedFeatureID) error
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
