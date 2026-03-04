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

const ErrCodeUnsupported models.ErrorCode = "unsupported"

var ErrUnsupported = models.NewValidationIssue(
	ErrCodeUnsupported,
	"unsupported",
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusInternalServerError),
)

const ErrCodeCreditRealizationsAlreadyAllocated models.ErrorCode = "credit_realizations_already_allocated"

var ErrCreditRealizationsAlreadyAllocated = models.NewValidationIssue(
	ErrCodeCreditRealizationsAlreadyAllocated,
	"credit realizations already allocated",
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodePaymentAlreadyAuthorized models.ErrorCode = "payment_already_authorized"

var ErrPaymentAlreadyAuthorized = models.NewValidationIssue(
	ErrCodePaymentAlreadyAuthorized,
	"payment already authorized",
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodePaymentAlreadySettled models.ErrorCode = "payment_already_settled"

var ErrPaymentAlreadySettled = models.NewValidationIssue(
	ErrCodePaymentAlreadySettled,
	"payment already settled",
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeCannotSettleNotAuthorizedPayment models.ErrorCode = "cannot_settle_not_authorized_payment"

var ErrCannotSettleNotAuthorizedPayment = models.NewValidationIssue(
	ErrCodeCannotSettleNotAuthorizedPayment,
	"cannot settle not authorized payment",
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)
