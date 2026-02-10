package account

import (
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/models"
)

type AddressData struct {
	ID          models.NamespacedID
	AccountType ledger.AccountType
	Dimensions  map[string]*Dimension
}

func NewAddressFromData(data AddressData) *Address {
	return &Address{
		data: data,
	}
}

type Address struct {
	data AddressData
}

// ----------------------------------------------------------------------------
// Let's implement ledger.Address interface
// ----------------------------------------------------------------------------

var _ ledger.Address = (*Address)(nil)

func (a *Address) ID() models.NamespacedID {
	return a.data.ID
}

func (a *Address) Type() ledger.AccountType {
	return a.data.AccountType
}

func (a *Address) Dimensions() []ledger.Dimension {
	return lo.MapToSlice(a.data.Dimensions, func(_ string, value *Dimension) ledger.Dimension {
		return value
	})
}

func (a *Address) Equal(other ledger.Address) bool {
	if a.ID() != other.ID() {
		return false
	}

	if a.Type() != other.Type() {
		return false
	}

	// order of dimensions is not important
	if len(a.Dimensions()) != len(other.Dimensions()) {
		return false
	}

	otherDims := make(map[string]ledger.Dimension, len(other.Dimensions()))
	for _, dimension := range other.Dimensions() {
		otherDims[dimension.Key()] = dimension
	}

	for _, dimension := range a.Dimensions() {
		otherDimension, ok := otherDims[dimension.Key()]
		if !ok || !dimension.Equal(otherDimension) {
			return false
		}
	}

	return true
}
