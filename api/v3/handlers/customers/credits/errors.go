package customerscredits

import (
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/models"
)

const errCodeCreditGrantExternalSettlementStatusInvalid models.ErrorCode = "credit_grant_external_settlement_status_invalid"

var errCreditGrantExternalSettlementStatusInvalid = models.NewValidationIssue(
	errCodeCreditGrantExternalSettlementStatusInvalid,
	"unsupported credit grant settlement status",
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
	models.WithFieldString("status"),
)

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
