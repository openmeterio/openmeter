package creditgrant

import (
	"net/http"

	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/models"
)

const ErrCodeCreditGrantFeatureFiltersUnsupported models.ErrorCode = "credit_grant_feature_filters_unsupported"

func newCreditGrantFeatureFiltersUnsupportedError(featureCount int) error {
	return models.NewValidationIssue(
		ErrCodeCreditGrantFeatureFiltersUnsupported,
		"credit grant feature filters are not supported yet",
		models.WithCriticalSeverity(),
		commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
		models.WithFieldString("filters", "features"),
		models.WithAttribute("feature_count", featureCount),
	)
}
