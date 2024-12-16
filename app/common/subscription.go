package common

import (
	"log/slog"

	"github.com/google/wire"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/customer"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	subscriptionchangeservice "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription/service"
	"github.com/openmeterio/openmeter/openmeter/registry"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionentitlement "github.com/openmeterio/openmeter/openmeter/subscription/adapters/entitlement"
	subscriptionrepo "github.com/openmeterio/openmeter/openmeter/subscription/repo"
	subscriptionservice "github.com/openmeterio/openmeter/openmeter/subscription/service"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
)

var Subscription = wire.NewSet(
	NewSubscriptionService,
	NewPlanSubscriptionService,
)

// Combine Srvice and WorkflowService into one struct
// We do this to able to initialize the Service and WorkflowService together
// and share the same subscriptionRepo.
type SubscriptionServiceWithWorkflow struct {
	Service         subscription.Service
	WorkflowService subscription.WorkflowService
}

func NewSubscriptionService(
	logger *slog.Logger,
	db *entdb.Client,
	productcatalogConfig config.ProductCatalogConfiguration,
	entitlementConfig config.EntitlementsConfiguration,
	featureConnector feature.FeatureConnector,
	entitlementRegistry *registry.Entitlement,
	customerService customer.Service,
	planService plan.Service,
	eventPublisher eventbus.Publisher,
) (SubscriptionServiceWithWorkflow, error) {
	if !productcatalogConfig.Enabled {
		return SubscriptionServiceWithWorkflow{}, nil
	}

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
		TransactionManager:    subscriptionRepo,
		Publisher:             eventPublisher,
	})

	subscriptionWorkflowService := subscriptionservice.NewWorkflowService(subscriptionservice.WorkflowServiceConfig{
		Service:            subscriptionService,
		CustomerService:    customerService,
		TransactionManager: subscriptionRepo,
	})

	return SubscriptionServiceWithWorkflow{
		Service:         subscriptionService,
		WorkflowService: subscriptionWorkflowService,
	}, nil
}

func NewPlanSubscriptionService(
	planService plan.Service,
	subsServices SubscriptionServiceWithWorkflow,
	logger *slog.Logger,
) plansubscription.PlanSubscriptionService {
	adapter := plansubscription.NewPlanSubscriptionAdapter(plansubscription.PlanSubscriptionAdapterConfig{
		PlanService: planService,
		Logger:      logger.With("subsystem", "subscription.plan.adapter"),
	})

	return subscriptionchangeservice.New(subscriptionchangeservice.Config{
		WorkflowService:     subsServices.WorkflowService,
		SubscriptionService: subsServices.Service,
		Logger:              logger.With("subsystem", "subscription.change.service"),
		PlanAdapter:         adapter,
	})
}
