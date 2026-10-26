package models

import (
	"encoding/json"
	"maps"
	"reflect"

	"github.com/samber/lo"
)

var (
	_ error          = Error{}
	_ json.Marshaler = Error{}
	_ Equaler[Error] = Error{}
)

type ErrorExtension map[string]any

// Error carries diagnostic context without classifying the failure as validation.
type Error = structuredError[generalErrorKind]

type generalErrorKind struct{}

// The kind is only a type parameter: fluent methods keep the error's type.
type structuredError[K any] struct {
	attributes Attributes
	code       ErrorCode
	component  ComponentName
	message    string
	field      *FieldDescriptor
	severity   ErrorSeverity

	// Earlier copies keep errors.Is(enriched, original) working after adding context.
	previous error
	// The underlying failure. Unwrap exposes it to errors.Is and errors.As.
	cause error
}

func (i structuredError[K]) MarshalJSON() ([]byte, error) {
	m := i.AsErrorExtension()

	return json.Marshal(m)
}

func (i structuredError[K]) Clone() structuredError[K] {
	return structuredError[K]{
		attributes: i.attributes.Clone(),
		code:       i.code,
		component:  i.component,
		message:    i.message,
		field:      i.field,
		severity:   i.severity,
		previous:   i,
		cause:      i.cause,
	}
}

// Attributes returns a shallow-copy of the underlying attributes map.
func (i structuredError[K]) Attributes() Attributes {
	return maps.Clone(i.attributes)
}

func (i structuredError[K]) Code() ErrorCode {
	return i.code
}

func (i structuredError[K]) Component() ComponentName {
	return i.component
}

func (i structuredError[K]) Message() string {
	return i.message
}

func (i structuredError[K]) Field() *FieldDescriptor {
	if i.field == nil {
		return NewFieldSelectorGroup()
	}

	return i.field
}

func (i structuredError[K]) Severity() ErrorSeverity {
	return i.severity
}

func (i structuredError[K]) With(opts ...ErrorOption) structuredError[K] {
	v := i.Clone()
	for _, opt := range opts {
		opt((*Error)(&v))
	}

	return v
}

func (i structuredError[K]) WithField(parts ...*FieldDescriptor) structuredError[K] {
	v := i.Clone()
	group := NewFieldSelectorGroup(parts...)

	v.field = group

	return v
}

func (i structuredError[K]) WithPathString(parts ...string) structuredError[K] {
	return i.WithField(lo.Map(parts, func(item string, _ int) *FieldDescriptor {
		return NewFieldSelector(item)
	})...)
}

func (i structuredError[K]) WithSeverity(s ErrorSeverity) structuredError[K] {
	v := i.Clone()
	v.severity = s

	return v
}

func (i structuredError[K]) WithAttr(key any, value any) structuredError[K] {
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

func (i structuredError[K]) WithAttrs(attrs Attributes) structuredError[K] {
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

func (i structuredError[K]) SetAttributes(attrs Attributes) structuredError[K] {
	v := i.Clone()

	v.attributes = attrs

	return v
}

func (i structuredError[K]) Error() string {
	return i.message
}

// Keep matching the original error after adding context. Earlier copies are
// identity history, not causes: Unwrap should expose the actual underlying failure.
func (i structuredError[K]) Is(target error) bool {
	targetError, ok := asStructuredError[K](target)
	if !ok {
		return false
	}

	for {
		if i.Equal(targetError) {
			return true
		}

		previous, ok := asStructuredError[K](i.previous)
		if !ok {
			return false
		}

		i = previous
	}
}

func (i structuredError[K]) Unwrap() error { return i.cause }

// Cause returns the underlying failure, without diagnostic context added by this error.
func (i structuredError[K]) Cause() error { return i.cause }

func (i structuredError[K]) Equal(other structuredError[K]) bool {
	return maps.EqualFunc(i.attributes, other.attributes, equalErrorAttributeValue) &&
		i.code == other.code &&
		i.component == other.component &&
		i.message == other.message &&
		i.severity == other.severity &&
		i.Field().String() == other.Field().String()
}

// equalErrorAttributeValue is used to compare attribute values for equality.
// It's better than reflect.DeepEqual because it handles nil values properly (DeepEqual would consider nil and empty value equal)
func equalErrorAttributeValue(left, right any) bool {
	if left == nil || right == nil {
		return left == right
	}

	// Let's not use DeepEqual for comparable types
	if t := reflect.TypeOf(left); t != nil && t.Comparable() {
		return left == right
	}

	return reflect.DeepEqual(left, right)
}

func asStructuredError[K any](err error) (structuredError[K], bool) {
	switch t := err.(type) {
	case structuredError[K]:
		return t, true
	case *structuredError[K]:
		if t == nil {
			return structuredError[K]{}, false
		}

		return *t, true
	default:
		return structuredError[K]{}, false
	}
}

func (i structuredError[K]) AsErrorExtension() ErrorExtension {
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

	if path := i.Field().JSONPath(); path != "" {
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

type ErrorOption func(*Error)

func WithAttribute(key any, value any) ErrorOption {
	if key == nil {
		panic("validation issue attribute key must not be nil")
	}

	if t := reflect.TypeOf(key); t == nil || !t.Comparable() {
		panic("validation issue attribute key is not comparable")
	}

	return func(i *Error) {
		i.attributes = i.attributes.Merge(Attributes{key: value})
	}
}

func WithAttributes(attrs Attributes) ErrorOption {
	return func(i *Error) {
		if len(attrs) == 0 {
			i.attributes = nil
			return
		}

		i.attributes = attrs
	}
}

func WithComponent(component ComponentName) ErrorOption {
	return func(i *Error) {
		i.component = component
	}
}

func WithField(parts ...*FieldDescriptor) ErrorOption {
	return func(i *Error) {
		sels := NewFieldSelectorGroup(parts...)

		i.field = sels
	}
}

func WithFieldString(parts ...string) ErrorOption {
	return func(i *Error) {
		sels := NewFieldSelectorGroup(lo.Map(parts, func(item string, _ int) *FieldDescriptor {
			return NewFieldSelector(item)
		})...)

		i.field = sels
	}
}

func WithSeverity(severity ErrorSeverity) ErrorOption {
	return func(i *Error) {
		i.severity = severity
	}
}

func WithCriticalSeverity() ErrorOption {
	return WithSeverity(ErrorSeverityCritical)
}

func WithWarningSeverity() ErrorOption {
	return WithSeverity(ErrorSeverityWarning)
}

func withMessage(message string) ErrorOption {
	return func(i *Error) {
		i.message = message
	}
}

func withCode(code ErrorCode) ErrorOption {
	return func(i *Error) {
		i.code = code
	}
}

func newStructuredError[K any](opts ...ErrorOption) structuredError[K] {
	i := structuredError[K]{}

	for _, opt := range opts {
		opt((*Error)(&i))
	}

	return i
}

// NewError creates a structured system error. It is not a validation issue.
func NewError(code ErrorCode, message string, opts ...ErrorOption) Error {
	return newStructuredError[generalErrorKind](append([]ErrorOption{withCode(code), withMessage(message)}, opts...)...)
}

// WithCause retains the underlying error for errors.Is/As and errors.Unwrap.
func WithCause(cause error) ErrorOption {
	return func(i *Error) { i.cause = cause }
}
