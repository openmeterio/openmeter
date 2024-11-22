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
