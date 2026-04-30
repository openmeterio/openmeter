package appservice

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var _ app.AppService = (*Service)(nil)

func (s *Service) CreateApp(ctx context.Context, input app.CreateAppInput) (app.AppBase, error) {
	if err := input.Validate(); err != nil {
		return app.AppBase{}, models.NewGenericValidationError(err)
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (app.AppBase, error) {
		appBase, err := s.adapter.CreateApp(ctx, input)
		if err != nil {
			return app.AppBase{}, err
		}

		if err := s.hooks.PostCreate(ctx, &appBase); err != nil {
			return app.AppBase{}, err
		}

		event := app.NewAppCreateEvent(ctx, appBase)
		if err := s.publisher.Publish(ctx, event); err != nil {
			return app.AppBase{}, err
		}

		return appBase, nil
	})
}

func (s *Service) GetApp(ctx context.Context, input app.GetAppInput) (app.App, error) {
	if err := input.Validate(); err != nil {
		return nil, models.NewGenericValidationError(err)
	}

	return s.adapter.GetApp(ctx, input)
}

func (s *Service) UpdateApp(ctx context.Context, input app.UpdateAppInput) (app.App, error) {
	if err := input.Validate(); err != nil {
		return nil, models.NewGenericValidationError(err)
	}

	// Fetch existing app to get the hook payload.
	existingApp, err := s.adapter.GetApp(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (app.App, error) {
		existingBase := existingApp.GetAppBase()

		if err := s.hooks.PreUpdate(ctx, &existingBase); err != nil {
			return nil, err
		}

		updatedApp, err := s.adapter.UpdateApp(ctx, input)
		if err != nil {
			return nil, err
		}

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

		updatedBase := updatedApp.GetAppBase()

		if err := s.hooks.PostUpdate(ctx, &updatedBase); err != nil {
			return nil, err
		}

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
	if err := input.Validate(); err != nil {
		return models.NewGenericValidationError(err)
	}

	// Fetch existing app to get the hook/event payload.
	existingApp, err := s.adapter.GetApp(ctx, input)
	if err != nil {
		return err
	}

	_, err = transaction.Run(ctx, s.adapter, func(ctx context.Context) (struct{}, error) {
		existingBase := existingApp.GetAppBase()

		if err := s.hooks.PreDelete(ctx, &existingBase); err != nil {
			return struct{}{}, err
		}

		appBase, err := s.adapter.UninstallApp(ctx, input)
		if err != nil {
			return struct{}{}, err
		}

		if err := s.hooks.PostDelete(ctx, appBase); err != nil {
			return struct{}{}, err
		}

		eventAppData, err := existingApp.GetEventAppData()
		if err != nil {
			return struct{}{}, err
		}

		event := app.NewAppDeleteEvent(ctx, *appBase, eventAppData)
		return struct{}{}, s.publisher.Publish(ctx, event)
	})

	return err
}

func (s *Service) UpdateAppStatus(ctx context.Context, input app.UpdateAppStatusInput) error {
	if err := input.Validate(); err != nil {
		return models.NewGenericValidationError(err)
	}

	// Fetch existing app for hook payload.
	existingApp, err := s.adapter.GetApp(ctx, input.ID)
	if err != nil {
		return err
	}

	_, err = transaction.Run(ctx, s.adapter, func(ctx context.Context) (struct{}, error) {
		existingBase := existingApp.GetAppBase()

		if err := s.hooks.PreUpdate(ctx, &existingBase); err != nil {
			return struct{}{}, err
		}

		if err := s.adapter.UpdateAppStatus(ctx, input); err != nil {
			return struct{}{}, err
		}

		updatedApp, err := s.adapter.GetApp(ctx, input.ID)
		if err != nil {
			return struct{}{}, err
		}

		updatedBase := updatedApp.GetAppBase()

		if err := s.hooks.PostUpdate(ctx, &updatedBase); err != nil {
			return struct{}{}, err
		}

		event, err := app.NewAppUpdateEvent(ctx, updatedApp)
		if err != nil {
			return struct{}{}, err
		}

		return struct{}{}, s.publisher.Publish(ctx, event)
	})

	return err
}
