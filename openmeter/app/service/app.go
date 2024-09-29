package appservice

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/app"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var _ app.AppService = (*Service)(nil)

func (s *Service) CreateApp(ctx context.Context, input appentity.CreateAppInput) (appentity.App, error) {
	if err := input.Validate(); err != nil {
		return nil, app.ValidationError{
			Err: err,
		}
	}

	return app.WithTx(ctx, s.adapter, func(ctx context.Context, adapter app.TxAdapter) (appentity.App, error) {
		return adapter.CreateApp(ctx, input)
	})
}

func (s *Service) GetApp(ctx context.Context, input appentity.GetAppInput) (appentity.App, error) {
	if err := input.Validate(); err != nil {
		return nil, app.ValidationError{
			Err: err,
		}
	}

	return s.adapter.GetApp(ctx, input)
}

func (s *Service) GetDefaultApp(ctx context.Context, input appentity.GetDefaultAppInput) (appentity.App, error) {
	if err := input.Validate(); err != nil {
		return nil, app.ValidationError{
			Err: err,
		}
	}

	return s.adapter.GetDefaultApp(ctx, input)
}

func (s *Service) ListApps(ctx context.Context, input appentity.ListAppInput) (pagination.PagedResponse[appentity.App], error) {
	if err := input.Validate(); err != nil {
		return pagination.PagedResponse[appentity.App]{}, app.ValidationError{
			Err: err,
		}
	}

	return s.adapter.ListApps(ctx, input)
}

func (s *Service) UninstallApp(ctx context.Context, input appentity.DeleteAppInput) error {
	if err := input.Validate(); err != nil {
		return app.ValidationError{
			Err: err,
		}
	}

	return app.WithTxNoValue(ctx, s.adapter, func(ctx context.Context, adapter app.TxAdapter) error {
		return adapter.UninstallApp(ctx, input)
	})
}
