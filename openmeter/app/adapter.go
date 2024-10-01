package app

import (
	"context"

	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	entcontext "github.com/openmeterio/openmeter/pkg/framework/entutils/context"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type Adapter interface {
	AppAdapter

	DB() entcontext.DB
}

type MarketplaceAdapter interface {
	Register(input appentity.RegisterMarketplaceListingInput) error
	Get(ctx context.Context, input appentity.MarketplaceGetInput) (appentity.RegistryItem, error)
	List(ctx context.Context, input appentity.MarketplaceListInput) (pagination.PagedResponse[appentity.RegistryItem], error)
	InstallAppWithAPIKey(ctx context.Context, input appentity.InstallAppWithAPIKeyInput) (appentity.App, error)
	GetOauth2InstallURL(ctx context.Context, input appentity.GetOauth2InstallURLInput) (appentity.GetOauth2InstallURLOutput, error)
	AuthorizeOauth2Install(ctx context.Context, input appentity.AuthorizeOauth2InstallInput) error
}

type AppAdapter interface {
	CreateApp(ctx context.Context, input appentity.CreateAppInput) (appentitybase.AppBase, error)
	GetApp(ctx context.Context, input appentity.GetAppInput) (appentity.App, error)
	GetDefaultApp(ctx context.Context, input appentity.GetDefaultAppInput) (appentity.App, error)
	ListApps(ctx context.Context, input appentity.ListAppInput) (pagination.PagedResponse[appentity.App], error)
	UninstallApp(ctx context.Context, input appentity.DeleteAppInput) error
}
