package account

import (
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/models"
)

type SubAccountData struct {
	ID          string
	Namespace   string
	Annotations models.Annotations
	CreatedAt   time.Time

	AccountID   string
	AccountType ledger.AccountType

	Route     ledger.Route
	RouteMeta SubAccountRouteData
}

type SubAccountRouteData struct {
	ID                string
	RoutingKeyVersion ledger.RoutingKeyVersion
	RoutingKey        string
}

func NewSubAccountFromData(data SubAccountData) (*SubAccount, error) {
	routingKey, err := ledger.NewRoutingKey(data.RouteMeta.RoutingKeyVersion, data.RouteMeta.RoutingKey)
	if err != nil {
		return nil, fmt.Errorf("routing key: %w", err)
	}

	addr, err := NewAddressFromData(AddressData{
		SubAccountID: data.ID,
		AccountType:  data.AccountType,
		Route:        data.Route,
		RouteID:      data.RouteMeta.ID,
		RoutingKey:   routingKey,
	})
	if err != nil {
		return nil, fmt.Errorf("posting address: %w", err)
	}

	return &SubAccount{
		data:    data,
		address: addr,
	}, nil
}

type SubAccount struct {
	data    SubAccountData
	address *Address
}

var _ ledger.SubAccount = (*SubAccount)(nil)

func (s *SubAccount) Address() ledger.PostingAddress {
	return s.address
}

func (s *SubAccount) Route() ledger.Route {
	return s.data.Route
}

func (s *SubAccount) AccountID() models.NamespacedID {
	return models.NamespacedID{
		Namespace: s.data.Namespace,
		ID:        s.data.AccountID,
	}
}
