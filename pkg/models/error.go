package models

import "fmt"

// TODO: these should be picked up in a general server error handler

// MeterNotFoundError represents an error when a meter is not found
var _ error = (*MeterNotFoundError)(nil)

type MeterNotFoundError struct {
	MeterSlug string
}

func (e *MeterNotFoundError) Error() string {
	return fmt.Sprintf("meter not found: %s", e.MeterSlug)
}

// NamespaceNotFoundError represents an error when a namespace is not found
var _ error = (*NamespaceNotFoundError)(nil)

type NamespaceNotFoundError struct {
	Namespace string
}

func (e *NamespaceNotFoundError) Error() string {
	return fmt.Sprintf("namespace not found: %s", e.Namespace)
}

// GenericServerError represents an error when a server error occurs
var _ error = (*GenericUserError)(nil)

type GenericUserError struct {
	Inner error
}

func (e *GenericUserError) Error() string {
	return e.Inner.Error()
}

func (e *GenericUserError) Unwrap() error {
	return e.Inner
}

// GenericNotFoundError represents an error when a resource is not found
var _ error = (*GenericNotFoundError)(nil)

type GenericNotFoundError genericError

func (e GenericNotFoundError) Error() string {
	return e.Err.Error()
}

func (e GenericNotFoundError) Unwrap() error {
	return e.Err
}

// GenericBadGateway represents an error when a resource is not found
var _ error = (*GenericBadGateway)(nil)

type GenericBadGateway genericError

func (e GenericBadGateway) Error() string {
	return e.Err.Error()
}

func (e GenericBadGateway) Unwrap() error {
	return e.Err
}

// GenericValidationError represents an error when a resource is not found
var _ error = (*GenericValidationError)(nil)

type GenericValidationError genericError

func (e GenericValidationError) Error() string {
	return e.Err.Error()
}

// GenericConflictError is a generic error for conflict errors
var _ error = (*GenericConflictError)(nil)

type GenericConflictError struct {
	Inner error
}

func (e *GenericConflictError) Error() string {
	return e.Inner.Error()
}

func (e *GenericConflictError) Unwrap() error {
	return e.Inner
}

// GenericForbiddenError is a generic error for forbidden errors
var _ error = (*GenericForbiddenError)(nil)

type GenericForbiddenError struct {
	Inner error
}

func (e *GenericForbiddenError) Error() string {
	return e.Inner.Error()
}

func (e *GenericForbiddenError) Unwrap() error {
	return e.Inner
}

// genericError represents a generic error
type genericError struct {
	Err error
}
