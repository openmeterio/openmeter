package models

import (
	"maps"
)

var _ Equaler[Metadata] = (*Metadata)(nil)

type Metadata map[string]string

func (m Metadata) Equal(v Metadata) bool {
	return maps.Equal(m, v)
}

func (m Metadata) ToMap() map[string]string {
	return m
}

func (m Metadata) Merge(d Metadata) Metadata {
	if len(m) == 0 && len(d) == 0 {
		return nil
	}

	r := make(Metadata)

	for k, v := range m {
		r[k] = v
	}

	for k, v := range d {
		r[k] = v
	}

	return r
}

func (m Metadata) Clone() (Metadata, error) {
	return Metadata(maps.Clone(m)), nil
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
