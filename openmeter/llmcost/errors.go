package llmcost

import (
	"net/http"

	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/models"
)

const ErrCodePriceNotFound models.ErrorCode = "llm_cost_price_not_found"

var ErrPriceNotFound = models.NewValidationIssue(
	ErrCodePriceNotFound,
	"llm cost price not found",
	models.WithFieldString("id"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusNotFound),
)

func NewPriceNotFoundError(id string) error {
	return ErrPriceNotFound.WithAttr("id", id)
}

const ErrCodeProviderEmpty models.ErrorCode = "llm_cost_provider_empty"

var ErrProviderEmpty = models.NewValidationIssue(
	ErrCodeProviderEmpty,
	"provider must not be empty",
	models.WithFieldString("provider"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeModelIDEmpty models.ErrorCode = "llm_cost_model_id_empty"

var ErrModelIDEmpty = models.NewValidationIssue(
	ErrCodeModelIDEmpty,
	"model_id must not be empty",
	models.WithFieldString("model_id"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeNamespaceEmpty models.ErrorCode = "llm_cost_namespace_empty"

var ErrNamespaceEmpty = models.NewValidationIssue(
	ErrCodeNamespaceEmpty,
	"namespace must not be empty",
	models.WithFieldString("namespace"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodePriceIDEmpty models.ErrorCode = "llm_cost_price_id_empty"

var ErrPriceIDEmpty = models.NewValidationIssue(
	ErrCodePriceIDEmpty,
	"price id must not be empty",
	models.WithFieldString("id"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodePriceMustBeNonNegative models.ErrorCode = "llm_cost_price_non_negative"

var ErrPriceMustBeNonNegative = models.NewValidationIssue(
	ErrCodePriceMustBeNonNegative,
	"price must be non-negative",
	models.WithFieldString("pricing"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeEffectiveFromAfterTo models.ErrorCode = "llm_cost_effective_from_after_to"

var ErrEffectiveFromAfterTo = models.NewValidationIssue(
	ErrCodeEffectiveFromAfterTo,
	"effective_from must not be after effective_to",
	models.WithFieldString("effective_from"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeInvalidPriceSource models.ErrorCode = "llm_cost_invalid_source"

var ErrInvalidPriceSource = models.NewValidationIssue(
	ErrCodeInvalidPriceSource,
	"invalid price source",
	models.WithFieldString("source"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)
