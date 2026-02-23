package taxcode

import (
	"net/http"

	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/models"
)

const ErrCodeResourceNamespaceEmpty models.ErrorCode = "resource_namespace_empty"

var ErrResourceNamespaceEmpty = models.NewValidationIssue(
	ErrCodeResourceNamespaceEmpty,
	"namespace must not be empty",
	models.WithFieldString("namespace"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeResourceKeyEmpty models.ErrorCode = "resource_key_empty"

var ErrResourceKeyEmpty = models.NewValidationIssue(
	ErrCodeResourceKeyEmpty,
	"key must not be empty",
	models.WithFieldString("key"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeResourceNameEmpty models.ErrorCode = "resource_name_empty"

var ErrResourceNameEmpty = models.NewValidationIssue(
	ErrCodeResourceNameEmpty,
	"name must not be empty",
	models.WithFieldString("name"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeAppTypesMustBeUnique models.ErrorCode = "app_types_must_be_unique"

var ErrAppTypesMustBeUnique = models.NewValidationIssue(
	ErrCodeAppTypesMustBeUnique,
	"app types must be unique",
	models.WithFieldString("app_mappings"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeTaxCodeNotFound models.ErrorCode = "tax_code_not_found"

var ErrTaxCodeNotFound = models.NewValidationIssue(
	ErrCodeTaxCodeNotFound,
	"tax code not found",
	models.WithFieldString("id"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusNotFound),
)

func NewTaxCodeNotFoundError(id string) error {
	return ErrTaxCodeNotFound.WithAttr("id", id)
}

const ErrCodeTaxCodeEmpty models.ErrorCode = "tax_code_empty"

var ErrTaxCodeEmpty = models.NewValidationIssue(
	ErrCodeTaxCodeEmpty,
	"tax code cannot be empty",
	models.WithFieldString("app_mappings"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)

const ErrCodeTaxCodeStripeInvalid models.ErrorCode = "tax_code_stripe_invalid"

var ErrTaxCodeStripeInvalid = models.NewValidationIssue(
	ErrCodeTaxCodeStripeInvalid,
	"stripe tax code must be in the format of txcd_12345678",
	models.WithFieldString("app_mappings"),
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)
