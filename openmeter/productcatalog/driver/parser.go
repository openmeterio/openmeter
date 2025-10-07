package productcatalogdriver

import (
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/apiconverter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/filter"
)

func MapFeatureToResponse(f feature.Feature) api.Feature {
	meterGroupByFilters := feature.ConvertMeterGroupByFiltersToMapString(f.MeterGroupByFilters)

	return api.Feature{
		CreatedAt:                   f.CreatedAt,
		DeletedAt:                   nil,
		UpdatedAt:                   f.UpdatedAt,
		Id:                          f.ID,
		Key:                         f.Key,
		Metadata:                    convert.MapToPointer(f.Metadata),
		Name:                        f.Name,
		ArchivedAt:                  f.ArchivedAt,
		MeterGroupByFilters:         convert.MapToPointer(meterGroupByFilters),
		AdvancedMeterGroupByFilters: convert.MapToPointer(apiconverter.ConvertStringMapToAPIPtr(f.MeterGroupByFilters)),
		MeterSlug:                   f.MeterSlug,
	}
}

func MapFeatureCreateInputsRequest(namespace string, f api.FeatureCreateInputs) feature.CreateFeatureInputs {
	// if advancedMeterGroupByFilters is set, use it
	// otherwise, use legacy meterGroupByFilters
	meterGroupByFilters := lo.FromPtrOr(apiconverter.ConvertStringMapPtr(f.AdvancedMeterGroupByFilters), map[string]filter.FilterString{})
	if len(meterGroupByFilters) == 0 {
		meterGroupByFilters = feature.ConvertMapStringToMeterGroupByFilters(lo.FromPtrOr(f.MeterGroupByFilters, map[string]string{}))
	}

	return feature.CreateFeatureInputs{
		Namespace:           namespace,
		Name:                f.Name,
		Key:                 f.Key,
		MeterSlug:           f.MeterSlug,
		MeterGroupByFilters: meterGroupByFilters,
		Metadata:            convert.DerefHeaderPtr[string](f.Metadata),
	}
}
