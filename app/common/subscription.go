package common

import (
	"log/slog"

	"github.com/google/wire"

	"github.com/openmeterio/openmeter/openmeter/customer"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	subscriptionchangeservice "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription/service"
	"github.com/openmeterio/openmeter/openmeter/registry"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionentitlement "github.com/openmeterio/openmeter/openmeter/subscription/entitlement"
	subscriptionrepo "github.com/openmeterio/openmeter/openmeter/subscription/repo"
	subscriptionservice "github.com/openmeterio/openmeter/openmeter/subscription/service"
	subscriptioncustomer "github.com/openmeterio/openmeter/openmeter/subscription/validators/customer"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
)

var Subscription = wire.NewSet(
	NewSubscriptionServices,
)

// Combine Srvice and WorkflowService into one struct
// We do this to able to initialize the Service and WorkflowService together
// and share the same subscriptionRepo.
type SubscriptionServiceWithWorkflow struct {
	Service                 subscription.Service
	WorkflowService         subscription.WorkflowService
	PlanSubscriptionService plansubscription.PlanSubscriptionService
}

func NewSubscriptionServices(
	logger *slog.Logger,
	db *entdb.Client,
	featureConnector feature.FeatureConnector,
	entitlementRegistry *registry.Entitlement,
	customerService customer.Service,
	planService plan.Service,
	eventPublisher eventbus.Publisher,
) (SubscriptionServiceWithWorkflow, error) {
	subscriptionRepo := subscriptionrepo.NewSubscriptionRepo(db)
	subscriptionPhaseRepo := subscriptionrepo.NewSubscriptionPhaseRepo(db)
	subscriptionItemRepo := subscriptionrepo.NewSubscriptionItemRepo(db)

	subscriptionEntitlementAdapter := subscriptionentitlement.NewSubscriptionEntitlementAdapter(
		entitlementRegistry.Entitlement,
		subscriptionItemRepo,
		subscriptionItemRepo,
	)

	subscriptionService := subscriptionservice.New(subscriptionservice.ServiceConfig{
		SubscriptionRepo:      subscriptionRepo,
		SubscriptionPhaseRepo: subscriptionPhaseRepo,
		SubscriptionItemRepo:  subscriptionItemRepo,
		CustomerService:       customerService,
		EntitlementAdapter:    subscriptionEntitlementAdapter,
		FeatureService:        featureConnector,
		TransactionManager:    subscriptionRepo,
		Publisher:             eventPublisher,
	})

	subscriptionWorkflowService := subscriptionservice.NewWorkflowService(subscriptionservice.WorkflowServiceConfig{
		Service:            subscriptionService,
		CustomerService:    customerService,
		TransactionManager: subscriptionRepo,
	})

	planSubscriptionService := subscriptionchangeservice.New(subscriptionchangeservice.Config{
		WorkflowService:     subscriptionWorkflowService,
		SubscriptionService: subscriptionService,
		PlanService:         planService,
		CustomerService:     customerService,
		Logger:              logger.With("subsystem", "subscription.change.service"),
	})

	validator, err := subscriptioncustomer.NewValidator(subscriptionService, customerService)
	if err != nil {
		return SubscriptionServiceWithWorkflow{}, err
	}

	customerService.RegisterRequestValidator(validator)

	return SubscriptionServiceWithWorkflow{
		Service:                 subscriptionService,
		WorkflowService:         subscriptionWorkflowService,
		PlanSubscriptionService: planSubscriptionService,
	}, nil
}
