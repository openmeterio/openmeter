package billing

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

func TestValidationWithComponentPrecedence(t *testing.T) {
	baseIssue := ValidationIssue{
		Severity:  ValidationIssueSeverityWarning,
		Message:   "canonical message",
		Code:      "canonical_code",
		Component: "issue-component",
		Path:      "original/path",
	}
	err := ValidationWithComponent(
		"outer-component",
		ValidationWithComponent("inner-component", baseIssue),
	)

	issues, systemErr := ToValidationIssues(err)
	require.NoError(t, systemErr)
	require.Equal(t, ValidationIssues{
		{
			Severity:  baseIssue.Severity,
			Message:   baseIssue.Message,
			Code:      baseIssue.Code,
			Component: "outer-component",
			Path:      "/original/path",
		},
	}, issues)
}

func TestAsError(t *testing.T) {
	issues := ValidationIssues{
		{
			Severity:  ValidationIssueSeverityCritical,
			Message:   "error1",
			Component: "component1",
			Path:      "/some/path/from/component1",
		},
	}

	err := issues.AsError()

	validationIssues, err := ToValidationIssues(err)
	require.NoError(t, err)
	require.Equal(t, issues, validationIssues)
}

func TestValidationWithMessagef(t *testing.T) {
	baseIssue := ValidationIssue{
		Severity:  ValidationIssueSeverityWarning,
		Message:   "canonical message",
		Code:      "canonical_code",
		Component: "original-component",
		Path:      "original/path",
	}

	t.Run("adds formatted context and preserves error identity", func(t *testing.T) {
		err := ValidationWithMessagef(baseIssue, "feature[%s]", "requests")

		require.EqualError(t, err, "feature[requests]: canonical message")
		require.ErrorIs(t, err, baseIssue)

		issues, systemErr := ToValidationIssues(err)
		require.NoError(t, systemErr)
		require.Equal(t, ValidationIssues{
			{
				Severity:  baseIssue.Severity,
				Message:   "feature[requests]: canonical message",
				Code:      baseIssue.Code,
				Component: baseIssue.Component,
				Path:      "/original/path",
			},
		}, issues)
	})

	t.Run("composes nested message component and field context", func(t *testing.T) {
		err := ValidationWithMessagef(
			ValidationWithComponent(
				"outer-component",
				ValidationWithFieldPrefix(
					"lines/line-1",
					ValidationWithMessagef(baseIssue, "inner[%d]", 42),
				),
			),
			"outer",
		)

		issues, systemErr := ToValidationIssues(err)
		require.NoError(t, systemErr)
		require.Equal(t, ValidationIssues{
			{
				Severity:  baseIssue.Severity,
				Message:   "outer: inner[42]: canonical message",
				Code:      baseIssue.Code,
				Component: "outer-component",
				Path:      "/lines/line-1/original/path",
			},
		}, issues)
	})

	t.Run("prefixes every validation issue in a joined error", func(t *testing.T) {
		secondIssue := NewValidationError("second_code", "second message")
		err := ValidationWithMessagef(errors.Join(baseIssue, secondIssue), "shared context")

		issues, systemErr := ToValidationIssues(err)
		require.NoError(t, systemErr)
		require.Equal(t, ValidationIssues{
			{
				Severity:  baseIssue.Severity,
				Message:   "shared context: canonical message",
				Code:      baseIssue.Code,
				Component: baseIssue.Component,
				Path:      "/original/path",
			},
			{
				Severity: secondIssue.Severity,
				Message:  "shared context: second message",
				Code:     secondIssue.Code,
			},
		}, issues)
	})

	t.Run("does not promote a system error", func(t *testing.T) {
		systemErr := errors.New("database unavailable")
		err := ValidationWithMessagef(systemErr, "loading invoice[%s]", "invoice-1")

		require.EqualError(t, err, "loading invoice[invoice-1]: database unavailable")
		require.ErrorIs(t, err, systemErr)

		issues, extractionErr := ToValidationIssues(err)
		require.Nil(t, issues)
		require.Equal(t, err, extractionErr)
	})

	t.Run("returns nil for a nil error", func(t *testing.T) {
		require.NoError(t, ValidationWithMessagef(nil, "unused %s", "context"))
	})
}
