//go:generate go run github.com/jmattheis/goverter/cmd/goverter gen ./
package meters

import (
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
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
// goverter:extend ConvertMetadataToLabels
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

// ConvertMetadataToLabels converts models.Metadata to api.Labels.
// Always returns an initialized map (never nil) so JSON serializes to {} instead of null.
func ConvertMetadataToLabels(source models.Metadata) *api.Labels {
	labels := make(api.Labels)
	for k, v := range source {
		labels[k] = v
	}
	return &labels
}

var iso8601ToWindowSize = map[string]meter.WindowSize{
	"PT1M": meter.WindowSizeMinute,
	"PT1H": meter.WindowSizeHour,
	"P1D":  meter.WindowSizeDay,
	"P1M":  meter.WindowSizeMonth,
}

var windowSizeToISO8601 = map[meter.WindowSize]string{
	meter.WindowSizeMinute: "PT1M",
	meter.WindowSizeHour:   "PT1H",
	meter.WindowSizeDay:    "P1D",
	meter.WindowSizeMonth:  "P1M",
}

func ConvertISO8601DurationToWindowSize(duration string) (meter.WindowSize, error) {
	ws, ok := iso8601ToWindowSize[duration]
	if !ok {
		return "", NewInvalidWindowSizeError(duration)
	}
	return ws, nil
}

func ConvertWindowSizeToISO8601Duration(ws meter.WindowSize) (string, error) {
	if d, ok := windowSizeToISO8601[ws]; ok {
		return d, nil
	}
	return "", fmt.Errorf("unknown WindowSize: %q", ws)
}

func ConvertMeterQueryRowToAPI(row meter.MeterQueryRow) api.MeterQueryRow {
	dimensions := api.MeterQueryRow_Dimensions{
		CustomerId: row.CustomerID,
		Subject:    row.Subject,
	}

	if len(row.GroupBy) > 0 {
		dimensions.AdditionalProperties = make(map[string]string, len(row.GroupBy))

		for key, value := range row.GroupBy {
			switch key {
			case dimensionSubject:
				dimensions.Subject = value
			case dimensionCustomerID:
				dimensions.CustomerId = value
			default:
				if value != nil {
					dimensions.AdditionalProperties[key] = *value
				}
			}
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

// ExtractStringsFromQueryFilter extracts a flat list of string values from a QueryFilterString.
// Only the eq and in operators are supported; an error is returned if any other operator is set.
func ExtractStringsFromQueryFilter(f *api.QueryFilterString, fieldPath ...string) ([]string, error) {
	if f == nil {
		return nil, nil
	}

	if f.Neq != nil || f.Nin != nil ||
		f.Contains != nil || f.Ncontains != nil ||
		f.And != nil || f.Or != nil {
		return nil, NewUnsupportedFilterOperatorError(fieldPath...)
	}
	if f.Eq != nil && f.In != nil {
		return nil, NewUnsupportedFilterOperatorError(fieldPath...)
	}

	var result []string
	if f.Eq != nil {
		result = append(result, *f.Eq)
	}
	if f.In != nil {
		result = append(result, *f.In...)
	}
	return result, nil
}
