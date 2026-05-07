package subscriptionaddons

import (
	"context"

	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	ListSubscriptionAddons() ListSubscriptionAddonsHandler
	GetSubscriptionAddon() GetSubscriptionAddonHandler
}

type handler struct {
	resolveNamespace func(ctx context.Context) (string, error)
	addonService     subscriptionaddon.Service
	options          []httptransport.HandlerOption
}

func New(
	resolveNamespace func(ctx context.Context) (string, error),
	addonService subscriptionaddon.Service,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		resolveNamespace: resolveNamespace,
		addonService:     addonService,
		options:          options,
	}
}
