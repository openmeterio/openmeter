package customerscredits

import (
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/models"
)

const errCodeCreditGrantExternalSettlementStatusInvalid models.ErrorCode = "credit_grant_external_settlement_status_invalid"

func newCreditGrantExternalSettlementStatusInvalid(status string) error {
	return models.NewValidationIssue(
		errCodeCreditGrantExternalSettlementStatusInvalid,
		fmt.Sprintf("unsupported credit grant settlement status: %s", status),
		models.WithCriticalSeverity(),
		commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
		models.WithFieldString("status"),
		models.WithAttribute("status", status),
	)
}

const errCodeCreditGrantVoidPaymentAdjustmentInvalid models.ErrorCode = "credit_grant_void_payment_adjustment_invalid"

func newCreditGrantVoidPaymentAdjustmentInvalid(adjustment string) error {
	return models.NewValidationIssue(
		errCodeCreditGrantVoidPaymentAdjustmentInvalid,
		fmt.Sprintf("unsupported credit grant void payment adjustment: %s", adjustment),
		models.WithCriticalSeverity(),
		commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
		models.WithFieldString("payment_adjustment"),
		models.WithAttribute("payment_adjustment", adjustment),
	)
}
