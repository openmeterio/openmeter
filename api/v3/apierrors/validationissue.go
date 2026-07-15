package apierrors

import (
	"github.com/openmeterio/openmeter/pkg/models"
)

// InvalidParametersFromValidationIssues maps domain validation issues to the
// v3 invalid_parameters wire structure. Domain error codes are deliberately
// not emitted as the machine-readable rule: rule is a closed enum in the API
// spec, and domain codes are not members of it, so they travel only inside the
// human-readable reason.
func InvalidParametersFromValidationIssues(issues models.ValidationIssues) InvalidParameters {
	if len(issues) == 0 {
		return nil
	}

	params := make(InvalidParameters, 0, len(issues))
	for _, issue := range issues {
		params = append(params, InvalidParameter{
			Field:  issue.Field().JSONPath(),
			Reason: issue.Message(),
			Source: InvalidParamSourceBody,
		})
	}

	return params
}
