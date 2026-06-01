package subscriptions

import (
	"context"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/featuregate"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	ListSubscriptions() ListSubscriptionsHandler
	GetSubscription() GetSubscriptionHandler
	CreateSubscription() CreateSubscriptionHandler
	CancelSubscription() CancelSubscriptionHandler
	UnscheduleCancelation() UnscheduleCancelationHandler
	ChangeSubscription() ChangeSubscriptionHandler
}

type handler struct {
	resolveNamespace        func(ctx context.Context) (string, error)
	customerService         customer.Service
	planService             plan.Service
	planSubscriptionService plansubscription.PlanSubscriptionService
	subscriptionService     subscription.Service
	options                 []httptransport.HandlerOption
	featureGate             featuregate.Gate
	credits                 config.CreditsConfiguration
}

func New(
	resolveNamespace func(ctx context.Context) (string, error),
	customerService customer.Service,
	planService plan.Service,
	planSubscriptionService plansubscription.PlanSubscriptionService,
	subscriptionService subscription.Service,
	featureGate featuregate.Gate,
	credits config.CreditsConfiguration,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		resolveNamespace:        resolveNamespace,
		customerService:         customerService,
		planService:             planService,
		planSubscriptionService: planSubscriptionService,
		subscriptionService:     subscriptionService,
		options:                 options,
		featureGate:             featureGate,
		credits:                 credits,
	}
}
