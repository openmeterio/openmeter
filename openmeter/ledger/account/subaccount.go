package account

import (
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
	sAcc := &SubAccount{
		data: data,
	}

	return sAcc, nil
}

type SubAccount struct {
	data SubAccountData
}

var _ ledger.SubAccount = (*SubAccount)(nil)

func (s *SubAccount) Address() ledger.PostingAddress {
	return NewAddressFromData(AddressData{
		SubAccountID:      s.data.ID,
		AccountType:       s.data.AccountType,
		RouteID:           s.data.RouteMeta.ID,
		RoutingKeyVersion: s.data.RouteMeta.RoutingKeyVersion,
		RoutingKey:        s.data.RouteMeta.RoutingKey,
	})
}

func (s *SubAccount) AccountID() string {
	return s.data.AccountID
}
