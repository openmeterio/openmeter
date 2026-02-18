package charges

import (
	"net/http"

	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/models"
)

const ErrCodeChargeNamespaceEmpty models.ErrorCode = "charge_namespace_empty"

var ErrChargeNamespaceEmpty = models.NewValidationIssue(
	ErrCodeChargeNamespaceEmpty,
	"namespace must not be empty",
	models.WithFieldString("namespace"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeChargeNotFound models.ErrorCode = "charge_not_found"

var ErrChargeNotFound = models.NewValidationIssue(
	ErrCodeChargeNotFound,
	"charge not found",
	models.WithFieldString("id"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusNotFound),
)

func NewChargeNotFoundError(namespace, id string) error {
	return ErrChargeNotFound.WithAttr("namespace", namespace).WithAttr("id", id)
}

const ErrCodeChargeInvalid models.ErrorCode = "charge_invalid"

var ErrChargeInvalid = models.NewValidationIssue(
	ErrCodeChargeInvalid,
	"charge is invalid",
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)
