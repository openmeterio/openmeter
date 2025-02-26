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
	err       error
	namespace string
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

// IsGenericNotImplementedError returns true if the error is a BadRequestError.
func IsGenericNotImplementedError(err error) bool {
	if err == nil {
		return false
	}

	var e *GenericNotImplementedError

	return errors.As(err, &e)
}
