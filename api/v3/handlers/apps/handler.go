package apps

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/app"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	// App handlers
	ListApps() ListAppsHandler
	GetApp() GetAppHandler
	UninstallApp() UninstallAppHandler

	CatalogHandler
}

type CatalogHandler interface {
	ListAppCatalog() ListAppCatalogHandler
	GetAppCatalog() GetAppCatalogHandler
	InstallApp() InstallAppHandler
}

var _ Handler = (*handler)(nil)

type handler struct {
	resolveNamespace func(ctx context.Context) (string, error)
	appService       app.Service
	billingService   billing.Service
	stripeAppService appstripe.Service
	options          []httptransport.HandlerOption
}

func New(
	resolveNamespace func(ctx context.Context) (string, error),
	appService app.Service,
	billingService billing.Service,
	stripeAppService appstripe.Service,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		resolveNamespace: resolveNamespace,
		appService:       appService,
		billingService:   billingService,
		stripeAppService: stripeAppService,
		options:          options,
	}
}
