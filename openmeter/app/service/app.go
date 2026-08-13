package appservice

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/servicehooks"
)

var _ app.AppService = (*Service)(nil)

func (s *Service) RegisterHook(name string, hook servicehooks.Hook[app.LifecycleEvent]) error {
	return s.hooks.Register(name, hook)
}

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

func (s *Service) UpdateApp(ctx context.Context, input app.UpdateAppInput) (app.App, error) {
	// Validate the input
	if err := input.Validate(); err != nil {
		return nil, models.NewGenericValidationError(err)
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (app.App, error) {
		// Update the app
		updatedApp, err := s.adapter.UpdateApp(ctx, input)
		if err != nil {
			return nil, err
		}

		// Update the app specific entity
		if input.AppConfigUpdate != nil {
			err := updatedApp.UpdateAppConfig(ctx, input.AppConfigUpdate)
			if err != nil {
				return nil, err
			}

			updatedApp, err = s.adapter.GetApp(ctx, input.AppID)
			if err != nil {
				return nil, err
			}
		}

		// Emit the app updated event
		event, err := app.NewAppUpdateEvent(ctx, updatedApp)
		if err != nil {
			return nil, err
		}

		if err := s.publisher.Publish(ctx, event); err != nil {
			return nil, err
		}

		return updatedApp, nil
	})
}

func (s *Service) ListApps(ctx context.Context, input app.ListAppInput) (pagination.Result[app.App], error) {
	if err := input.Validate(); err != nil {
		return pagination.Result[app.App]{}, models.NewGenericValidationError(err)
	}

	return s.adapter.ListApps(ctx, input)
}

func (s *Service) UninstallApp(ctx context.Context, input app.UninstallAppInput) error {
	// Validate the input
	if err := input.Validate(); err != nil {
		return models.NewGenericValidationError(err)
	}

	type uninstallResult struct {
		appBase      app.AppBase
		eventAppData app.EventAppData
	}

	result, err := transaction.Run(ctx, s.adapter, func(ctx context.Context) (uninstallResult, error) {
		// Existing app
		existingApp, err := s.adapter.GetApp(ctx, input)
		if err != nil {
			return uninstallResult{}, err
		}

		before := existingApp.GetAppBase()
		if err := s.hooks.Invoke(ctx, app.LifecycleEvent{
			Operation: app.OperationKindUninstall,
			Before:    &before,
		}); err != nil {
			return uninstallResult{}, fmt.Errorf("invoking app lifecycle hooks: %w", err)
		}

		// Delete the app
		appBase, err := s.adapter.UninstallApp(ctx, input)
		if err != nil {
			return uninstallResult{}, err
		}

		// Capture event data while the concrete app still exists.
		eventAppData, err := existingApp.GetEventAppData()
		if err != nil {
			return uninstallResult{}, err
		}

		return uninstallResult{
			appBase:      *appBase,
			eventAppData: eventAppData,
		}, nil
	})
	if err != nil {
		return err
	}

	// Emit the app deleted event after the uninstall transaction has committed.
	event := app.NewAppDeleteEvent(ctx, result.appBase, result.eventAppData)
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
	updatedApp, err := s.adapter.GetApp(ctx, input.ID)
	if err != nil {
		return err
	}

	// Emit the app updated event
	event, err := app.NewAppUpdateEvent(ctx, updatedApp)
	if err != nil {
		return err
	}

	if err := s.publisher.Publish(ctx, event); err != nil {
		return err
	}

	return nil
}
