package billing

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/models"
)

func TestValidationIssueAttributes(t *testing.T) {
	issue := ValidationIssue{
		Severity: ValidationIssueSeverityWarning,
		Message:  "invoice line needs attention",
		Code:     "line_context",
		Attributes: models.Annotations{
			"invoice": map[string]any{"id": "invoice-1"},
			"line":    map[string]any{"id": "line-1"},
		},
	}

	t.Run("preserved through error extraction", func(t *testing.T) {
		wrapped := ValidationWithComponent("test", issue)
		require.ErrorIs(t, wrapped, issue)

		issues, err := ToValidationIssues(wrapped)
		require.NoError(t, err)
		require.Equal(t, issue.Attributes, issues[0].Attributes)
	})

	t.Run("code determines error identity", func(t *testing.T) {
		require.ErrorIs(t, issue, ValidationIssue{
			Severity: ValidationIssueSeverityCritical,
			Message:  "different context",
			Code:     issue.Code,
		})
		require.NotErrorIs(t, issue, ValidationIssue{Message: issue.Message})
	})

	t.Run("encoded as API attributes", func(t *testing.T) {
		extension := issue.EncodeAsErrorExtension()
		require.Equal(t, issue.Attributes, extension["attributes"])
		require.NotContains(t, extension, "annotations")
	})

	t.Run("deep cloned", func(t *testing.T) {
		clone, err := issue.Clone()
		require.NoError(t, err)

		clone.Attributes["invoice"].(map[string]any)["id"] = "invoice-2"
		require.Equal(t, "invoice-1", issue.Attributes["invoice"].(map[string]any)["id"])
	})
}

func TestValidationIssuesHasWithComponentCode(t *testing.T) {
	existing := ValidationIssue{
		Severity:  ValidationIssueSeverityWarning,
		Message:   "existing",
		Code:      "existing",
		Component: "component",
	}
	issues := ValidationIssues{existing}

	require.True(t, issues.HasWithComponentCode(existing.Component, existing.Code))
	require.False(t, issues.HasWithComponentCode("other", existing.Code))
	require.False(t, issues.HasWithComponentCode(existing.Component, "other"))
}

func TestValidationIssuesWithout(t *testing.T) {
	issue := ValidationIssue{
		Severity:  ValidationIssueSeverityCritical,
		Message:   "blocked",
		Code:      "blocked",
		Component: "component",
	}
	otherComponentIssue := issue
	otherComponentIssue.Component = "other"
	existing := ValidationIssue{
		Severity:  ValidationIssueSeverityWarning,
		Message:   "existing",
		Code:      "existing",
		Component: "component",
	}

	t.Run("removes every issue with the code and component", func(t *testing.T) {
		original := ValidationIssues{issue, existing, otherComponentIssue, issue}

		issues := original.Without(issue.Component, issue.Code)

		require.Equal(t, ValidationIssues{existing, otherComponentIssue}, issues)
		require.Len(t, original, 4)
	})

	t.Run("keeps the collection unchanged when the issue is absent", func(t *testing.T) {
		original := ValidationIssues{existing, otherComponentIssue}

		issues := original.Without(issue.Component, issue.Code)

		require.Equal(t, original, issues)
	})
}

func TestValidationWithAttributes(t *testing.T) {
	baseIssue := ValidationIssue{
		Severity: ValidationIssueSeverityWarning,
		Message:  "invoice line needs attention",
		Code:     "line_context",
		Attributes: models.Annotations{
			"base":       "base",
			"precedence": "base",
		},
	}

	t.Run("merges nested attributes with outermost precedence", func(t *testing.T) {
		err := ValidationWithAttributes(
			models.Annotations{
				"outer":      "outer",
				"precedence": "outer",
			},
			ValidationWithAttributes(
				models.Annotations{
					"inner":      "inner",
					"precedence": "inner",
				},
				baseIssue,
			),
		)

		issues, systemErr := ToValidationIssues(err)
		require.NoError(t, systemErr)
		require.Equal(t, models.Annotations{
			"base":       "base",
			"inner":      "inner",
			"outer":      "outer",
			"precedence": "outer",
		}, issues[0].Attributes)
	})

	t.Run("applies attributes to every joined issue", func(t *testing.T) {
		attributes := models.Annotations{
			"invoice": map[string]any{"id": "invoice-1"},
		}
		err := ValidationWithAttributes(
			attributes,
			errors.Join(baseIssue, NewValidationError("second", "second issue")),
		)

		issues, systemErr := ToValidationIssues(err)
		require.NoError(t, systemErr)
		require.Len(t, issues, 2)
		require.Equal(t, "invoice-1", issues[0].Attributes["invoice"].(map[string]any)["id"])
		require.Equal(t, "invoice-1", issues[1].Attributes["invoice"].(map[string]any)["id"])

		issues[0].Attributes["invoice"].(map[string]any)["id"] = "invoice-2"
		require.Equal(t, "invoice-1", issues[1].Attributes["invoice"].(map[string]any)["id"])
		require.Equal(t, "invoice-1", attributes["invoice"].(map[string]any)["id"])
	})

	t.Run("classifies an ordinary error as a validation issue", func(t *testing.T) {
		err := ValidationWithAttributes(
			models.Annotations{"invoice_id": "invoice-1"},
			errors.New("invoice context unavailable"),
		)

		issues, systemErr := ToValidationIssues(err)
		require.NoError(t, systemErr)
		require.Equal(t, ValidationIssues{
			{
				Severity:   ValidationIssueSeverityCritical,
				Message:    "invoice context unavailable",
				Attributes: models.Annotations{"invoice_id": "invoice-1"},
			},
		}, issues)
	})

	t.Run("does not alias input attributes", func(t *testing.T) {
		attributes := models.Annotations{
			"invoice": map[string]any{"id": "invoice-1"},
		}
		err := ValidationWithAttributes(attributes, baseIssue)

		issues, systemErr := ToValidationIssues(err)
		require.NoError(t, systemErr)

		issues[0].Attributes["invoice"].(map[string]any)["id"] = "invoice-2"
		require.Equal(t, "invoice-1", attributes["invoice"].(map[string]any)["id"])
	})

	t.Run("returns nil for a nil error", func(t *testing.T) {
		require.NoError(t, ValidationWithAttributes(models.Annotations{"unused": true}, nil))
	})
}

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
