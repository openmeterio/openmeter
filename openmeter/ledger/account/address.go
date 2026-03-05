package account

import (
	"errors"
	"fmt"

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
	if err := validateAddressData(data); err != nil {
		panic(err)
	}

	return &Address{
		data: data,
	}
}

func validateAddressData(data AddressData) error {
	if data.SubAccountID == "" {
		return errors.New("sub-account id is required")
	}
	if err := data.AccountType.Validate(); err != nil {
		return fmt.Errorf("account type: %w", err)
	}
	if data.RouteID == "" {
		return errors.New("route id is required")
	}
	routingKey, err := ledger.NewRoutingKey(data.RoutingKeyVersion, data.RoutingKey)
	if err != nil {
		return fmt.Errorf("routing key: %w", err)
	}
	if _, err := ledger.NewSubAccountRoute(data.RouteID, routingKey); err != nil {
		return fmt.Errorf("route: %w", err)
	}

	return nil
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
