package productcatalogdriver

import (
	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/productcatalog"
	"github.com/openmeterio/openmeter/pkg/convert"
)

func MapFeatureToResponse(f productcatalog.Feature) api.Feature {
	return api.Feature{
		CreatedAt:           &f.CreatedAt,
		DeletedAt:           nil,
		UpdatedAt:           &f.UpdatedAt,
		Id:                  &f.ID,
		Key:                 f.Key,
		Metadata:            convert.MapToPointer(f.Metadata),
		Name:                f.Name,
		ArchivedAt:          f.ArchivedAt,
		MeterGroupByFilters: convert.MapToPointer(f.MeterGroupByFilters),
		MeterSlug:           f.MeterSlug,
	}
}
