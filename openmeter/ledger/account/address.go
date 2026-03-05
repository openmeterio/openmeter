package account

import (
	"github.com/openmeterio/openmeter/openmeter/ledger"
)

type AddressData struct {
	SubAccountID      string
	AccountType       ledger.AccountType
	RouteID           string
	RoutingKeyVersion ledger.RoutingKeyVersion
	RoutingKey        string
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

func (a *Address) SubAccountID() string {
	return a.data.SubAccountID
}

func (a *Address) AccountType() ledger.AccountType {
	return a.data.AccountType
}

func (a *Address) Route() ledger.SubAccountRoute {
	return ledger.MustNewSubAccountRoute(
		a.data.RouteID,
		ledger.MustNewRoutingKey(a.data.RoutingKeyVersion, a.data.RoutingKey),
	)
}

func (a *Address) Equal(other ledger.PostingAddress) bool {
	if a.SubAccountID() != other.SubAccountID() {
		return false
	}

	if a.AccountType() != other.AccountType() {
		return false
	}

	if a.Route().ID() != other.Route().ID() {
		return false
	}

	if a.Route().RoutingKey().Version() != other.Route().RoutingKey().Version() {
		return false
	}

	if a.Route().RoutingKey().Value() != other.Route().RoutingKey().Value() {
		return false
	}

	return true
}
