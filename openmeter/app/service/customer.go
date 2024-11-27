package appservice

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/app"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var _ app.AppService = (*Service)(nil)

func (s *Service) ListCustomerData(ctx context.Context, input app.ListCustomerDataInput) (pagination.PagedResponse[appentity.CustomerData], error) {
	return s.adapter.ListCustomerData(ctx, input)
}

func (s *Service) UpsertCustomerData(ctx context.Context, input app.UpsertCustomerDataInput) error {
	return s.adapter.UpsertCustomerData(ctx, input)
}

func (s *Service) DeleteCustomerData(ctx context.Context, input app.DeleteCustomerDataInput) error {
	return s.adapter.DeleteCustomerData(ctx, input)
}
