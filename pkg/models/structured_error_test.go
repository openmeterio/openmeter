package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStructuredErrorFluentMethodsPreserveKind(t *testing.T) {
	var (
		system     Error
		validation ValidationIssue
	)

	// These assignments also check the return type of every fluent method.
	system = NewError("ledger_failed", "accounting failed").
		Clone().With(WithComponent("ledger")).WithField(NewFieldSelector("amount")).
		WithPathString("source", "amount").WithSeverity(ErrorSeverityCritical).
		WithAttr("reason", "missing_source").WithAttrs(Attributes{"amount": 10}).
		SetAttributes(Attributes{"reason": "missing_source"})
	validation = NewValidationIssue("invalid_amount", "amount is invalid").
		Clone().With(WithComponent("ledger")).WithField(NewFieldSelector("amount")).
		WithPathString("source", "amount").WithSeverity(ErrorSeverityWarning).
		WithAttr("reason", "negative_amount").WithAttrs(Attributes{"amount": -10}).
		SetAttributes(Attributes{"reason": "negative_amount"})

	var issue ValidationIssue
	require.False(t, errors.As(system, &issue))
	require.ErrorAs(t, fmt.Errorf("validate: %w", validation), &issue)
	require.Equal(t, "negative_amount", issue.Attributes()["reason"])
	require.Equal(t, ErrorSeverityWarning, issue.Severity())

	_, err := AsValidationIssues(system)
	require.ErrorIs(t, err, system)
	issues, err := AsValidationIssues(validation)
	require.NoError(t, err)
	require.Equal(t, "invalid_amount", string(issues[0].Code()))
}

func TestStructuredErrorIdentityAndCause(t *testing.T) {
	cause := errors.New("storage unavailable")
	base := NewError("ledger_failed", "cannot load original entry", WithCause(cause))
	err := base.WithAttr("reason", "source_missing").With(WithComponent("ledger"))

	require.ErrorIs(t, err, base)
	require.ErrorIs(t, fmt.Errorf("collect: %w", err), cause)
	require.Same(t, cause, errors.Unwrap(err))
	require.Same(t, cause, err.Cause())
	require.Empty(t, base.Attributes())
	require.Equal(t, base.Error(), err.Error())

	// Replacing a cause must not leave the old one reachable through fluent copies.
	replacement := errors.New("another failure")
	changed := err.With(WithCause(replacement))
	require.ErrorIs(t, changed, base)
	require.ErrorIs(t, changed, replacement)
	require.NotErrorIs(t, changed, cause)
	require.NoError(t, errors.Unwrap(changed.With(WithCause(nil))))

	validation := NewValidationIssue(base.Code(), base.Message())
	require.NotErrorIs(t, base, validation)
	require.NotErrorIs(t, validation, base)
}

func TestStructuredErrorDoesNotBecomeValidation(t *testing.T) {
	issue := NewValidationIssue("invalid_amount", "amount is invalid")
	system := NewError("ledger_failed", "accounting failed", WithCause(issue)).WithAttr("reason", "missing_source")

	for _, tc := range []struct {
		name string
		err  error
	}{
		{name: "value", err: system},
		{name: "pointer", err: &system},
		{name: "formatted wrapper", err: fmt.Errorf("collect: %w", system)},
		{name: "component wrapper", err: ErrorWithComponent("ledger", system)},
		{name: "field wrapper", err: ErrorWithFieldPrefix(NewFieldSelector("amount"), system)},
		{name: "joined with validation", err: errors.Join(issue, system)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			issues, err := AsValidationIssues(tc.err)
			require.ErrorIs(t, err, system)
			require.Empty(t, issues)
		})
	}

	// Error inspection can still reach the validation cause without classifying
	// the enclosing system failure as a validation result.
	var cause ValidationIssue
	require.ErrorAs(t, system, &cause)
	require.Equal(t, issue, cause)
}

func TestStructuredErrorJSONAndAttributeIsolation(t *testing.T) {
	err := NewError("ledger_failed", "accounting failed", WithComponent("ledger"), WithCause(errors.New("private cause"))).
		WithAttr("reason", "missing_source")
	attrs := err.Attributes()
	attrs["reason"] = "changed"
	require.Equal(t, "missing_source", err.Attributes()["reason"])

	encoded, marshalErr := json.Marshal(err)
	require.NoError(t, marshalErr)
	require.JSONEq(t, `{"code":"ledger_failed","message":"accounting failed","component":"ledger","severity":"critical","reason":"missing_source"}`, string(encoded))
	require.NotContains(t, string(encoded), "private cause")
}
