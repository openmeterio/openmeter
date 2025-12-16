//go:generate go run github.com/jmattheis/goverter/cmd/goverter gen ./
package meters

import (
	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/meter"
)

// goverter:variables
// goverter:skipCopySameType
// goverter:output:file ./convert.gen.go
// goverter:useZeroValueOnPointerInconsistency
// goverter:useUnderlyingTypeMethods
// goverter:matchIgnoreCase
// goverter:extend ConvertAPIMeterAggregationToMeterAggregation
// goverter:extend ConvertMeterAggregationToAPIMeterAggregation
// goverter:extend IntToFloat32
var (
	// goverter:context namespace
	// goverter:map Namespace | NamespaceFromContext
	// goverter:map Dimensions GroupBy
	// goverter:map Labels Metadata
	// goverter:map EventsFrom EventFrom
	// goverter:ignore Annotations
	// goverter:ignore inputOptions
	ConvertFromCreateMeterRequestToCreateMeterInput func(namespace string, createMeterRequest api.CreateMeterRequest) meter.CreateMeterInput
	// goverter:map Metadata Labels
	// goverter:map GroupBy Dimensions
	// goverter:map EventFrom EventsFrom
	// goverter:map ManagedResource.ID Id
	// goverter:map ManagedResource.Description Description
	// goverter:map ManagedResource.Name Name
	// goverter:map ManagedResource.ManagedModel.CreatedAt CreatedAt
	// goverter:map ManagedResource.ManagedModel.UpdatedAt UpdatedAt
	// goverter:map ManagedResource.ManagedModel.DeletedAt DeletedAt
	ConvertMeterToAPIMeter   func(meter.Meter) api.Meter
	ConvertMeterListResponse func(meters response.PagePaginationResponse[meter.Meter]) api.MeterPagePaginatedResponse
)

//goverter:context namespace
func NamespaceFromContext(namespace string) string {
	return namespace
}

func ConvertMeterAggregationToAPIMeterAggregation(aggregation meter.MeterAggregation) api.MeterAggregation {
	switch aggregation {
	case meter.MeterAggregationSum:
		return api.MeterAggregationSum
	case meter.MeterAggregationCount:
		return api.MeterAggregationCount
	case meter.MeterAggregationUniqueCount:
		return api.MeterAggregationUniqueCount
	case meter.MeterAggregationAvg:
		return api.MeterAggregationAvg
	case meter.MeterAggregationMin:
		return api.MeterAggregationMin
	case meter.MeterAggregationMax:
		return api.MeterAggregationMax
	case meter.MeterAggregationLatest:
		return api.MeterAggregationLatest
	}

	return api.MeterAggregation("")
}

func ConvertAPIMeterAggregationToMeterAggregation(aggregation api.MeterAggregation) meter.MeterAggregation {
	switch aggregation {
	case api.MeterAggregationSum:
		return meter.MeterAggregationSum
	case api.MeterAggregationCount:
		return meter.MeterAggregationCount
	case api.MeterAggregationUniqueCount:
		return meter.MeterAggregationUniqueCount
	case api.MeterAggregationAvg:
		return meter.MeterAggregationAvg
	case api.MeterAggregationMin:
		return meter.MeterAggregationMin
	case api.MeterAggregationMax:
		return meter.MeterAggregationMax
	case api.MeterAggregationLatest:
		return meter.MeterAggregationLatest
	}

	return ""
}

func IntToFloat32(i int) float32 {
	return float32(i)
}
