//go:generate go run github.com/jmattheis/goverter/cmd/goverter gen ./
package meters

import (
	"context"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/handlers/meters/query"
	"github.com/openmeterio/openmeter/api/v3/labels"
	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
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
// goverter:extend FromAPIMeterAggregation
// goverter:extend ToAPIMeterAggregation
// goverter:extend ConvertMetadataAnnotationsToLabels
// goverter:extend ConvertLabelsToMetadata
// goverter:extend IntToFloat32
var (
	// goverter:context namespace
	// goverter:map Namespace | NamespaceFromContext
	// goverter:map Dimensions GroupBy
	// goverter:map Labels Metadata
	// goverter:map EventsFrom EventFrom
	// goverter:ignore Annotations
	// goverter:ignore inputOptions
	FromAPICreateMeterRequest func(namespace string, createMeterRequest apiv3models.CreateMeterRequest) (meter.CreateMeterInput, error)
	// goverter:map GroupBy Dimensions
	// goverter:map EventFrom EventsFrom
	// goverter:map ManagedResource.ID Id
	// goverter:map ManagedResource.Description Description
	// goverter:map ManagedResource.Name Name
	// goverter:map ManagedResource.ManagedModel.CreatedAt CreatedAt
	// goverter:map ManagedResource.ManagedModel.UpdatedAt UpdatedAt
	// goverter:map ManagedResource.ManagedModel.DeletedAt DeletedAt
	// goverter:map . Labels | ConvertMetadataAnnotationsToLabels
	ToAPIMeter                      func(meter.Meter) apiv3models.Meter
	ToAPIMeterPagePaginatedResponse func(meters response.PagePaginationResponse[meter.Meter]) apiv3models.MeterPagePaginatedResponse
)

var ConvertLabelsToMetadata = labels.ToMetadata

func ConvertMetadataAnnotationsToLabels(source meter.Meter) *apiv3models.Labels {
	return labels.FromMetadataAnnotations(source.Metadata, source.Annotations)
}

//goverter:context namespace
func NamespaceFromContext(namespace string) string {
	return namespace
}

func ToAPIMeterAggregation(aggregation meter.MeterAggregation) apiv3models.MeterAggregation {
	switch aggregation {
	case meter.MeterAggregationSum:
		return apiv3models.MeterAggregationSum
	case meter.MeterAggregationCount:
		return apiv3models.MeterAggregationCount
	case meter.MeterAggregationUniqueCount:
		return apiv3models.MeterAggregationUniqueCount
	case meter.MeterAggregationAvg:
		return apiv3models.MeterAggregationAvg
	case meter.MeterAggregationMin:
		return apiv3models.MeterAggregationMin
	case meter.MeterAggregationMax:
		return apiv3models.MeterAggregationMax
	case meter.MeterAggregationLatest:
		return apiv3models.MeterAggregationLatest
	}

	return apiv3models.MeterAggregation("")
}

func FromAPIMeterAggregation(aggregation apiv3models.MeterAggregation) meter.MeterAggregation {
	switch aggregation {
	case apiv3models.MeterAggregationSum:
		return meter.MeterAggregationSum
	case apiv3models.MeterAggregationCount:
		return meter.MeterAggregationCount
	case apiv3models.MeterAggregationUniqueCount:
		return meter.MeterAggregationUniqueCount
	case apiv3models.MeterAggregationAvg:
		return meter.MeterAggregationAvg
	case apiv3models.MeterAggregationMin:
		return meter.MeterAggregationMin
	case apiv3models.MeterAggregationMax:
		return meter.MeterAggregationMax
	case apiv3models.MeterAggregationLatest:
		return meter.MeterAggregationLatest
	}

	return ""
}

func IntToFloat32(i int) float32 {
	return float32(i)
}

func FromAPIUpdateMeterRequest(namespace string, meterID string, body apiv3models.UpdateMeterRequest) (meter.UpdateMeterInput, error) {
	input := meter.UpdateMeterInput{
		ID: models.NamespacedID{
			Namespace: namespace,
			ID:        meterID,
		},
	}

	if body.Name != nil {
		input.Name = *body.Name
	}

	input.Description = body.Description

	if body.Dimensions != nil {
		input.GroupBy = *body.Dimensions
	}

	if body.Labels != nil {
		metadata, err := labels.ToMetadata(body.Labels)
		if err != nil {
			return meter.UpdateMeterInput{}, err
		}

		input.Metadata = metadata
	}

	return input, nil
}

// ConvertMetadataToLabels converts models.Metadata to apiv3models.Labels.
// Always returns an initialized map (never nil) so JSON serializes to {} instead of null.
func ConvertMetadataToLabels(source models.Metadata) *apiv3models.Labels {
	labels := make(apiv3models.Labels)
	for k, v := range source {
		labels[k] = v
	}

	return &labels
}

func ToAPIMeterQueryRow(row meter.MeterQueryRow) apiv3models.MeterQueryRow {
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

	return apiv3models.MeterQueryRow{
		Value:      alpacadecimal.NewFromFloat(row.Value).String(),
		From:       row.WindowStart,
		To:         row.WindowEnd,
		Dimensions: dimensions,
	}
}

func ToAPIMeterQueryResult(from *apiv3models.DateTime, to *apiv3models.DateTime, rows []meter.MeterQueryRow) apiv3models.MeterQueryResult {
	return apiv3models.MeterQueryResult{
		From: from,
		To:   to,
		Data: lo.Map(rows, func(row meter.MeterQueryRow, _ int) apiv3models.MeterQueryRow {
			return ToAPIMeterQueryRow(row)
		}),
	}
}

func FromAPIMeterSortField(ctx context.Context, field string) (meter.OrderBy, error) {
	switch field {
	case "key":
		return meter.OrderByKey, nil
	case "name":
		return meter.OrderByName, nil
	case "aggregation":
		return meter.OrderByAggregation, nil
	case "created_at":
		return meter.OrderByCreatedAt, nil
	case "updated_at":
		return meter.OrderByUpdatedAt, nil
	default:
		return "", apierrors.NewUnsupportedSortFieldError(
			ctx, field, "key", "name", "aggregation", "created_at", "updated_at",
		)
	}
}
