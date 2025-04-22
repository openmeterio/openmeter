package appcustominvoicing

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/app"
)

type Service interface {
	CustomerDataService

	FactoryService
}

type CustomerDataService interface {
	GetCustomerData(ctx context.Context, input GetAppCustomerDataInput) (CustomerData, error)
	UpsertCustomerData(ctx context.Context, input UpsertCustomerDataInput) error
	DeleteCustomerData(ctx context.Context, input DeleteAppCustomerDataInput) error
}

type FactoryService interface {
	CreateApp(ctx context.Context, input CreateAppInput) (app.AppBase, error)
	DeleteApp(ctx context.Context, input app.UninstallAppInput) error
	UpsertAppConfiguration(ctx context.Context, input UpsertAppConfigurationInput) error
	GetAppConfiguration(ctx context.Context, appID app.AppID) (Configuration, error)
}
