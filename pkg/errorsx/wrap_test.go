package errorsx

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// Custom error types for testing error chain preservation
type ValidationError struct {
	Field string
	Msg   string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error on field %s: %s", e.Field, e.Msg)
}

type SomeError struct {
	Code string
	Msg  string
}

func (e SomeError) Error() string {
	return fmt.Sprintf("some error %s: %s", e.Code, e.Msg)
}

func TestFormatJoinedError(t *testing.T) {
	t.Run("basic functionality", func(t *testing.T) {
		tests := []struct {
			name     string
			input    error
			expected string
			wantNil  bool
		}{
			{
				name:    "nil error",
				input:   nil,
				wantNil: true,
			},
			{
				name:     "single error",
				input:    fmt.Errorf("single error"),
				expected: "single error",
			},
			{
				name:     "single custom error",
				input:    ValidationError{Field: "name", Msg: "required"},
				expected: "validation error on field name: required",
			},
			{
				name:     "joined error with single error",
				input:    errors.Join(fmt.Errorf("only error")),
				expected: "only error",
			},
			{
				name: "joined error with two errors",
				input: errors.Join(
					fmt.Errorf("first error"),
					fmt.Errorf("second error"),
				),
				expected: "multiple errors: first error; second error",
			},
			{
				name: "joined error with three errors",
				input: errors.Join(
					fmt.Errorf("first error"),
					fmt.Errorf("second error"),
					fmt.Errorf("third error"),
				),
				expected: "multiple errors: first error; second error; third error",
			},
			{
				name: "joined error with custom error types",
				input: errors.Join(
					ValidationError{Field: "email", Msg: "invalid format"},
					SomeError{Code: "BIZ001", Msg: "user already exists"},
					fmt.Errorf("system error"),
				),
				expected: "multiple errors: validation error on field email: invalid format; some error BIZ001: user already exists; system error",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := FormatJoinedError(tt.input)

				if tt.wantNil {
					if result != nil {
						t.Errorf("FormatJoinedError() = %v, want nil", result)
					}
					return
				}

				if result == nil {
					t.Fatal("FormatJoinedError() returned nil, expected an error")
				}

				if result.Error() != tt.expected {
					t.Errorf("FormatJoinedError() = %q, want %q", result.Error(), tt.expected)
				}
			})
		}
	})

	t.Run("error chain preservation", func(t *testing.T) {
		originalErr := ValidationError{Field: "username", Msg: "too short"}
		secondErr := SomeError{Code: "BIZ002", Msg: "rate limited"}
		thirdErr := fmt.Errorf("network error")

		joinedErr := errors.Join(originalErr, secondErr, thirdErr)
		result := FormatJoinedError(joinedErr)

		// Test that we can still find the original error using errors.Is
		if !errors.Is(result, originalErr) {
			t.Errorf("errors.Is(result, originalErr) = false, want true")
		}

		// Test that we can still extract the original error using errors.As
		var validationErr ValidationError
		if !errors.As(result, &validationErr) {
			t.Errorf("errors.As(result, &validationErr) = false, want true")
		} else {
			if validationErr.Field != "username" || validationErr.Msg != "too short" {
				t.Errorf("extracted ValidationError = %+v, want {Field: username, Msg: too short}", validationErr)
			}
		}

		// Test that we cannot find the second error (since it's not wrapped)
		if errors.Is(result, secondErr) {
			t.Errorf("errors.Is(result, secondErr) = true, want false (second error should not be in chain)")
		}

		var businessErr SomeError
		if errors.As(result, &businessErr) {
			t.Errorf("errors.As(result, &businessErr) = true, want false (second error should not be in chain)")
		}
	})

	t.Run("non-joined errors unchanged", func(t *testing.T) {
		tests := []struct {
			name  string
			input error
		}{
			{
				name:  "simple error",
				input: fmt.Errorf("simple error"),
			},
			{
				name:  "wrapped error",
				input: fmt.Errorf("wrapped: %w", fmt.Errorf("inner error")),
			},
			{
				name:  "custom error",
				input: ValidationError{Field: "test", Msg: "invalid"},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := FormatJoinedError(tt.input)

				// For non-joined errors, the function should return the same error
				if result != tt.input {
					t.Errorf("FormatJoinedError() should return the same error for non-joined errors, got different error")
				}

				if result.Error() != tt.input.Error() {
					t.Errorf("FormatJoinedError() = %q, want %q", result.Error(), tt.input.Error())
				}
			})
		}
	})

	t.Run("message format", func(t *testing.T) {
		err1 := fmt.Errorf("error one")
		err2 := fmt.Errorf("error two")
		err3 := fmt.Errorf("error three")

		joinedErr := errors.Join(err1, err2, err3)
		result := FormatJoinedError(joinedErr)

		resultMsg := result.Error()

		// Check that it starts with "multiple errors:"
		if !strings.HasPrefix(resultMsg, "multiple errors:") {
			t.Errorf("result should start with 'multiple errors:', got: %s", resultMsg)
		}

		// Check that all error messages are present
		expectedParts := []string{"error one", "error two", "error three"}
		for _, part := range expectedParts {
			if !strings.Contains(resultMsg, part) {
				t.Errorf("result should contain %q, got: %s", part, resultMsg)
			}
		}

		// Check the exact format
		expected := "multiple errors: error one; error two; error three"
		if resultMsg != expected {
			t.Errorf("FormatJoinedError() = %q, want %q", resultMsg, expected)
		}
	})
}
