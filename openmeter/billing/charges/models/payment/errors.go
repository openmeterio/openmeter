package payment

import (
	"net/http"

	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/models"
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
