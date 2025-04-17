package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

var _ appcustominvoicing.FactoryService = (*Service)(nil)

func (s *Service) CreateApp(ctx context.Context, input appcustominvoicing.CreateAppInput) (app.AppBase, error) {
	if err := input.Validate(); err != nil {
		return app.AppBase{}, fmt.Errorf("invalid input: %w", err)
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (app.AppBase, error) {
		// Let's create the app first
		appBase, err := s.appService.CreateApp(ctx, app.CreateAppInput{
			Namespace: input.Namespace,
			Name:      input.Name,
			Type:      app.AppTypeCustomInvoicing,
		})
		if err != nil {
			return app.AppBase{}, fmt.Errorf("failed to create app: %w", err)
		}

		// Let's create the app settings entity
		err = s.adapter.UpsertAppConfiguration(ctx, appcustominvoicing.UpsertAppConfigurationInput{
			AppID:         app.AppID{ID: appBase.ID, Namespace: appBase.Namespace},
			Configuration: input.Config,
		})
		if err != nil {
			return app.AppBase{}, fmt.Errorf("failed to create app settings: %w", err)
		}

		return appBase, nil
	})
}

func (s *Service) DeleteApp(ctx context.Context, input app.UninstallAppInput) error {
	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		return s.adapter.DeleteAppConfiguration(ctx, input)
	})
}

func (s *Service) UpsertAppConfiguration(ctx context.Context, input appcustominvoicing.UpsertAppConfigurationInput) error {
	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		return s.adapter.UpsertAppConfiguration(ctx, input)
	})
}

func (s *Service) GetAppConfiguration(ctx context.Context, appID app.AppID) (appcustominvoicing.Configuration, error) {
	return s.adapter.GetAppConfiguration(ctx, appID)
}
