package feature

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type FeatureRepo interface {
	CreateFeature(ctx context.Context, feature CreateFeatureInputs) (Feature, error)
	ArchiveFeature(ctx context.Context, featureID models.NamespacedID) error
	ListFeatures(ctx context.Context, params ListFeaturesParams) (pagination.Result[Feature], error)
	HasActiveFeatureForMeter(ctx context.Context, namespace string, meterSlug string) (bool, error)

	GetByIdOrKey(ctx context.Context, namespace string, idOrKey string, includeArchived bool) (*Feature, error)
	entutils.TxCreator
	entutils.TxUser[FeatureRepo]
}
