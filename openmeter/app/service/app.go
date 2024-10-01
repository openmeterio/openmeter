package appservice

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/app"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	entcontext "github.com/openmeterio/openmeter/pkg/framework/entutils/context"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var _ app.AppService = (*Service)(nil)

func (s *Service) CreateApp(ctx context.Context, input appentity.CreateAppInput) (appentitybase.AppBase, error) {
	if err := input.Validate(); err != nil {
		return appentitybase.AppBase{}, app.ValidationError{
			Err: err,
		}
	}

	return entcontext.WithTx(ctx, s.adapter.DB(), func(ctx context.Context) (appentitybase.AppBase, error) {
		return s.adapter.CreateApp(ctx, input)
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

	return entcontext.WithTxNoValue(ctx, s.adapter.DB(), func(ctx context.Context) error {
		return s.adapter.UninstallApp(ctx, input)
	})
}
