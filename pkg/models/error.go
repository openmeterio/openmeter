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

// TODO: this should be picked up in a general server error handler
type GenericUserError struct {
	Message string
}

func (e *GenericUserError) Error() string {
	return e.Message
}
