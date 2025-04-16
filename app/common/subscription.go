package common

import (
	"log/slog"

	"github.com/google/wire"

	"github.com/openmeterio/openmeter/openmeter/customer"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	subscriptionchangeservice "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription/service"
	"github.com/openmeterio/openmeter/openmeter/registry"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	subscriptionaddonrepo "github.com/openmeterio/openmeter/openmeter/subscription/addon/repo"
	subscriptionaddonservice "github.com/openmeterio/openmeter/openmeter/subscription/addon/service"
	subscriptionentitlement "github.com/openmeterio/openmeter/openmeter/subscription/entitlement"
	subscriptionrepo "github.com/openmeterio/openmeter/openmeter/subscription/repo"
	subscriptionservice "github.com/openmeterio/openmeter/openmeter/subscription/service"
	subscriptioncustomer "github.com/openmeterio/openmeter/openmeter/subscription/validators/customer"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	subscriptionworkflowservice "github.com/openmeterio/openmeter/openmeter/subscription/workflow/service"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
)

var Subscription = wire.NewSet(
	NewSubscriptionServices,
)

// TODO: break up to multiple initializers
type SubscriptionServiceWithWorkflow struct {
	Service                  subscription.Service
	WorkflowService          subscriptionworkflow.Service
	PlanSubscriptionService  plansubscription.PlanSubscriptionService
	SubscriptionAddonService subscriptionaddon.Service
}

func NewSubscriptionServices(
	logger *slog.Logger,
	db *entdb.Client,
	featureConnector feature.FeatureConnector,
	entitlementRegistry *registry.Entitlement,
	customerService customer.Service,
	planService plan.Service,
	addonService addon.Service,
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

	subAddRepo := subscriptionaddonrepo.NewSubscriptionAddonRepo(db)
	subAddQtyRepo := subscriptionaddonrepo.NewSubscriptionAddonQuantityRepo(db)

	subAddSvc := subscriptionaddonservice.NewService(subscriptionaddonservice.Config{
		TxManager:     subAddRepo,
		Logger:        logger,
		AddonService:  addonService,
		SubService:    subscriptionService,
		SubAddRepo:    subAddRepo,
		SubAddQtyRepo: subAddQtyRepo,
	})

	subscriptionWorkflowService := subscriptionworkflowservice.NewWorkflowService(subscriptionworkflowservice.WorkflowServiceConfig{
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
		Service:                  subscriptionService,
		WorkflowService:          subscriptionWorkflowService,
		PlanSubscriptionService:  planSubscriptionService,
		SubscriptionAddonService: subAddSvc,
	}, nil
}
