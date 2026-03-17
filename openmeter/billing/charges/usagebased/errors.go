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

const ErrCodeCreditAllocationsDoNotMatchTotal models.ErrorCode = "credit_allocations_do_not_match_total"

var ErrCreditAllocationsDoNotMatchTotal = models.NewValidationIssue(
	ErrCodeCreditAllocationsDoNotMatchTotal,
	"credit allocations do not match total",
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)
