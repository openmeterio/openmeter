package appcustominvoicing

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type Adapter interface {
	CustomerDataAdapter
	AppConfigAdapter

	entutils.TxCreator
}

type CustomerDataAdapter interface {
	GetCustomerData(ctx context.Context, input GetAppCustomerDataInput) (CustomerData, error)
	UpsertCustomerData(ctx context.Context, input UpsertCustomerDataInput) error
	DeleteCustomerData(ctx context.Context, input DeleteAppCustomerDataInput) error
}

type AppConfigAdapter interface {
	GetAppConfiguration(ctx context.Context, input app.AppID) (Configuration, error)
	UpsertAppConfiguration(ctx context.Context, input UpsertAppConfigurationInput) error
	DeleteAppConfiguration(ctx context.Context, input app.AppID) error
}
