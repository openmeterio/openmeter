package models

import (
	"errors"
	"fmt"
)

type NamespaceNotFoundError struct {
	Namespace string
}

func (e *NamespaceNotFoundError) Error() string {
	return fmt.Sprintf("namespace not found: %s", e.Namespace)
}

// TODO: these should be picked up in a general server error handler
type GenericUserError struct {
	Inner error
}

func (e *GenericUserError) Error() string {
	return e.Inner.Error()
}

func (e *GenericUserError) Unwrap() error {
	return e.Inner
}

// NewGenericConflictError returns a new GenericConflictError.
func NewGenericConflictError(err error) error {
	return &GenericConflictError{Inner: err}
}

var _ error = &GenericConflictError{}

type GenericConflictError struct {
	Inner error
}

func (e *GenericConflictError) Error() string {
	return e.Inner.Error()
}

func (e *GenericConflictError) Unwrap() error {
	return e.Inner
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
	return &GenericForbiddenError{Inner: err}
}

var _ error = &GenericForbiddenError{}

type GenericForbiddenError struct {
	Inner error
}

func (e *GenericForbiddenError) Error() string {
	return e.Inner.Error()
}

func (e *GenericForbiddenError) Unwrap() error {
	return e.Inner
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

var _ error = &GenericValidationError{}

// GenericValidationError is returned when a meter is not found.
type GenericValidationError struct {
	err error
}

// Error returns the error message.
func (e *GenericValidationError) Error() string {
	return fmt.Sprintf("validation error: %s", e.err)
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

var _ error = &GenericNotImplementedError{}

// GenericNotImplementedError is returned when a meter is not found.
type GenericNotImplementedError struct {
	err error
}

// Error returns the error message.
func (e *GenericNotImplementedError) Error() string {
	return fmt.Sprintf("validation error: %s", e.err)
}

// IsGenericNotImplementedError returns true if the error is a BadRequestError.
func IsGenericNotImplementedError(err error) bool {
	if err == nil {
		return false
	}

	var e *GenericNotImplementedError

	return errors.As(err, &e)
}
