package subscriptions

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	ListSubscriptions() ListSubscriptionsHandler
	GetSubscription() GetSubscriptionHandler
	CreateSubscription() CreateSubscriptionHandler
	CancelSubscription() CancelSubscriptionHandler
	UnscheduleCancelation() UnscheduleCancelationHandler
	ChangeSubscription() ChangeSubscriptionHandler
	EditSubscription() EditSubscriptionHandler
}

type handler struct {
	resolveNamespace            func(ctx context.Context) (string, error)
	customerService             customer.Service
	planService                 plan.Service
	planSubscriptionService     plansubscription.PlanSubscriptionService
	subscriptionService         subscription.Service
	subscriptionWorkflowService subscriptionworkflow.Service
	options                     []httptransport.HandlerOption
}

func New(
	resolveNamespace func(ctx context.Context) (string, error),
	customerService customer.Service,
	planService plan.Service,
	planSubscriptionService plansubscription.PlanSubscriptionService,
	subscriptionService subscription.Service,
	subscriptionWorkflowService subscriptionworkflow.Service,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		resolveNamespace:            resolveNamespace,
		customerService:             customerService,
		planService:                 planService,
		planSubscriptionService:     planSubscriptionService,
		subscriptionService:         subscriptionService,
		subscriptionWorkflowService: subscriptionWorkflowService,
		options:                     options,
	}
}
