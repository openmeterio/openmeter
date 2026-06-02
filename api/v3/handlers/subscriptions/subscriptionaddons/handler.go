package subscriptionaddons

import (
	"context"

	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	ListSubscriptionAddons() ListSubscriptionAddonsHandler
	GetSubscriptionAddon() GetSubscriptionAddonHandler
	UpdateSubscriptionAddon() UpdateSubscriptionAddonHandler
}

type handler struct {
	resolveNamespace            func(ctx context.Context) (string, error)
	addonService                subscriptionaddon.Service
	subscriptionWorkflowService subscriptionworkflow.Service
	options                     []httptransport.HandlerOption
}

func New(
	resolveNamespace func(ctx context.Context) (string, error),
	addonService subscriptionaddon.Service,
	subscriptionWorkflowService subscriptionworkflow.Service,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		resolveNamespace:            resolveNamespace,
		addonService:                addonService,
		subscriptionWorkflowService: subscriptionWorkflowService,
		options:                     options,
	}
}
