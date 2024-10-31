package billingentity

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidationIssueParsing(t *testing.T) {
	quantityNegativeErr := NewValidationError("quantity_negative", "Quantity is negative")
	quantityZeroWarn := NewValidationWarning("quantity_zero", "Quantity is zero")

	appCannotSyncErr := errors.New("app_cannot_sync")
	appMissingCountry := NewValidationError("app_missing_country", "Country is missing")

	validationError := errors.Join(
		fmt.Errorf("recalculating invoice: %w",
			ValidationWithComponent("openmeter",
				ValidationWithFieldPrefix("lines/ID",
					errors.Join(
						ValidationWithFieldPrefix("qty", quantityNegativeErr),
						quantityZeroWarn)),
			),
		),
		fmt.Errorf("app: %w",
			ValidationWithComponent("app",
				errors.Join(appCannotSyncErr, appMissingCountry))),
	)

	mockError := fmt.Errorf("error: %w", fmt.Errorf("error2: %w", validationError))

	issues, err := ToValidationIssues(mockError)

	require.NoError(t, err)
	require.Equal(t, ValidationIssues{
		{
			Severity:  quantityNegativeErr.Severity,
			Message:   quantityNegativeErr.Message,
			Code:      quantityNegativeErr.Code,
			Component: "openmeter",
			Path:      "/lines/ID/qty",
		},
		{
			Severity:  quantityZeroWarn.Severity,
			Message:   quantityZeroWarn.Message,
			Code:      quantityZeroWarn.Code,
			Component: "openmeter",
			Path:      "/lines/ID",
		},
		{
			Severity:  ValidationIssueSeverityCritical,
			Message:   "app_cannot_sync",
			Component: "app",
		},
		{
			Severity:  appMissingCountry.Severity,
			Message:   appMissingCountry.Message,
			Code:      appMissingCountry.Code,
			Component: "app",
		},
	}, issues)

	// When a top-level error is present that is not a validation issue, we treat this as an error
	issues, err = ToValidationIssues(errors.Join(mockError, errors.New("some other error")))
	require.Len(t, issues, 0)
	require.Error(t, err)
}
