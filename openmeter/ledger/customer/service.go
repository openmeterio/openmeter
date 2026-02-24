package customer

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
)

type Service struct{}

func (s *Service) GetCustomerAccounts(ctx context.Context, customerID customer.CustomerID) (ledger.CustomerAccounts, error) {
	panic("not implemented")
}

func (s *Service) CreateCustomerAccounts(ctx context.Context, customerID customer.CustomerID) (ledger.CustomerAccounts, error) {
	panic("not implemented")
}
