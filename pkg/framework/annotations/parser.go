package annotations

import "github.com/openmeterio/openmeter/pkg/models"

type Annotation struct {
	Key   string
	Value string
}

// Parser can be used to interact with Metadata Annotations
// Typical usecases could be access control validations through metadata.
// Most business logic should be implemented on proper entity properties, but those resulting from external user provided annotations, or those resulting from cross domain behavior, can be modeled as annotations.
type Parser[T models.Annotated, K any] interface {
	Parse(val T) (K, error)
}

type Writer[T models.Annotated] interface {
	Annotate(val T, annotation Annotation) (T, error)
}
