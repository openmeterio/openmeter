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
	"cannot settle an unauthorized payment",
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodePaymentBookingMismatch models.ErrorCode = "payment_booking_mismatch"

var ErrPaymentBookingMismatch = models.NewValidationIssue(
	ErrCodePaymentBookingMismatch,
	"payment is booked for a different invoice line",
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodePaymentMissingOnFinalCharge models.ErrorCode = "payment_missing_on_final_charge"

var ErrPaymentMissingOnFinalCharge = models.NewValidationIssue(
	ErrCodePaymentMissingOnFinalCharge,
	"final charge has no payment booking",
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodePaymentNotSettledOnFinalCharge models.ErrorCode = "payment_not_settled_on_final_charge"

var ErrPaymentNotSettledOnFinalCharge = models.NewValidationIssue(
	ErrCodePaymentNotSettledOnFinalCharge,
	"final charge has an unsettled payment booking",
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)
