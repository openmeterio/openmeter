package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/samber/lo"
)

type Attributes map[any]any

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

// AsStringMap converts Attributes into a map[string]any by:
// - keeping string keys as-is
// - stringifying comparable non-string keys as "<type>:<value>"
func (a Attributes) AsStringMap() map[string]any {
	if len(a) == 0 {
		return nil
	}

	out := make(map[string]any, len(a))
	for k, v := range a {
		if sk, ok := k.(string); ok {
			out[sk] = v
			continue
		}

		t := reflect.TypeOf(k)
		if t == nil {
			continue
		}
		if t.Comparable() {
			key := fmt.Sprintf("%T:%v", k, k)
			out[key] = v
		}
	}

	return out
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

type ErrorExtension map[string]any

var (
	_ error          = (*ValidationIssue)(nil)
	_ json.Marshaler = (*ValidationIssue)(nil)
)

type ValidationIssue struct {
	attributes Attributes
	code       ErrorCode
	component  ComponentName
	message    string
	field      FieldSelectors
	severity   ErrorSeverity
}

func (i ValidationIssue) MarshalJSON() ([]byte, error) {
	m := i.AsErrorExtension()

	return json.Marshal(m)
}

func (i ValidationIssue) Clone() ValidationIssue {
	return ValidationIssue{
		attributes: i.attributes.Clone(),
		code:       i.code,
		component:  i.component,
		message:    i.message,
		field:      i.field,
		severity:   i.severity,
	}
}

func (i ValidationIssue) Attributes() Attributes {
	return i.attributes
}

func (i ValidationIssue) Code() ErrorCode {
	return i.code
}

func (i ValidationIssue) Component() ComponentName {
	return i.component
}

func (i ValidationIssue) Message() string {
	return i.message
}

func (i ValidationIssue) Field() FieldSelectors {
	return i.field
}

func (i ValidationIssue) Severity() ErrorSeverity {
	return i.severity
}

func (i ValidationIssue) WithField(parts ...FieldSelector) ValidationIssue {
	v := i.Clone()
	v.field = parts

	return v
}

func (i ValidationIssue) WithPathString(parts ...string) ValidationIssue {
	return i.WithField(lo.Map(parts, func(item string, _ int) FieldSelector {
		return NewFieldSelector(item)
	})...)
}

func (i ValidationIssue) WithSeverity(s ErrorSeverity) ValidationIssue {
	v := i.Clone()
	v.severity = s

	return v
}

func (i ValidationIssue) WithAttr(key any, value any) ValidationIssue {
	if key == nil {
		panic("validation issue attribute key must not be nil")
	}

	if t := reflect.TypeOf(key); t == nil || !t.Comparable() {
		panic("validation issue attribute key is not comparable")
	}

	return i.WithAttrs(Attributes{
		key: value,
	})
}

func (i ValidationIssue) WithAttrs(attrs Attributes) ValidationIssue {
	if len(attrs) == 0 {
		return i
	}

	for k := range attrs {
		if k == nil {
			panic("validation issue attribute key must not be nil")
		}

		if t := reflect.TypeOf(k); t == nil || !t.Comparable() {
			panic("validation issue attribute key is not comparable")
		}
	}

	v := i.Clone()
	v.attributes = i.attributes.Merge(attrs)

	return v
}

func (i ValidationIssue) SetAttributes(attrs Attributes) ValidationIssue {
	if len(attrs) == 0 {
		return i
	}

	v := i.Clone()
	v.attributes = attrs
	return v
}

func (i ValidationIssue) Error() string {
	return i.message
}

func (i ValidationIssue) AsErrorExtension() ErrorExtension {
	attrs := i.attributes.AsStringMap()
	m := make(ErrorExtension, len(attrs)+5)

	for key, v := range attrs {
		switch key {
		// NOTE: skip reserved keys
		case "field", "code", "component", "severity", "message":
			// skip
		default:
			m[key] = v
		}
	}

	if path := i.field.JSONPath(); path != "" {
		m["field"] = i.field
	}

	if i.code != "" {
		m["code"] = i.code
	}

	if i.component != "" {
		m["component"] = string(i.component)
	}

	m["message"] = i.message
	m["severity"] = i.severity.String()

	return m
}

type ValidationIssueOption func(*ValidationIssue)

func WithAttribute(key any, value any) ValidationIssueOption {
	if key == nil {
		panic("validation issue attribute key must not be nil")
	}

	if t := reflect.TypeOf(key); t == nil || !t.Comparable() {
		panic("validation issue attribute key is not comparable")
	}

	return func(i *ValidationIssue) {
		i.attributes = i.attributes.Merge(Attributes{key: value})
	}
}

func WithAttributes(attrs Attributes) ValidationIssueOption {
	return func(i *ValidationIssue) {
		if len(attrs) == 0 {
			i.attributes = nil
			return
		}

		i.attributes = attrs
	}
}

func WithComponent(component ComponentName) ValidationIssueOption {
	return func(i *ValidationIssue) {
		i.component = component
	}
}

func WithField(parts ...FieldSelector) ValidationIssueOption {
	return func(i *ValidationIssue) {
		i.field = parts
	}
}

func WithFieldString(parts ...string) ValidationIssueOption {
	return func(i *ValidationIssue) {
		i.field = lo.Map(parts, func(item string, _ int) FieldSelector {
			return NewFieldSelector(item)
		})
	}
}

func WithSeverity(severity ErrorSeverity) ValidationIssueOption {
	return func(i *ValidationIssue) {
		i.severity = severity
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
		message: message,
		code:    code,
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

// AsValidationIssues returns a list of ValidationIssue from the input error or the errIn error in case:
// * the errIn is `nil`
// * any leaf errors in the error tree that are not wrapped with WrapWithComponent or WrapWithFieldPrefix functions are treated as critical errors
func AsValidationIssues(errIn error) (ValidationIssues, error) {
	if errIn == nil {
		return nil, nil
	}

	issues, err := asValidationIssues(errIn, NewFieldSelectors(), "", false)
	if err != nil {
		return nil, errIn
	}

	return issues, nil
}

func asValidationIssues(err error, prefix FieldSelectors, component ComponentName, unknownAsValidationIssue bool) (ValidationIssues, error) {
	if err == nil {
		return nil, nil
	}

	switch e := err.(type) {
	case componentWrapper:
		return asValidationIssues(e.err, prefix, e.component, true)
	case fieldPrefixedWrapper:
		return asValidationIssues(e.err, e.prefix.WithPrefix(prefix), component, true)
	case ValidationIssue:
		return ValidationIssues{
			ValidationIssue{
				attributes: e.attributes,
				code:       e.code,
				component: func() ComponentName {
					if component == "" {
						return e.component
					}

					return component
				}(),
				message:  e.message,
				field:    e.field.WithPrefix(prefix),
				severity: e.severity,
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
					component: component,
					message:   err.Error(),
					field:     prefix,
					severity:  ErrorSeverityCritical,
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
