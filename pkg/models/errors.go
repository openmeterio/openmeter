package models

import (
	"errors"
	"fmt"
)

// GenericError is an interface that all generic errors must implement.
type GenericError interface {
	error
	Unwrap() error
}

// NewNamespaceNotFoundError returns a new NamespaceNotFoundError.
func NewNamespaceNotFoundError(namespace string) error {
	return &NamespaceNotFoundError{
		err: NewGenericNotFoundError(fmt.Errorf("namespace not found: %s", namespace)),
	}
}

var _ GenericError = &NamespaceNotFoundError{}

// IsNamespaceNotFoundError returns true if the error is a NamespaceNotFoundError.
type NamespaceNotFoundError struct {
	err error
}

func (e *NamespaceNotFoundError) Error() string {
	return e.err.Error()
}

// Unwrap returns the wrapped error.
func (e *NamespaceNotFoundError) Unwrap() error {
	return e.err
}

// NewGenericNotFoundError returns a new GenericNotFoundError.
func NewGenericNotFoundError(err error) error {
	return &GenericNotFoundError{err: err}
}

var _ GenericError = &GenericNotFoundError{}

type GenericNotFoundError struct {
	err error
}

func (e *GenericNotFoundError) Error() string {
	return fmt.Sprintf("not found error: %s", e.err)
}

func (e *GenericNotFoundError) Unwrap() error {
	return e.err
}

// IsGenericNotFoundError returns true if the error is a GenericNotFoundError.
func IsGenericNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	var e *GenericNotFoundError

	return errors.As(err, &e)
}

// NewGenericConflictError returns a new GenericConflictError.
func NewGenericConflictError(err error) error {
	return &GenericConflictError{err: err}
}

var _ GenericError = &GenericConflictError{}

type GenericConflictError struct {
	err error
}

func (e *GenericConflictError) Error() string {
	return fmt.Sprintf("conflict error: %s", e.err)
}

func (e *GenericConflictError) Unwrap() error {
	return e.err
}

// IsGenericConflictError returns true if the error is a GenericConflictError.
func IsGenericConflictError(err error) bool {
	if err == nil {
		return false
	}

	var e *GenericConflictError

	return errors.As(err, &e)
}

// NewGenericValidationError returns a new GenericForbiddenError.
func NewGenericForbiddenError(err error) error {
	return &GenericForbiddenError{err: err}
}

var _ GenericError = &GenericForbiddenError{}

type GenericForbiddenError struct {
	err error
}

func (e *GenericForbiddenError) Error() string {
	return fmt.Sprintf("forbidden error: %s", e.err)
}

func (e *GenericForbiddenError) Unwrap() error {
	return e.err
}

// IsGenericForbiddenError returns true if the error is a GenericForbiddenError.
func IsGenericForbiddenError(err error) bool {
	if err == nil {
		return false
	}

	var e *GenericForbiddenError

	return errors.As(err, &e)
}

// NewNillableGenericValidationError returns a new GenericValidationError or nil if the error is nil.
// This is useful when someone passes in an errors.Join to the error.
func NewNillableGenericValidationError(err error) error {
	if err == nil {
		return nil
	}

	return NewGenericValidationError(err)
}

// NewGenericValidationError returns a new GenericValidationError.
func NewGenericValidationError(err error) error {
	return &GenericValidationError{err: err}
}

var _ GenericError = &GenericValidationError{}

// GenericValidationError is returned when a meter is not found.
type GenericValidationError struct {
	err error
}

// Error returns the error message.
func (e *GenericValidationError) Error() string {
	return fmt.Sprintf("validation error: %s", e.err)
}

// Unwrap returns the wrapped error.
func (e *GenericValidationError) Unwrap() error {
	return e.err
}

// IsGenericValidationError returns true if the error is a GenericValidationError.
func IsGenericValidationError(err error) bool {
	if err == nil {
		return false
	}

	var e *GenericValidationError

	return errors.As(err, &e)
}

// NewGenericNotImplementedError returns a new GenericNotImplementedError.
func NewGenericNotImplementedError(err error) error {
	return &GenericNotImplementedError{err: err}
}

var _ GenericError = &GenericNotImplementedError{}

// GenericNotImplementedError is returned when a meter is not found.
type GenericNotImplementedError struct {
	err error
}

// Error returns the error message.
func (e *GenericNotImplementedError) Error() string {
	return fmt.Sprintf("not implemented error: %s", e.err)
}

// Unwrap returns the wrapped error.
func (e *GenericNotImplementedError) Unwrap() error {
	return e.err
}

// IsGenericNotImplementedError returns true if the error is a GenericNotImplementedError.
func IsGenericNotImplementedError(err error) bool {
	if err == nil {
		return false
	}

	var e *GenericNotImplementedError

	return errors.As(err, &e)
}

// GenericStatusFailedDependencyError
func NewGenericStatusFailedDependencyError(err error) error {
	return &GenericStatusFailedDependencyError{err: err}
}

var _ GenericError = &GenericStatusFailedDependencyError{}

type GenericStatusFailedDependencyError struct {
	err error
}

func (e *GenericStatusFailedDependencyError) Error() string {
	return fmt.Sprintf("status failed dependency error: %s", e.err)
}

func (e *GenericStatusFailedDependencyError) Unwrap() error {
	return e.err
}

func IsGenericStatusFailedDependencyError(err error) bool {
	if err == nil {
		return false
	}

	var e *GenericStatusFailedDependencyError

	return errors.As(err, &e)
}

// GenericPreConditionFailedError
func NewGenericPreConditionFailedError(err error) error {
	return &GenericPreConditionFailedError{err: err}
}

var _ GenericError = &GenericPreConditionFailedError{}

type GenericPreConditionFailedError struct {
	err error
}

func (e *GenericPreConditionFailedError) Error() string {
	return fmt.Sprintf("precondition failed error: %s", e.err)
}

func (e *GenericPreConditionFailedError) Unwrap() error {
	return e.err
}

func IsGenericPreConditionFailedError(err error) bool {
	if err == nil {
		return false
	}

	var e *GenericPreConditionFailedError

	return errors.As(err, &e)
}

// GenericUnauthorizedError
func NewGenericUnauthorizedError(err error) error {
	return &GenericUnauthorizedError{err: err}
}

var _ GenericError = &GenericUnauthorizedError{}

type GenericUnauthorizedError struct {
	err error
}

func (e *GenericUnauthorizedError) Error() string {
	return fmt.Sprintf("unauthorized error: %s", e.err)
}

func (e *GenericUnauthorizedError) Unwrap() error {
	return e.err
}

func IsGenericUnauthorizedError(err error) bool {
	if err == nil {
		return false
	}

	var e *GenericUnauthorizedError

	return errors.As(err, &e)
}

// ComponentName is the name of an internal or external component/service the error is related to or originated from.
type ComponentName string

// ErrorCode is the machine-readable error code.
type ErrorCode string

const (
	ErrorSeverityCritical ErrorSeverity = iota
	ErrorSeverityWarning
)

// ErrorSeverity describes the severity of an error.
type ErrorSeverity int8

func (s ErrorSeverity) String() string {
	switch s {
	case ErrorSeverityCritical:
		return "critical"
	case ErrorSeverityWarning:
		return "warning"
	default:
		return "invalid"
	}
}

func (s ErrorSeverity) Values() []string {
	return []string{
		ErrorSeverityCritical.String(),
		ErrorSeverityWarning.String(),
	}
}

type fieldPrefixedWrapper struct {
	prefix *FieldDescriptor
	err    error
}

func (p fieldPrefixedWrapper) Error() string {
	if p.prefix != nil {
		return p.prefix.String() + ": " + p.err.Error()
	}

	return p.err.Error()
}

func (p fieldPrefixedWrapper) Unwrap() error {
	return p.err
}

// ErrorWithFieldPrefix wraps an error with a field prefix. It returns nil if err is also nil.
func ErrorWithFieldPrefix(prefix *FieldDescriptor, err error) error {
	if err == nil {
		return nil
	}

	return fieldPrefixedWrapper{prefix: prefix, err: err}
}

type componentWrapper struct {
	component ComponentName
	err       error
}

func (e componentWrapper) Error() string {
	if e.component != "" {
		return string(e.component) + ": " + e.err.Error()
	}

	return e.err.Error()
}

func (e componentWrapper) Unwrap() error {
	return e.err
}

// ErrorWithComponent wraps an error with a component name. It returns nil if err is also nil.
// This can be used to add context to an error when we are crossing service boundaries.
func ErrorWithComponent(component ComponentName, err error) error {
	if err == nil {
		return nil
	}

	return componentWrapper{component: component, err: err}
}
