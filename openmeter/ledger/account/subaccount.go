package account

import (
	"context"
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

func NewSubAccountFromData(data SubAccountData, account *Account) (*SubAccount, error) {
	sAcc := &SubAccount{
		data:    data,
		account: account,
	}

	return sAcc, nil
}

type SubAccount struct {
	data    SubAccountData
	account *Account
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

func (s *SubAccount) Route() ledger.Route {
	return s.data.Route
}

func (s *SubAccount) AccountID() string {
	return s.data.AccountID
}

func (s *SubAccount) GetBalance(ctx context.Context) (ledger.Balance, error) {
	if s.account == nil {
		return nil, fmt.Errorf("parent account is required")
	}

	res, err := s.account.GetBalance(ctx, s.data.Route.Filter())
	if err != nil {
		return nil, fmt.Errorf("failed to get balance for sub-account %s: %w", s.data.ID, err)
	}

	return res, nil
}
