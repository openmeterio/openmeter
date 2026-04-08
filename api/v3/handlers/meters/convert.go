//go:generate go run github.com/jmattheis/goverter/cmd/goverter gen ./
package meters

import (
	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/handlers/meters/query"
	"github.com/openmeterio/openmeter/api/v3/labels"
	"github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/models"
)

// goverter:variables
// goverter:skipCopySameType
// goverter:output:file ./convert.gen.go
// goverter:useZeroValueOnPointerInconsistency
// goverter:useUnderlyingTypeMethods
// goverter:matchIgnoreCase
// goverter:extend ConvertAPIMeterAggregationToMeterAggregation
// goverter:extend ConvertMeterAggregationToAPIMeterAggregation
// goverter:extend ConvertMetadataAnnotationsToLabels
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
	// goverter:map GroupBy Dimensions
	// goverter:map EventFrom EventsFrom
	// goverter:map ManagedResource.ID Id
	// goverter:map ManagedResource.Description Description
	// goverter:map ManagedResource.Name Name
	// goverter:map ManagedResource.ManagedModel.CreatedAt CreatedAt
	// goverter:map ManagedResource.ManagedModel.UpdatedAt UpdatedAt
	// goverter:map ManagedResource.ManagedModel.DeletedAt DeletedAt
	// goverter:map . Labels | ConvertMetadataAnnotationsToLabels
	ConvertMeterToAPIMeter   func(meter.Meter) api.Meter
	ConvertMeterListResponse func(meters response.PagePaginationResponse[meter.Meter]) api.MeterPagePaginatedResponse
)

func ConvertMetadataAnnotationsToLabels(source meter.Meter) *api.Labels {
	return labels.FromMetadataAnnotations(source.Metadata, source.Annotations)
}

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

// ConvertMetadataToLabels converts models.Metadata to api.Labels.
// Always returns an initialized map (never nil) so JSON serializes to {} instead of null.
func ConvertMetadataToLabels(source models.Metadata) *api.Labels {
	labels := make(api.Labels)
	for k, v := range source {
		labels[k] = v
	}
	return &labels
}

func ConvertMeterQueryRowToAPI(row meter.MeterQueryRow) api.MeterQueryRow {
	dimensions := make(map[string]string)

	if row.Subject != nil {
		dimensions[query.DimensionSubject] = *row.Subject
	}

	if row.CustomerID != nil {
		dimensions[query.DimensionCustomerID] = *row.CustomerID
	}

	for key, value := range row.GroupBy {
		if key == query.DimensionSubject || key == query.DimensionCustomerID {
			continue
		}
		if value != nil {
			dimensions[key] = *value
		}
	}

	return api.MeterQueryRow{
		Value:      alpacadecimal.NewFromFloat(row.Value).String(),
		From:       row.WindowStart,
		To:         row.WindowEnd,
		Dimensions: dimensions,
	}
}

func ConvertMeterQueryResultToAPI(from *api.DateTime, to *api.DateTime, rows []meter.MeterQueryRow) api.MeterQueryResult {
	return api.MeterQueryResult{
		From: from,
		To:   to,
		Data: lo.Map(rows, func(row meter.MeterQueryRow, _ int) api.MeterQueryRow {
			return ConvertMeterQueryRowToAPI(row)
		}),
	}
}
