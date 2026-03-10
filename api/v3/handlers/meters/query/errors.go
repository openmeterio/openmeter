package query

import (
	"net/http"

	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/models"
)

const ErrCodeInvalidWindowSize models.ErrorCode = "invalid_window_size"

var ErrInvalidWindowSize = models.NewValidationIssue(
	ErrCodeInvalidWindowSize,
	"unsupported window size duration, supported durations are: PT1M (minute), PT1H (hour), P1D (day), P1M (month)",
	models.WithFieldString("granularity"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

func NewInvalidWindowSizeError(duration string) error {
	return ErrInvalidWindowSize.WithAttr("value", duration)
}

const ErrCodeInvalidTimeZone models.ErrorCode = "invalid_time_zone"

var ErrInvalidTimeZone = models.NewValidationIssue(
	ErrCodeInvalidTimeZone,
	"invalid time zone",
	models.WithFieldString("time_zone"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

func NewInvalidTimeZoneError(tz string) error {
	return ErrInvalidTimeZone.WithAttr("value", tz)
}

const ErrCodeInvalidGroupBy models.ErrorCode = "invalid_group_by"

var ErrInvalidGroupBy = models.NewValidationIssue(
	ErrCodeInvalidGroupBy,
	"invalid group by dimension",
	models.WithFieldString("group_by_dimensions"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

func NewInvalidGroupByError(dimension string) error {
	return ErrInvalidGroupBy.WithAttr("value", dimension)
}

const ErrCodeInvalidDimensionFilter models.ErrorCode = "invalid_dimension_filter"

var ErrInvalidDimensionFilter = models.NewValidationIssue(
	ErrCodeInvalidDimensionFilter,
	"invalid dimension filter",
	models.WithFieldString("filters", "dimensions"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

func NewInvalidDimensionFilterError(dimension string) error {
	return ErrInvalidDimensionFilter.WithPathString("filters", "dimensions", dimension)
}

const ErrCodeUnsupportedFilterOperator models.ErrorCode = "unsupported_filter_operator"

var ErrUnsupportedFilterOperator = models.NewValidationIssue(
	ErrCodeUnsupportedFilterOperator,
	"only eq and in filter operators are supported for this field",
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

func NewUnsupportedFilterOperatorError(path ...string) error {
	return ErrUnsupportedFilterOperator.WithPathString(append([]string{"filters"}, path...)...)
}

const ErrCodeUnknownFilterOperator models.ErrorCode = "unknown_filter_operator"

var ErrUnknownFilterOperator = models.NewValidationIssue(
	ErrCodeUnknownFilterOperator,
	"unknown filter operator",
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

func NewUnknownFilterOperatorError(operator string, path ...string) error {
	return ErrUnknownFilterOperator.
		WithPathString(append([]string{"filters"}, path...)...).
		WithAttr("operator", operator)
}

const ErrCodeCustomerNotFound models.ErrorCode = "customer_not_found"

var ErrCustomerNotFound = models.NewValidationIssue(
	ErrCodeCustomerNotFound,
	"customer not found",
	models.WithFieldString("filters", "dimensions", DimensionCustomerID),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusNotFound),
)

func NewCustomerNotFoundError(id string) error {
	return ErrCustomerNotFound.WithAttr(DimensionCustomerID, id)
}
