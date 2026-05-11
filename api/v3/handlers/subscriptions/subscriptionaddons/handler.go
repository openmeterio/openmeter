package subscriptionaddons

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	ListSubscriptionAddons() ListSubscriptionAddonsHandler
}

type handler struct {
	resolveNamespace         func(ctx context.Context) (string, error)
	subscriptionAddonService subscriptionaddon.Service
	subscriptionService      subscription.Service
	options                  []httptransport.HandlerOption
}

func New(
	resolveNamespace func(ctx context.Context) (string, error),
	subscriptionAddonService subscriptionaddon.Service,
	subscriptionService subscription.Service,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		resolveNamespace:         resolveNamespace,
		subscriptionAddonService: subscriptionAddonService,
		subscriptionService:      subscriptionService,
		options:                  options,
	}
}
