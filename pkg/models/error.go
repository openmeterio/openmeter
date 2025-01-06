package models

import "fmt"

type MeterNotFoundError struct {
	MeterSlug string
}

func (e *MeterNotFoundError) Error() string {
	return fmt.Sprintf("meter not found: %s", e.MeterSlug)
}

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
