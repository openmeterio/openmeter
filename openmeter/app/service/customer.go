package appservice

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/app"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var _ app.AppService = (*Service)(nil)

func (s *Service) ListCustomerData(ctx context.Context, input app.ListCustomerInput) (pagination.PagedResponse[appentity.CustomerApp], error) {
	return s.adapter.ListCustomerData(ctx, input)
}

func (s *Service) EnsureCustomer(ctx context.Context, input app.EnsureCustomerInput) error {
	return s.adapter.EnsureCustomer(ctx, input)
}

func (s *Service) DeleteCustomer(ctx context.Context, input app.DeleteCustomerInput) error {
	return s.adapter.DeleteCustomer(ctx, input)
}
