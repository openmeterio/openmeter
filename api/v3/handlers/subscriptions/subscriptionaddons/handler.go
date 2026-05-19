package subscriptionaddons

import (
	"context"

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
	SubscriptionWorkflowService subscriptionworkflow.Service
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
		SubscriptionWorkflowService: subscriptionWorkflowService,
		options:                     options,
	}
}
