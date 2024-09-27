package appservice

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var _ app.AppService = (*Service)(nil)

func (s *Service) GetApp(ctx context.Context, input app.GetAppInput) (app.App, error) {
	if err := input.Validate(); err != nil {
		return nil, app.ValidationError{
			Err: err,
		}
	}

	return s.adapter.GetApp(ctx, input)
}

func (s *Service) ListApps(ctx context.Context, input app.ListAppInput) (pagination.PagedResponse[app.App], error) {
	if err := input.Validate(); err != nil {
		return pagination.PagedResponse[app.App]{}, app.ValidationError{
			Err: err,
		}
	}

	return s.adapter.ListApps(ctx, input)
}

func (s *Service) UninstallApp(ctx context.Context, input app.DeleteAppInput) error {
	if err := input.Validate(); err != nil {
		return app.ValidationError{
			Err: err,
		}
	}

	return s.adapter.UninstallApp(ctx, input)
}
