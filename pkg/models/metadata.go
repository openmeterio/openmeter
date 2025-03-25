package models

import (
	"maps"
)

var _ Equaler[Metadata] = (*Metadata)(nil)

type Metadata map[string]string

func (m Metadata) Equal(v Metadata) bool {
	return maps.Equal(m, v)
}

func NewMetadata[T ~map[string]string](m T) Metadata {
	return Metadata(m)
}

type (
	annotatedMarker bool // marker is used so only AnnotatedModel can implement Annotated
	Metadatad       interface {
		annotated() annotatedMarker
	}
)

type MetadataModel struct {
	Metadata Metadata `json:"metadata,omitempty"`
}

var _ Metadatad = MetadataModel{}

func (a MetadataModel) annotated() annotatedMarker {
	return true
}
