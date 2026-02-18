package account

import (
	"net/http"

	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/models"
)

const ErrCodeDimensionConflict models.ErrorCode = "dimension_conflict"

var ErrDimensionConflict = models.NewValidationIssue(
	ErrCodeDimensionConflict,
	"dimension conflict, a dimension with the same key and value already exists",
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusConflict),
)
