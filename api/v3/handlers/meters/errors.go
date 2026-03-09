package meters

import (
	"net/http"

	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/models"
)

const ErrCodeReservedDimension models.ErrorCode = "reserved_dimension"

var ErrReservedDimension = models.NewValidationIssue(
	ErrCodeReservedDimension,
	"dimension name is reserved",
	models.WithFieldString("dimensions"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

func NewReservedDimensionError(dimension string) error {
	return ErrReservedDimension.WithPathString("dimensions", dimension).WithAttr("value", dimension)
}
