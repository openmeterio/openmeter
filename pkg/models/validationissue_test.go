package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidationIssue_JSON(t *testing.T) {
	tests := []struct {
		name         string
		issue        ValidationIssue
		expectedJSON string
	}{
		{
			name:         "empty",
			issue:        ValidationIssue{},
			expectedJSON: `{"message":"", "severity":"critical"}`,
		},
		{
			name: "bare",
			issue: ValidationIssue{
				message: "error message",
			},
			expectedJSON: `{"message":"error message", "severity":"critical"}`,
		},
		{
			name: "generic",
			issue: ValidationIssue{
				message:   "error message",
				field:     NewFieldSelectorGroup(NewFieldSelector("field_name")),
				code:      "invalid_param",
				component: "openmeter",
				severity:  ErrorSeverityCritical,
			},
			expectedJSON: `{
				"code":"invalid_param",
				"component":"openmeter",
				"message":"error message",
				"field":"$.field_name",
				"severity":"critical"
			}`,
		},
		{
			name: "with attributes",
			issue: ValidationIssue{
				message: "error message",
				field: NewFieldSelectorGroup(NewFieldSelector("field_name").WithExpression(
					NewFieldAttrValue("key", "value"),
				)),
				code:      "invalid_param",
				component: "openmeter",
				severity:  ErrorSeverityWarning,
				attributes: Attributes{
					"attr1": "value1",
				},
			},
			expectedJSON: `{
				"code":"invalid_param",
				"component":"openmeter",
				"message":"error message",
				"field":"$.field_name[?(@.key=='value')]",
				"severity":"warning",
				"attr1": "value1"
			}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			b, err := json.Marshal(test.issue)
			require.NoErrorf(t, err, "marshaling ValidationIssue to JSON must not fail")

			assert.JSONEqf(t, test.expectedJSON, string(b), "JSON representation must match")
		})
	}
}

var errTestValidationIssue = ValidationIssue{
	code:     "test_validation_issue",
	message:  "test validation issue",
	field:    NewFieldSelectorGroup(NewFieldSelector("field")),
	severity: ErrorSeverityCritical,
}

func TestValidationIssue_WithAttrs(t *testing.T) {
	tests := []struct {
		name  string
		issue ValidationIssue
		attrs Attributes
	}{
		{
			name:  "empty",
			issue: errTestValidationIssue,
			attrs: nil,
		},
		{
			name:  "with attributes",
			issue: errTestValidationIssue,
			attrs: Attributes{
				"key1": "value1",
				"key2": "value2",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := test.issue.WithAttrs(test.attrs)

			assert.Emptyf(t, errTestValidationIssue.Attributes(), "original attributes must be empty")
			assert.Equalf(t, test.attrs, actual.Attributes(), "attributes must match")
		})
	}
}

func TestValidationIssue_WithField(t *testing.T) {
	t.Run("Should UNSET field if nil is passed", func(t *testing.T) {
		issue := errTestValidationIssue.Clone()

		withField := issue.WithField()

		assert.Equal(t, "", withField.Field().String(), "field must match")
		assert.Equal(t, "", withField.Field().JSONPath(), "field path must match")
	})

	t.Run("Should override field if non-nil path is passed", func(t *testing.T) {
		issue := errTestValidationIssue.Clone()

		desc := NewFieldSelectorGroup(NewFieldSelector("field_name"))

		withField := issue.WithField(desc)

		assert.Equal(t, desc.String(), withField.Field().String(), "field must match")
		assert.Equal(t, "$.field_name", withField.Field().JSONPath(), "field path must match")
	})
}

func TestValidationIssue_Clone(t *testing.T) {
	tests := []struct {
		name          string
		issue         ValidationIssue
		expectedIssue ValidationIssue
	}{
		{
			name:  "clone",
			issue: errTestValidationIssue,
			expectedIssue: ValidationIssue{
				attributes: errTestValidationIssue.attributes,
				code:       errTestValidationIssue.code,
				component:  errTestValidationIssue.component,
				message:    errTestValidationIssue.message,
				field:      errTestValidationIssue.field,
				severity:   errTestValidationIssue.severity,
			},
		},
		{
			name:          "empty",
			issue:         ValidationIssue{},
			expectedIssue: ValidationIssue{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := test.issue.Clone()

			assert.Equalf(t, test.expectedIssue, actual, "must match after clone")
		})
	}
}

func TestAsValidationIssues(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectError    bool
		expectedIssues ValidationIssues
	}{
		{
			name:           "nil",
			err:            nil,
			expectError:    false,
			expectedIssues: nil,
		},
		{
			name:           "error",
			err:            errors.New("error"),
			expectError:    true,
			expectedIssues: nil,
		},
		{
			name:           "error/wrapped",
			err:            fmt.Errorf("test: %w", errors.New("error")),
			expectError:    true,
			expectedIssues: nil,
		},
		{
			name: "mixed",
			err: fmt.Errorf("test: %w",
				errors.Join(
					errors.New("error 1"),
					NewValidationError("errcode1", "error message 1"),
					NewValidationWarning("errcode2", "error message 2"),
				),
			),
			expectError:    true,
			expectedIssues: nil,
		},
		{
			name: "mixed",
			err: fmt.Errorf("test: %w",
				errors.Join(
					errors.New("error 1"),
					NewValidationError("errcode1", "error message 1"),
					NewValidationWarning("errcode2", "error message 2"),
				),
			),
			expectError:    true,
			expectedIssues: nil,
		},
		{
			name: "unknown wrapped",
			err: fmt.Errorf("test: %w",
				errors.Join(
					ErrorWithComponent("component1",
						errors.New("error 1"),
					),
					NewValidationError("errcode1", "error message 1"),
					NewValidationWarning("errcode2", "error message 2"),
				),
			),
			expectError: false,
			expectedIssues: ValidationIssues{
				ValidationIssue{
					component: "component1",
					message:   "error 1",
					severity:  ErrorSeverityCritical,
				},
				ValidationIssue{
					code:     "errcode1",
					message:  "error message 1",
					severity: ErrorSeverityCritical,
				},
				ValidationIssue{
					code:     "errcode2",
					message:  "error message 2",
					severity: ErrorSeverityWarning,
				},
			},
		},
		{
			name: "nested and wrapped with unknown",
			err: fmt.Errorf("test: %w",
				errors.Join(
					ErrorWithComponent("component1",
						errors.New("error 1"),
					),
					ErrorWithComponent("component2",
						ErrorWithFieldPrefix(NewFieldSelectorGroup(NewFieldSelector("prefix")),
							errors.Join(
								NewValidationIssue("errcode1", "error message 1",
									WithFieldString("field1"),
									WithCriticalSeverity(),
								),
								NewValidationIssue("errcode2", "error message 2",
									WithFieldString("field2"),
									WithWarningSeverity(),
								),
								ErrorWithFieldPrefix(NewFieldSelectorGroup(NewFieldSelector("prefix2")),
									ErrorWithComponent("component3",
										errors.Join(
											NewValidationIssue("errcode3", "error message 3",
												WithFieldString("field3"),
												WithCriticalSeverity(),
											),
											NewValidationIssue("errcode4", "error message 4",
												WithField(
													NewFieldSelector("prefix3"),
													NewFieldSelector("field4"),
												),
												WithWarningSeverity(),
												WithAttributes(Attributes{
													"attr1": "value1",
												}),
											),
										),
									),
								),
							),
						),
					),
					NewValidationIssue("errcode5", "error message 5",
						WithField(NewFieldSelector("field5")),
						WithCriticalSeverity(),
						WithComponent("component4"),
					),
				),
			),
			expectError: false,
			expectedIssues: ValidationIssues{
				ValidationIssue{
					component: "component1",
					message:   "error 1",
					severity:  ErrorSeverityCritical,
				},
				ValidationIssue{
					code:      "errcode1",
					component: "component2",
					message:   "error message 1",
					field: NewFieldSelectorGroup(
						NewFieldSelector("prefix"),
						NewFieldSelector("field1"),
					),
					severity: ErrorSeverityCritical,
				},
				ValidationIssue{
					code:      "errcode2",
					component: "component2",
					message:   "error message 2",
					field: NewFieldSelectorGroup(
						NewFieldSelector("prefix"),
						NewFieldSelector("field2"),
					),
					severity: ErrorSeverityWarning,
				},
				ValidationIssue{
					code:      "errcode3",
					component: "component3",
					message:   "error message 3",
					field: NewFieldSelectorGroup(
						NewFieldSelector("prefix"),
						NewFieldSelector("prefix2"),
						NewFieldSelector("field3"),
					),
					severity: ErrorSeverityCritical,
				},
				ValidationIssue{
					code:      "errcode4",
					component: "component3",
					message:   "error message 4",
					field: NewFieldSelectorGroup(
						NewFieldSelector("prefix"),
						NewFieldSelector("prefix2"),
						NewFieldSelector("prefix3"),
						NewFieldSelector("field4"),
					),
					severity: ErrorSeverityWarning,
					attributes: Attributes{
						"attr1": "value1",
					},
				},
				ValidationIssue{
					code:      "errcode5",
					component: "component4",
					message:   "error message 5",
					field: NewFieldSelectorGroup(
						NewFieldSelector("field5"),
					),
					severity: ErrorSeverityCritical,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := AsValidationIssues(test.err)

			if test.expectError {
				require.EqualErrorf(t, err, test.err.Error(), "error must match")
			} else {
				require.NoError(t, err, "error must be nil")
			}

			RequireValidationIssuesMatch(t, test.expectedIssues, actual)
		})
	}
}

func TestValidationIssues_AsError(t *testing.T) {
	issues := ValidationIssues{
		{
			severity:  ErrorSeverityCritical,
			message:   "error1",
			component: "component1",
			field:     NewFieldSelectorGroup(NewFieldSelector("field1")),
		},
	}

	err := issues.AsError()

	validationIssues, err := AsValidationIssues(err)
	require.NoError(t, err)
	RequireValidationIssuesMatch(t, issues, validationIssues)
}
