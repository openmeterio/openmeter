package appservice

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/samber/lo"
)

var _ app.AppService = (*Service)(nil)

func (s *Service) CreateApp(ctx context.Context, input app.CreateAppInput) (app.AppBase, error) {
	// Validate the input
	if err := input.Validate(); err != nil {
		return app.AppBase{}, models.NewGenericValidationError(err)
	}

	// Create the app
	appBase, err := s.adapter.CreateApp(ctx, input)
	if err != nil {
		return app.AppBase{}, err
	}

	// Emit the app created event
	event := app.NewAppCreateEvent(ctx, appBase)
	if err := s.publisher.Publish(ctx, event); err != nil {
		return app.AppBase{}, err
	}

	return appBase, nil
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
	// Validate the input
	if err := input.Validate(); err != nil {
		return nil, models.NewGenericValidationError(err)
	}

	// Update the app
	updatedApp, err := s.adapter.UpdateApp(ctx, input)
	if err != nil {
		return nil, err
	}

	// Emit the app updated event
	event := app.NewAppUpdateEvent(ctx, updatedApp.GetAppBase())
	if err := s.publisher.Publish(ctx, event); err != nil {
		return nil, err
	}

	return updatedApp, nil
}

func (s *Service) ListApps(ctx context.Context, input app.ListAppInput) (pagination.PagedResponse[app.App], error) {
	if err := input.Validate(); err != nil {
		return pagination.PagedResponse[app.App]{}, models.NewGenericValidationError(err)
	}

	return s.adapter.ListApps(ctx, input)
}

func (s *Service) UninstallApp(ctx context.Context, input app.UninstallAppInput) error {
	// Validate the input
	if err := input.Validate(); err != nil {
		return models.NewGenericValidationError(err)
	}

	// Get the app before it is deleted
	appToDelete, err := s.adapter.GetApp(ctx, app.GetAppInput(input))
	if err != nil {
		return err
	}

	// Delete the app
	if err := s.adapter.UninstallApp(ctx, input); err != nil {
		return err
	}

	appBase := appToDelete.GetAppBase()

	// FIXME: this is a hack to get the deleted app to include in the event
	// we don't read back the deleted app from the database because it will read the stripe app data
	appBase.DeletedAt = lo.ToPtr(time.Now())

	// Emit the app deleted event
	event := app.NewAppDeleteEvent(ctx, appBase)
	if err := s.publisher.Publish(ctx, event); err != nil {
		return err
	}

	return nil
}

func (s *Service) UpdateAppStatus(ctx context.Context, input app.UpdateAppStatusInput) error {
	// Validate the input
	if err := input.Validate(); err != nil {
		return models.NewGenericValidationError(err)
	}

	// Update the app status
	if err := s.adapter.UpdateAppStatus(ctx, input); err != nil {
		return err
	}

	// Get the app after status update to include in the event
	updatedApp, err := s.adapter.GetApp(ctx, app.GetAppInput(input.ID))
	if err != nil {
		return err
	}

	// Emit the app updated event
	event := app.NewAppUpdateEvent(ctx, updatedApp.GetAppBase())
	if err := s.publisher.Publish(ctx, event); err != nil {
		return err
	}

	return nil
}
