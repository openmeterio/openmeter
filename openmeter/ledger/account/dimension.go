package account

import (
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Dimension struct {
	ID          models.NamespacedID
	Annotations models.Annotations
	models.ManagedModel

	DimensionKey   string
	DimensionValue string // TBD
}

var _ ledger.Dimension = (*Dimension)(nil)

func (d *Dimension) Equal(other ledger.Dimension) bool {
	o, ok := other.(*Dimension)
	if !ok {
		return false
	}

	return d.DimensionKey == o.DimensionKey && d.DimensionValue == o.DimensionValue
}

func (d *Dimension) Key() string {
	return d.DimensionKey
}

func (d *Dimension) Value() any {
	return d.DimensionValue
}
