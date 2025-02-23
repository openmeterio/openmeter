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

type GenericConflictError struct {
	Inner error
}

func (e *GenericConflictError) Error() string {
	return e.Inner.Error()
}

func (e *GenericConflictError) Unwrap() error {
	return e.Inner
}

type GenericForbiddenError struct {
	Inner error
}

func (e *GenericForbiddenError) Error() string {
	return e.Inner.Error()
}

func (e *GenericForbiddenError) Unwrap() error {
	return e.Inner
}

// NewValidationError returns a new BadRequestError.
func NewValidationError(err error) error {
	return &ValidationError{err: err}
}

var _ error = &ValidationError{}

// ValidationError is returned when a meter is not found.
type ValidationError struct {
	err error
}

// Error returns the error message.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error: %s", e.err)
}

// IsValidationError returns true if the error is a BadRequestError.
func IsValidationError(err error) bool {
	if err == nil {
		return false
	}

	var e *ValidationError

	return errors.As(err, &e)
}
