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
			expectedJSON: `{"message":""}`,
		},
		{
			name: "bare",
			issue: ValidationIssue{
				Message: "error message",
			},
			expectedJSON: `{"message":"error message"}`,
		},
		{
			name: "generic",
			issue: ValidationIssue{
				Message:   "error message",
				Path:      "field_name",
				Code:      "invalid_param",
				Component: "openmeter",
				Severity:  ErrorSeverityCritical,
			},
			expectedJSON: `{
				"code":"invalid_param",
				"component":"openmeter",
				"message":"error message",
				"path":"field_name",
				"severity":"critical"
			}`,
		},
		{
			name: "with attributes",
			issue: ValidationIssue{
				Message:   "error message",
				Path:      "field_name",
				Code:      "invalid_param",
				Component: "openmeter",
				Severity:  ErrorSeverityWarning,
				Attributes: map[string]interface{}{
					"severity": "critical",
				},
			},
			expectedJSON: `{
				"code":"invalid_param",
				"component":"openmeter",
				"message":"error message",
				"path":"field_name",
				"severity":"warning",
				"attributes":{
					"severity":"critical"
				}
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
	Code:     "test_validation_issue",
	Message:  "test validation issue",
	Path:     "field",
	Severity: ErrorSeverityCritical,
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

			assert.Emptyf(t, errTestValidationIssue.Attributes, "original attributes must be empty")
			assert.Equalf(t, test.attrs, actual.Attributes, "attributes must match")
		})
	}
}

func TestValidationIssue_WithPath(t *testing.T) {
	tests := []struct {
		name  string
		issue ValidationIssue
		path  string
	}{
		{
			name:  "empty",
			issue: errTestValidationIssue,
			path:  "",
		},
		{
			name:  "new",
			issue: errTestValidationIssue,
			path:  "field2",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := test.issue.WithPath(test.path)

			assert.Equalf(t, "field", errTestValidationIssue.Path, "path not must be overwritten in the source error")
			assert.Equalf(t, test.path, actual.Path, "path must match")
		})
	}
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
				Attributes: errTestValidationIssue.Attributes,
				Code:       errTestValidationIssue.Code,
				Component:  errTestValidationIssue.Component,
				Message:    errTestValidationIssue.Message,
				Path:       errTestValidationIssue.Path,
				Severity:   errTestValidationIssue.Severity,
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
					Component: "component1",
					Message:   "error 1",
					Severity:  ErrorSeverityCritical,
				},
				ValidationIssue{
					Code:     "errcode1",
					Message:  "error message 1",
					Severity: ErrorSeverityCritical,
				},
				ValidationIssue{
					Code:     "errcode2",
					Message:  "error message 2",
					Severity: ErrorSeverityWarning,
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
						ErrorWithFieldPrefix("prefix",
							errors.Join(
								NewValidationIssue("errcode1", "error message 1",
									WithPath("field1"),
									WithCriticalSeverity(),
								),
								NewValidationIssue("errcode2", "error message 2",
									WithPath("field2"),
									WithWarningSeverity(),
								),
								ErrorWithFieldPrefix("prefix2",
									ErrorWithComponent("component3",
										errors.Join(
											NewValidationIssue("errcode3", "error message 3",
												WithPath("field3"),
												WithCriticalSeverity(),
											),
											NewValidationIssue("errcode4", "error message 4",
												WithPath("prefix3", "field4"),
												WithWarningSeverity(),
												WithAttributes(map[string]interface{}{
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
						WithPath("field6"),
						WithCriticalSeverity(),
						WithComponent("component4"),
					),
				),
			),
			expectError: false,
			expectedIssues: ValidationIssues{
				ValidationIssue{
					Component: "component1",
					Message:   "error 1",
					Severity:  ErrorSeverityCritical,
				},
				ValidationIssue{
					Code:      "errcode1",
					Component: "component2",
					Message:   "error message 1",
					Path:      "/prefix/field1",
					Severity:  ErrorSeverityCritical,
				},
				ValidationIssue{
					Code:      "errcode2",
					Component: "component2",
					Message:   "error message 2",
					Path:      "/prefix/field2",
					Severity:  ErrorSeverityWarning,
				},
				ValidationIssue{
					Code:      "errcode3",
					Component: "component3",
					Message:   "error message 3",
					Path:      "/prefix/prefix2/field3",
					Severity:  ErrorSeverityCritical,
				},
				ValidationIssue{
					Code:      "errcode4",
					Component: "component3",
					Message:   "error message 4",
					Path:      "/prefix/prefix2/prefix3/field4",
					Severity:  ErrorSeverityWarning,
					Attributes: map[string]interface{}{
						"attr1": "value1",
					},
				},
				ValidationIssue{
					Code:      "errcode5",
					Component: "component4",
					Message:   "error message 5",
					Path:      "/field6",
					Severity:  ErrorSeverityCritical,
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

			require.Equalf(t, test.expectedIssues, actual, "issues must match")
		})
	}
}

func TestValidationIssues_AsError(t *testing.T) {
	issues := ValidationIssues{
		{
			Severity:  ErrorSeverityCritical,
			Message:   "error1",
			Component: "component1",
			Path:      "/some/path/from/component1",
		},
	}

	err := issues.AsError()

	validationIssues, err := AsValidationIssues(err)
	require.NoError(t, err)
	require.Equal(t, issues, validationIssues)
}
