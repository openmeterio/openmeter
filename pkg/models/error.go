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
