package subscriptionaddons

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	CreateSubscriptionAddon() CreateSubscriptionAddonHandler
	ListSubscriptionAddons() ListSubscriptionAddonsHandler
	GetSubscriptionAddon() GetSubscriptionAddonHandler
}

type handler struct {
	resolveNamespace            func(ctx context.Context) (string, error)
	addonService                subscriptionaddon.Service
	subscriptionService         subscription.Service
	subscriptionWorkflowService subscriptionworkflow.Service
	options                     []httptransport.HandlerOption
}

func New(
	resolveNamespace func(ctx context.Context) (string, error),
	addonService subscriptionaddon.Service,
	subscriptionService subscription.Service,
	subscriptionWorkflowService subscriptionworkflow.Service,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		resolveNamespace:            resolveNamespace,
		addonService:                addonService,
		subscriptionService:         subscriptionService,
		subscriptionWorkflowService: subscriptionWorkflowService,
		options:                     options,
	}
}
