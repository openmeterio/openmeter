package apps

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	// App handlers
	ListApps() ListAppsHandler
	GetApp() GetAppHandler
}

var _ Handler = (*handler)(nil)

type handler struct {
	resolveNamespace func(ctx context.Context) (string, error)
	appService       app.Service
	options          []httptransport.HandlerOption
}

func New(
	resolveNamespace func(ctx context.Context) (string, error),
	appService app.Service,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		resolveNamespace: resolveNamespace,
		appService:       appService,
		options:          options,
	}
}
