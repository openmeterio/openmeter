package models

import (
	"errors"

	"github.com/samber/lo"
)

type Attributes map[string]any

func (a Attributes) Clone() Attributes {
	if a == nil {
		return nil
	}

	m := make(Attributes)

	if len(a) == 0 {
		return m
	}

	for k, v := range a {
		m[k] = v
	}

	return m
}

func (a Attributes) Merge(m Attributes) Attributes {
	if len(m) == 0 {
		return a.Clone()
	}

	r := make(Attributes, len(a)+len(m))

	for k, v := range a {
		r[k] = v
	}

	for k, v := range m {
		r[k] = v
	}

	return r
}

type ErrorExtensions map[string]any

var _ error = (*ValidationIssue)(nil)

type ValidationIssue struct {
	Attributes Attributes    `json:"attributes,omitempty"`
	Code       ErrorCode     `json:"code,omitempty"`
	Component  ComponentName `json:"component,omitempty"`
	Message    string        `json:"message"`
	Path       string        `json:"path,omitempty"`
	Severity   ErrorSeverity `json:"severity,omitempty"`
}

func (i ValidationIssue) Clone() ValidationIssue {
	return ValidationIssue{
		Attributes: i.Attributes.Clone(),
		Code:       i.Code,
		Component:  i.Component,
		Message:    i.Message,
		Path:       i.Path,
		Severity:   i.Severity,
	}
}

func (i ValidationIssue) WithPath(path string) ValidationIssue {
	v := i.Clone()
	v.Path = path

	return v
}

func (i ValidationIssue) WithSeverity(s ErrorSeverity) ValidationIssue {
	v := i.Clone()
	v.Severity = s

	return v
}

func (i ValidationIssue) WithAttr(key string, value any) ValidationIssue {
	return i.WithAttrs(Attributes{
		key: value,
	})
}

func (i ValidationIssue) WithAttrs(attrs Attributes) ValidationIssue {
	if len(attrs) == 0 {
		return i
	}

	v := i.Clone()
	v.Attributes = i.Attributes.Merge(attrs)

	return v
}

func (i ValidationIssue) Error() string {
	return i.Message
}

func (i ValidationIssue) AsErrorExtension() ErrorExtensions {
	m := make(ErrorExtensions, len(i.Attributes)+5)

	for k, v := range i.Attributes {
		switch k {
		// NOTE: skip reserved keys
		case "path", "code", "component", "severity", "message":
		default:
			m[k] = v
		}
	}

	if i.Path != "" {
		m["path"] = i.Path
	}

	if i.Code != "" {
		m["code"] = i.Code
	}

	if i.Component != "" {
		m["component"] = string(i.Component)
	}

	if i.Severity != "" {
		m["severity"] = string(i.Severity)
	}

	m["message"] = i.Message

	return m
}

type ValidationIssueOption func(*ValidationIssue)

func WithAttributes(attrs map[string]interface{}) ValidationIssueOption {
	return func(i *ValidationIssue) {
		i.Attributes = attrs
	}
}

func WithComponent(component ComponentName) ValidationIssueOption {
	return func(i *ValidationIssue) {
		i.Component = component
	}
}

func WithPath(parts ...string) ValidationIssueOption {
	return func(i *ValidationIssue) {
		i.Path = FieldPathFromParts(parts...)
	}
}

func WithSeverity(severity ErrorSeverity) ValidationIssueOption {
	return func(i *ValidationIssue) {
		i.Severity = severity
	}
}

func WithCriticalSeverity() ValidationIssueOption {
	return WithSeverity(ErrorSeverityCritical)
}

func WithWarningSeverity() ValidationIssueOption {
	return WithSeverity(ErrorSeverityWarning)
}

// NewValidationIssue returns a new ValidationIssue with code and message.
func NewValidationIssue(code ErrorCode, message string, opts ...ValidationIssueOption) ValidationIssue {
	i := ValidationIssue{
		Message: message,
		Code:    code,
	}

	for _, opt := range opts {
		opt(&i)
	}

	return i
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

// AsValidationIssues returns a list of ValidationIssue from the input error or the errIn error in case:
// * the errIn is `nil`
// * any errors in the error tree that are not wrapped with WrapWithComponent or WrapWithFieldPrefix functions which are treated as critical errors
func AsValidationIssues(errIn error) (ValidationIssues, error) {
	if errIn == nil {
		return nil, nil
	}

	issues, err := asValidationIssues(errIn, "", "", false)
	if err != nil {
		return nil, errIn
	}

	return issues, nil
}

func asValidationIssues(err error, prefix string, component ComponentName, unknownAsValidationIssue bool) (ValidationIssues, error) {
	if err == nil {
		return nil, nil
	}

	switch e := err.(type) {
	case componentWrapper:
		return asValidationIssues(e.err, prefix, e.component, true)
	case fieldPrefixedWrapper:
		return asValidationIssues(e.err, FieldPathFromParts(prefix, e.prefix), component, true)
	case ValidationIssue:
		return ValidationIssues{
			ValidationIssue{
				Attributes: e.Attributes,
				Code:       e.Code,
				Component: func() ComponentName {
					if component == "" {
						return e.Component
					}

					return component
				}(),
				Message:  e.Message,
				Path:     FieldPathFromParts(prefix, e.Path),
				Severity: e.Severity,
			},
		}, nil
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
				ValidationIssue{
					Component: component,
					Message:   err.Error(),
					Path:      prefix,
					Severity:  ErrorSeverityCritical,
				},
			}, nil
		}

		return nil, err
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
