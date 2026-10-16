package models

import (
	"encoding/json"
	"errors"

	"github.com/samber/lo"
)

var (
	_ error                    = ValidationIssue{}
	_ json.Marshaler           = ValidationIssue{}
	_ Equaler[ValidationIssue] = ValidationIssue{}
)

// ValidationIssue is a structured error that callers may handle as domain validation.
type ValidationIssue = structuredError[validationIssueKind]

type validationIssueKind struct{}

// ValidationIssueOption keeps existing option declarations compatible.
type ValidationIssueOption = ErrorOption

// NewValidationIssue creates a structured domain validation issue.
func NewValidationIssue(code ErrorCode, message string, opts ...ValidationIssueOption) ValidationIssue {
	return newValidationIssue(append([]ErrorOption{withCode(code), withMessage(message)}, opts...)...)
}

func newValidationIssue(opts ...ValidationIssueOption) ValidationIssue {
	return newStructuredError[validationIssueKind](opts...)
}

// NewValidationError returns a new ValidationIssue with code and message and its severity set to SeverityCritical.
func NewValidationError(code ErrorCode, message string) ValidationIssue {
	return NewValidationIssue(code, message, WithCriticalSeverity())
}

// NewValidationWarning returns a new ValidationIssue with code and message and its severity set to SeverityWarning.
func NewValidationWarning(code ErrorCode, message string) ValidationIssue {
	return NewValidationIssue(code, message, WithWarningSeverity())
}

type ValidationIssues []ValidationIssue

func (v ValidationIssues) Clone() ValidationIssues {
	return append(make(ValidationIssues, 0, len(v)), v...)
}

func (v ValidationIssues) AsError() error {
	if len(v) == 0 {
		return nil
	}

	return errors.Join(lo.Map(v, func(issue ValidationIssue, _ int) error {
		return issue
	})...)
}

func (v ValidationIssues) AsErrorExtensions() []ErrorExtension {
	return lo.Map(v, func(issue ValidationIssue, _ int) ErrorExtension {
		return issue.AsErrorExtension()
	})
}

func (v ValidationIssues) Error() string {
	return v.AsError().Error()
}

func (v ValidationIssues) WithSeverityOrHigher(severity ErrorSeverity) ValidationIssues {
	// NOTE: lower numeric values correspond to more severe errors.
	return lo.Filter(v, func(issue ValidationIssue, _ int) bool {
		return issue.Severity() <= severity
	})
}

// AsValidationIssues extracts issues through wrappers and joins. If any branch
// contains a system error, it returns the original error tree. Legacy field and
// component wrappers still convert plain errors; a structured Error stays a failure.
func AsValidationIssues(errIn error) (ValidationIssues, error) {
	if errIn == nil {
		return nil, nil
	}

	issues, err := asValidationIssues(errIn, NewFieldSelectorGroup(), "", false)
	if err != nil {
		return nil, errIn
	}

	return issues, nil
}

func asValidationIssues(err error, prefix *FieldDescriptor, component ComponentName, unknownAsValidationIssue bool) (ValidationIssues, error) {
	if err == nil {
		return nil, nil
	}

	switch e := err.(type) {
	case componentWrapper:
		return asValidationIssues(e.err, prefix, e.component, true)
	case fieldPrefixedWrapper:
		return asValidationIssues(e.err, e.prefix.WithPrefix(prefix), component, true)
	case Error:
		// A structured system error is a failure even if its cause is a validation issue.
		return nil, err
	case *Error:
		if e == nil {
			return nil, nil
		}

		return nil, err
	case ValidationIssue:
		return validationIssueAsValidationIssues(e, prefix, component), nil
	case *ValidationIssue:
		if e == nil {
			return nil, nil
		}

		return validationIssueAsValidationIssues(*e, prefix, component), nil
	}

	switch e := err.(type) {
	case interface{ Unwrap() []error }:
		issues := ValidationIssues{}

		for _, unwrapped := range e.Unwrap() {
			var items ValidationIssues

			items, err = asValidationIssues(unwrapped, prefix, component, unknownAsValidationIssue)
			if err != nil {
				return nil, err
			}

			if len(items) > 0 {
				issues = append(issues, items...)
			}
		}

		return issues, nil
	case interface{ Unwrap() error }:
		return asValidationIssues(e.Unwrap(), prefix, component, unknownAsValidationIssue)
	default:
		if unknownAsValidationIssue {
			return ValidationIssues{
				newValidationIssue(
					withMessage(err.Error()),
					WithField(prefix),
					WithSeverity(ErrorSeverityCritical),
					WithComponent(component),
				),
			}, nil
		}

		return nil, err
	}
}

func validationIssueAsValidationIssues(e ValidationIssue, prefix *FieldDescriptor, component ComponentName) ValidationIssues {
	opts := []ValidationIssueOption{
		withMessage(e.message),
		withCode(e.code),
		WithComponent(func() ComponentName {
			if component == "" {
				return e.component
			}

			return component
		}()),
		WithSeverity(e.severity),
		WithAttributes(e.Attributes()),
	}

	if e.field != nil {
		opts = append(opts, WithField(e.field.WithPrefix(prefix)))
	}

	return ValidationIssues{
		newValidationIssue(opts...),
	}
}

func EncodeValidationIssues[T error](err T) map[string]interface{} {
	validationIssues, _ := AsValidationIssues(err)

	if len(validationIssues) == 0 {
		return map[string]interface{}{}
	}

	var issues []map[string]interface{}
	for _, issue := range validationIssues {
		issues = append(issues, issue.AsErrorExtension())
	}

	return map[string]interface{}{
		"validationErrors": issues,
	}
}
