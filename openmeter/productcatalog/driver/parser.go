package productcatalogdriver

import (
	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/convert"
)

func MapFeatureToResponse(f feature.Feature) api.Feature {
	meterGroupByFilters := feature.ConvertMeterGroupByFiltersToMapString(f.MeterGroupByFilters)

	return api.Feature{
		CreatedAt:           f.CreatedAt,
		DeletedAt:           nil,
		UpdatedAt:           f.UpdatedAt,
		Id:                  f.ID,
		Key:                 f.Key,
		Metadata:            convert.MapToPointer(f.Metadata),
		Name:                f.Name,
		ArchivedAt:          f.ArchivedAt,
		MeterGroupByFilters: convert.MapToPointer(meterGroupByFilters),
		MeterSlug:           f.MeterSlug,
	}
}
