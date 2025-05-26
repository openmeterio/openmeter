package http

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport/encoder"
	"github.com/openmeterio/openmeter/pkg/models"
)

func ValidationErrorEncoder(kind ResourceKind) encoder.ErrorEncoder {
	return func(ctx context.Context, err error, w http.ResponseWriter, r *http.Request) bool {
		issues, err := models.AsValidationIssues(err)

		if err == nil && len(issues) > 0 {
			err = validationError{
				kind:   kind,
				issues: issues,
			}

			return commonhttp.HandleErrorIfTypeMatches[validationError](ctx, http.StatusBadRequest, err, w, validationErrorToExtensions)
		}

		return false
	}
}

var _ error = (*validationError)(nil)

type validationError struct {
	kind   ResourceKind
	issues models.ValidationIssues
}

func (e validationError) Error() string {
	return "invalid " + string(e.kind)
}

func validationErrorToExtensions(err validationError) map[string]interface{} {
	if len(err.issues) == 0 {
		return nil
	}

	var issues []map[string]interface{}
	for _, issue := range err.issues {
		issues = append(issues, issue.AsErrorExtension())
	}

	return map[string]interface{}{
		"validationErrors": issues,
	}
}
