package account

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/ledger"
)

type AddressData struct {
	SubAccountID string
	AccountType  ledger.AccountType
	Route        ledger.Route
	RouteID      string
	RoutingKey   ledger.RoutingKey
}

func NewAddressFromData(data AddressData) (*Address, error) {
	subRoute, err := newSubAccountRouteFromAddressData(data)
	if err != nil {
		return nil, err
	}

	return &Address{
		data:     data,
		subRoute: subRoute,
	}, nil
}

func newSubAccountRouteFromAddressData(data AddressData) (ledger.SubAccountRoute, error) {
	if data.SubAccountID == "" {
		return ledger.SubAccountRoute{}, errors.New("sub-account id is required")
	}
	if err := data.AccountType.Validate(); err != nil {
		return ledger.SubAccountRoute{}, fmt.Errorf("account type: %w", err)
	}
	if data.RouteID == "" {
		return ledger.SubAccountRoute{}, errors.New("route id is required")
	}

	subRoute, err := ledger.NewSubAccountRouteFromData(data.RouteID, data.RoutingKey, data.Route)
	if err != nil {
		return ledger.SubAccountRoute{}, fmt.Errorf("route: %w", err)
	}

	return subRoute, nil
}

type Address struct {
	data     AddressData
	subRoute ledger.SubAccountRoute
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
	return a.subRoute
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
