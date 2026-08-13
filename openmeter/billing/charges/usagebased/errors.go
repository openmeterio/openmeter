package usagebased

import (
	"net/http"

	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/models"
)

const ErrCodeChargeTotalIsNegative models.ErrorCode = "charge_total_is_negative"

var ErrChargeTotalIsNegative = models.NewValidationIssue(
	ErrCodeChargeTotalIsNegative,
	"charge total is negative",
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeActiveRealizationRunAlreadyExists models.ErrorCode = "active_realization_run_already_exists"

var ErrActiveRealizationRunAlreadyExists = models.NewValidationIssue(
	ErrCodeActiveRealizationRunAlreadyExists,
	"an active realization run already exists for this charge, please finalize any draft invoices for the customer first",
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)
