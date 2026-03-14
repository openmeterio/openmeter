package creditpurchase

import (
	"net/http"

	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/models"
)

const ErrCodeCreditPurchaseChargeNotActive models.ErrorCode = "credit_purchase_charge_not_active"

var ErrCreditPurchaseChargeNotActive = models.NewValidationIssue(
	ErrCodeCreditPurchaseChargeNotActive,
	"credit purchase charge is not active",
	models.WithFieldString("namespace"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)
