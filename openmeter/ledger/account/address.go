package account

import (
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/models"
)

type AddressData struct {
	SubAccountID models.NamespacedID
	AccountType  ledger.AccountType
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

var _ ledger.PostingAddress = (*Address)(nil)

func (a *Address) SubAccountID() models.NamespacedID {
	return a.data.SubAccountID
}

func (a *Address) AccountType() ledger.AccountType {
	return a.data.AccountType
}

func (a *Address) Equal(other ledger.PostingAddress) bool {
	if a.SubAccountID() != other.SubAccountID() {
		return false
	}

	if a.AccountType() != other.AccountType() {
		return false
	}

	return true
}
