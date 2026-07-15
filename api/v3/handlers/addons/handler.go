package addons

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	CreateAddon() CreateAddonHandler
	DeleteAddon() DeleteAddonHandler
	GetAddon() GetAddonHandler
	ListAddons() ListAddonsHandler
	UpdateAddon() UpdateAddonHandler
	PublishAddon() PublishAddonHandler
	ArchiveAddon() ArchiveAddonHandler
}

type handler struct {
	resolveNamespace  func(ctx context.Context) (string, error)
	service           addon.Service
	unitConfigEnabled bool
	options           []httptransport.HandlerOption
}

func New(
	resolveNamespace func(ctx context.Context) (string, error),
	service addon.Service,
	unitConfigEnabled bool,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		resolveNamespace:  resolveNamespace,
		service:           service,
		unitConfigEnabled: unitConfigEnabled,
		options:           options,
	}
}
