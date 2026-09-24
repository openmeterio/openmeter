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

func TestValidationIssuesHasComponent(t *testing.T) {
	existing := ValidationIssue{
		Severity:  ValidationIssueSeverityWarning,
		Message:   "existing",
		Code:      "existing",
		Component: "component",
	}
	issues := ValidationIssues{existing}

	require.True(t, issues.HasComponent(existing.Component))
	require.False(t, issues.HasComponent("other"))
}

func TestValidationIssuesAllWarnings(t *testing.T) {
	warning := NewValidationWarning("warning", "warning")
	critical := NewValidationError("critical", "critical")

	require.False(t, ValidationIssues(nil).AllWarnings())
	require.True(t, ValidationIssues{warning, warning}.AllWarnings())
	require.False(t, ValidationIssues{warning, critical}.AllWarnings())
}

func TestToValidationIssuesRequireWarningsOnly(t *testing.T) {
	warning := NewValidationWarning("warning", "warning")
	critical := NewValidationError("critical", "critical")
	systemErr := errors.New("system error")

	issues, err := ToValidationIssues(nil, RequireWarningsOnly())
	require.NoError(t, err)
	require.Nil(t, issues)

	issues, err = ToValidationIssues(errors.Join(warning, warning), RequireWarningsOnly())
	require.NoError(t, err)
	require.Len(t, issues, 2)
	require.True(t, issues.AllWarnings())

	for _, test := range []struct {
		name string
		err  error
	}{
		{name: "critical validation issue", err: errors.Join(critical)},
		{name: "mixed validation issues", err: errors.Join(warning, critical)},
		{name: "system error", err: systemErr},
		{name: "warning joined with system error", err: errors.Join(warning, systemErr)},
	} {
		t.Run(test.name, func(t *testing.T) {
			issues, err := ToValidationIssues(test.err, RequireWarningsOnly())
			require.Nil(t, issues)
			require.Same(t, test.err, err)
		})
	}

	issues, err = ToValidationIssues(critical)
	require.NoError(t, err)
	require.Equal(t, ValidationIssues{critical}, issues)
}

func TestValidationIssuesWithoutComponent(t *testing.T) {
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
	sameComponentIssue := existing
	sameComponentIssue.Code = "other-code"

	t.Run("removes every issue owned by the component", func(t *testing.T) {
		original := ValidationIssues{issue, existing, otherComponentIssue, sameComponentIssue}

		issues := original.WithoutComponent(issue.Component)

		require.Equal(t, ValidationIssues{otherComponentIssue}, issues)
		require.Equal(t, ValidationIssues{issue, existing, otherComponentIssue, sameComponentIssue}, original)
	})

	t.Run("keeps the collection unchanged when the issue is absent", func(t *testing.T) {
		original := ValidationIssues{otherComponentIssue}

		issues := original.WithoutComponent(issue.Component)

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

	t.Run("includes attributes in deterministic error text", func(t *testing.T) {
		err := ValidationWithAttributes(
			models.Annotations{
				"second": "two",
				"first":  1,
			},
			baseIssue,
		)

		require.EqualError(t, err, "invoice line needs attention [first=1,second=two]")
		require.ErrorIs(t, err, baseIssue)
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

	t.Run("preserves an ordinary error as a system error", func(t *testing.T) {
		err := ValidationWithAttributes(
			models.Annotations{"invoice_id": "invoice-1"},
			errors.New("invoice context unavailable"),
		)

		issues, systemErr := ToValidationIssues(err)
		require.Nil(t, issues)
		require.Equal(t, err, systemErr)
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
				WrapAsValidationIssue(errors.Join(appCannotSyncErr, appMissingCountry)))),
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

func TestModelValidationIssueConversion(t *testing.T) {
	modelIssue := models.NewValidationIssue(
		models.ErrorCode("model_issue"),
		"model issue",
		models.WithWarningSeverity(),
		models.WithComponent(models.ComponentName("model-component")),
		models.WithAttributes(models.Attributes{
			"model":      "attribute",
			"precedence": "model",
		}),
	)

	t.Run("supports values and pointers", func(t *testing.T) {
		tests := []struct {
			name string
			err  error
		}{
			{name: "value", err: modelIssue},
			{name: "pointer", err: &modelIssue},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				issues, systemErr := ToValidationIssues(test.err)
				require.NoError(t, systemErr)
				require.Equal(t, ValidationIssues{{
					Severity:  ValidationIssueSeverityWarning,
					Message:   "model issue",
					Code:      "model_issue",
					Component: "model-component",
					Attributes: models.Annotations{
						"model":      "attribute",
						"precedence": "model",
					},
				}}, issues)
			})
		}
	})

	t.Run("preserves metadata and surrounding billing context", func(t *testing.T) {
		err := ValidationWithComponent(
			"outer-component",
			ValidationWithFieldPrefix(
				"lines/line-1",
				ValidationWithAttributes(
					models.Annotations{
						"outer":      "attribute",
						"precedence": "outer",
					},
					ValidationWithMessage(
						modelIssue,
						"resolve the model issue",
						models.Attributes{"guidance": "attribute"},
					),
				),
			),
		)

		issues, systemErr := ToValidationIssues(err)
		require.NoError(t, systemErr)
		require.Equal(t, ValidationIssues{{
			Severity:  ValidationIssueSeverityWarning,
			Message:   "resolve the model issue: model issue",
			Code:      "model_issue",
			Component: "outer-component",
			Path:      "/lines/line-1",
			Attributes: models.Annotations{
				"guidance":   "attribute",
				"model":      "attribute",
				"outer":      "attribute",
				"precedence": "outer",
			},
		}}, issues)
	})

	t.Run("does not disappear through cloned issue unwrap chains", func(t *testing.T) {
		clonedIssue := modelIssue.Clone().WithAttrs(models.Attributes{"clone": true})

		issues, systemErr := ToValidationIssues(clonedIssue)
		require.NoError(t, systemErr)
		require.Len(t, issues, 1)
		require.Equal(t, "model_issue", issues[0].Code)
		cloned, ok := issues[0].Attributes["clone"].(bool)
		require.True(t, ok)
		require.True(t, cloned)
		require.Equal(t, "attribute", issues[0].Attributes["model"])
	})

	t.Run("retains system error behavior for mixed trees", func(t *testing.T) {
		systemErr := errors.New("database unavailable")
		err := errors.Join(modelIssue, systemErr)

		issues, extractionErr := ToValidationIssues(err)
		require.Nil(t, issues)
		require.Equal(t, err, extractionErr)
		require.False(t, IsValidationIssueOnly(err))
	})

	t.Run("is recognized by validation issue helpers", func(t *testing.T) {
		require.True(t, IsValidationIssueOnly(modelIssue))

		recorder := ValidationIssueRecorder{}
		require.NoError(t, recorder.Record(
			modelIssue,
			WithAttributes(models.Annotations{"recorder": "attribute"}),
			WithComponent("recorder-component"),
			WithPath("lines/line-1"),
		))

		issues, systemErr := ToValidationIssues(recorder.ErrorsOrNil())
		require.NoError(t, systemErr)
		require.Equal(t, ValidationIssues{{
			Severity:  ValidationIssueSeverityWarning,
			Message:   "model issue",
			Code:      "model_issue",
			Component: "recorder-component",
			Path:      "/lines/line-1",
			Attributes: models.Annotations{
				"model":      "attribute",
				"precedence": "model",
				"recorder":   "attribute",
			},
		}}, issues)
	})

	t.Run("omits model field descriptor while retaining billing field prefix", func(t *testing.T) {
		issueWithField := modelIssue.WithPathString("model", "field")

		issues, systemErr := ToValidationIssues(ValidationWithFieldPrefix("billing/line", issueWithField))
		require.NoError(t, systemErr)
		require.Len(t, issues, 1)
		require.Equal(t, "/billing/line", issues[0].Path)
	})
}

func TestIsValidationIssueOnly(t *testing.T) {
	validationErr := NewValidationError("invalid", "invalid")
	systemErr := errors.New("system error")

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name: "nil",
		},
		{
			name:     "validation issue",
			err:      validationErr,
			expected: true,
		},
		{
			name:     "joined validation issues",
			err:      errors.Join(validationErr, NewValidationWarning("warning", "warning")),
			expected: true,
		},
		{
			name: "system error",
			err:  systemErr,
		},
		{
			name: "mixed validation and system errors",
			err:  errors.Join(validationErr, systemErr),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expected, IsValidationIssueOnly(test.err))
		})
	}
}

func TestValidationContextWrappersPreserveSystemErrors(t *testing.T) {
	systemErr := errors.New("database unavailable")

	tests := []struct {
		name string
		wrap func(error) error
	}{
		{
			name: "component",
			wrap: func(err error) error {
				return ValidationWithComponent("component", err)
			},
		},
		{
			name: "field prefix",
			wrap: func(err error) error {
				return ValidationWithFieldPrefix("lines/line-1", err)
			},
		},
		{
			name: "attributes",
			wrap: func(err error) error {
				return ValidationWithAttributes(models.Annotations{"line_id": "line-1"}, err)
			},
		},
		{
			name: "message",
			wrap: func(err error) error {
				return ValidationWithMessage(err, "retry after the database recovers")
			},
		},
		{
			name: "combined",
			wrap: func(err error) error {
				return ValidationWithComponent(
					"component",
					ValidationWithFieldPrefix(
						"lines/line-1",
						ValidationWithAttributes(
							models.Annotations{"line_id": "line-1"},
							ValidationWithMessage(err, "retry after the database recovers"),
						),
					),
				)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.wrap(systemErr)

			issues, extractionErr := ToValidationIssues(err)
			require.Nil(t, issues)
			require.Equal(t, err, extractionErr)
			require.ErrorIs(t, err, systemErr)
		})
	}
}

func TestWrapAsValidationIssue(t *testing.T) {
	t.Run("converts an ordinary error tree with context", func(t *testing.T) {
		systemErr := errors.New("database unavailable")
		warning := NewValidationWarning("stale_data", "data is stale")
		err := WrapAsValidationIssue(
			ValidationWithComponent(
				"component",
				ValidationWithFieldPrefix(
					"lines/line-1",
					ValidationWithAttributes(
						models.Annotations{"line_id": "line-1"},
						ValidationWithMessage(errors.Join(systemErr, warning), "retry after restoring the database"),
					),
				),
			),
		)

		issues, extractionErr := ToValidationIssues(err)
		require.NoError(t, extractionErr)
		require.ErrorIs(t, err, systemErr)
		require.Equal(t, ValidationIssues{
			{
				Severity:   ValidationIssueSeverityCritical,
				Message:    "retry after restoring the database: database unavailable",
				Component:  "component",
				Path:       "/lines/line-1",
				Attributes: models.Annotations{"line_id": "line-1"},
			},
			{
				Severity:   warning.Severity,
				Message:    "retry after restoring the database: data is stale",
				Code:       warning.Code,
				Component:  "component",
				Path:       "/lines/line-1",
				Attributes: models.Annotations{"line_id": "line-1"},
			},
		}, issues)
	})

	t.Run("converts only the wrapped subtree", func(t *testing.T) {
		convertedErr := errors.New("converted")
		systemErr := errors.New("system")
		err := errors.Join(WrapAsValidationIssue(convertedErr), systemErr)

		issues, extractionErr := ToValidationIssues(err)
		require.Nil(t, issues)
		require.Equal(t, err, extractionErr)
		require.False(t, IsValidationIssueOnly(err))
	})

	t.Run("converts ordinary errors with warning severity", func(t *testing.T) {
		ordinaryErr := errors.New("provider cleanup skipped")
		criticalIssue := NewValidationError("invalid_invoice", "invoice is invalid")
		err := WrapAsValidationIssue(errors.Join(ordinaryErr, criticalIssue), WithWarningSeverity())

		issues, extractionErr := ToValidationIssues(err)
		require.NoError(t, extractionErr)
		require.Equal(t, ValidationIssues{
			{
				Severity: ValidationIssueSeverityWarning,
				Message:  ordinaryErr.Error(),
			},
			criticalIssue,
		}, issues)
	})

	t.Run("outermost conversion controls ordinary error severity", func(t *testing.T) {
		ordinaryErr := errors.New("provider cleanup skipped")
		err := WrapAsValidationIssue(WrapAsValidationIssue(ordinaryErr), WithWarningSeverity())

		issues, extractionErr := ToValidationIssues(err)
		require.NoError(t, extractionErr)
		require.Equal(t, ValidationIssues{{
			Severity: ValidationIssueSeverityWarning,
			Message:  ordinaryErr.Error(),
		}}, issues)
	})

	t.Run("returns nil for a nil error", func(t *testing.T) {
		require.NoError(t, WrapAsValidationIssue(nil))
		require.NoError(t, WrapAsValidationIssue(nil, WithWarningSeverity()))
	})
}

func TestValidationIssueRecorder(t *testing.T) {
	recorder := ValidationIssueRecorder{}
	warning := ValidationIssue{
		Severity:  ValidationIssueSeverityWarning,
		Message:   "rating warning",
		Code:      "rating_warning",
		Component: "original-component",
		Path:      "amount",
		Attributes: models.Annotations{
			"source": "issue",
			"keep":   true,
		},
	}

	require.NoError(t, recorder.Record(
		warning,
		WithAttributes(models.Annotations{
			AttributeKeyLineID: "line-1",
			"source":           "recorder",
		}),
		WithComponent("line-engine"),
		WithPath("lines/line-1"),
	))

	systemErr := errors.New("rating service unavailable")
	err := recorder.Record(
		systemErr,
		WithAttributes(models.Annotations{AttributeKeyLineID: "line-2"}),
		WithComponent("line-engine"),
		WithPath("lines/line-2"),
	)
	require.ErrorIs(t, err, systemErr)
	require.EqualError(t, err, "lines/line-2: line-engine: rating service unavailable [line_id=line-2]")
	require.False(t, IsValidationIssueOnly(err))

	issues, extractionErr := ToValidationIssues(recorder.ErrorsOrNil())
	require.NoError(t, extractionErr)
	require.Equal(t, ValidationIssues{{
		Severity:  warning.Severity,
		Message:   warning.Message,
		Code:      warning.Code,
		Component: "line-engine",
		Path:      "/lines/line-1/amount",
		Attributes: models.Annotations{
			AttributeKeyLineID: "line-1",
			"source":           "recorder",
			"keep":             true,
		},
	}}, issues)
}

func TestValidationIssueRecorderRecordWarnings(t *testing.T) {
	warning := NewValidationWarning("rating_warning", "rating warning")
	critical := NewValidationError("rating_blocked", "rating blocked")
	systemErr := errors.New("rating unavailable")

	recorder := ValidationIssueRecorder{}
	require.NoError(t, recorder.RecordWarnings(nil))
	require.NoError(t, recorder.RecordWarnings(errors.Join(warning, warning), WithComponent("rating")))

	for _, test := range []struct {
		name string
		err  error
	}{
		{name: "critical validation issue", err: errors.Join(critical)},
		{name: "mixed validation issues", err: errors.Join(warning, critical)},
		{name: "system error", err: systemErr},
		{name: "warning joined with system error", err: errors.Join(warning, systemErr)},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := recorder.RecordWarnings(test.err, WithComponent("rating"))
			require.Same(t, test.err, err)
		})
	}

	issues, err := ToValidationIssues(recorder.ErrorsOrNil())
	require.NoError(t, err)
	require.Len(t, issues, 2)
	for _, issue := range issues {
		require.Equal(t, warning.Code, issue.Code)
		require.Equal(t, ValidationIssueSeverityWarning, issue.Severity)
		require.Equal(t, ComponentName("rating"), issue.Component)
	}
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

func TestValidationWithMessage(t *testing.T) {
	baseIssue := ValidationIssue{
		Severity:  ValidationIssueSeverityWarning,
		Message:   "canonical message",
		Code:      "canonical_code",
		Component: "original-component",
		Path:      "original/path",
	}

	t.Run("adds guidance and attributes while preserving error identity", func(t *testing.T) {
		err := ValidationWithMessage(
			baseIssue,
			"resolve the feature configuration",
			models.Attributes{"feature": "requests"},
		)

		require.EqualError(t, err, "resolve the feature configuration: canonical message [feature=requests]")
		require.ErrorIs(t, err, baseIssue)

		issues, systemErr := ToValidationIssues(err)
		require.NoError(t, systemErr)
		require.Equal(t, ValidationIssues{
			{
				Severity:  baseIssue.Severity,
				Message:   "resolve the feature configuration: canonical message",
				Code:      baseIssue.Code,
				Component: baseIssue.Component,
				Path:      "/original/path",
				Attributes: models.Annotations{
					"feature": "requests",
				},
			},
		}, issues)
	})

	t.Run("composes nested message component and field context", func(t *testing.T) {
		err := ValidationWithMessage(
			ValidationWithComponent(
				"outer-component",
				ValidationWithFieldPrefix(
					"lines/line-1",
					ValidationWithMessage(baseIssue, "resolve the inner issue"),
				),
			),
			"resolve the outer issue",
		)

		issues, systemErr := ToValidationIssues(err)
		require.NoError(t, systemErr)
		require.Equal(t, ValidationIssues{
			{
				Severity:  baseIssue.Severity,
				Message:   "resolve the outer issue: resolve the inner issue: canonical message",
				Code:      baseIssue.Code,
				Component: "outer-component",
				Path:      "/lines/line-1/original/path",
			},
		}, issues)
	})

	t.Run("prefixes every validation issue in a joined error", func(t *testing.T) {
		secondIssue := NewValidationError("second_code", "second message")
		err := ValidationWithMessage(errors.Join(baseIssue, secondIssue), "resolve the shared issue")

		issues, systemErr := ToValidationIssues(err)
		require.NoError(t, systemErr)
		require.Equal(t, ValidationIssues{
			{
				Severity:  baseIssue.Severity,
				Message:   "resolve the shared issue: canonical message",
				Code:      baseIssue.Code,
				Component: baseIssue.Component,
				Path:      "/original/path",
			},
			{
				Severity: secondIssue.Severity,
				Message:  "resolve the shared issue: second message",
				Code:     secondIssue.Code,
			},
		}, issues)
	})

	t.Run("does not promote a system error", func(t *testing.T) {
		systemErr := errors.New("database unavailable")
		err := ValidationWithMessage(
			systemErr,
			"retry after the database recovers",
			models.Attributes{"invoice_id": "invoice-1"},
		)

		require.EqualError(t, err, "retry after the database recovers: database unavailable [invoice_id=invoice-1]")
		require.ErrorIs(t, err, systemErr)

		issues, extractionErr := ToValidationIssues(err)
		require.Nil(t, issues)
		require.Equal(t, err, extractionErr)
	})

	t.Run("merges attribute maps with later precedence", func(t *testing.T) {
		err := ValidationWithMessage(
			baseIssue,
			"resolve the conflicting values",
			models.Attributes{"first": 1, "precedence": "first"},
			models.Attributes{"second": 2, "precedence": "second"},
		)

		require.EqualError(t, err, "resolve the conflicting values: canonical message [first=1,precedence=second,second=2]")

		issues, systemErr := ToValidationIssues(err)
		require.NoError(t, systemErr)
		require.Equal(t, models.Annotations{
			"first":      1,
			"precedence": "second",
			"second":     2,
		}, issues[0].Attributes)
	})

	t.Run("returns nil for a nil error", func(t *testing.T) {
		require.NoError(t, ValidationWithMessage(nil, "unused", models.Attributes{"unused": true}))
	})
}
