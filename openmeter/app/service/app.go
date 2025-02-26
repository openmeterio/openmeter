package appservice

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var _ app.AppService = (*Service)(nil)

func (s *Service) CreateApp(ctx context.Context, input app.CreateAppInput) (app.AppBase, error) {
	if err := input.Validate(); err != nil {
		return app.AppBase{}, models.NewGenericValidationError(err)
	}

	return s.adapter.CreateApp(ctx, input)
}

func (s *Service) GetApp(ctx context.Context, input app.GetAppInput) (app.App, error) {
	if err := input.Validate(); err != nil {
		return nil, models.NewGenericValidationError(err)
	}

	return s.adapter.GetApp(ctx, input)
}

func (s *Service) GetDefaultApp(ctx context.Context, input app.GetDefaultAppInput) (app.App, error) {
	if err := input.Validate(); err != nil {
		return nil, models.NewGenericValidationError(err)
	}

	return s.adapter.GetDefaultApp(ctx, input)
}

func (s *Service) UpdateApp(ctx context.Context, input app.UpdateAppInput) (app.App, error) {
	if err := input.Validate(); err != nil {
		return nil, models.NewGenericValidationError(err)
	}

	return s.adapter.UpdateApp(ctx, input)
}

func (s *Service) ListApps(ctx context.Context, input app.ListAppInput) (pagination.PagedResponse[app.App], error) {
	if err := input.Validate(); err != nil {
		return pagination.PagedResponse[app.App]{}, models.NewGenericValidationError(err)
	}

	return s.adapter.ListApps(ctx, input)
}

func (s *Service) UninstallApp(ctx context.Context, input app.UninstallAppInput) error {
	if err := input.Validate(); err != nil {
		return models.NewGenericValidationError(err)
	}

	return s.adapter.UninstallApp(ctx, input)
}

func (s *Service) UpdateAppStatus(ctx context.Context, input app.UpdateAppStatusInput) error {
	if err := input.Validate(); err != nil {
		return models.NewGenericValidationError(err)
	}

	return s.adapter.UpdateAppStatus(ctx, input)
}
