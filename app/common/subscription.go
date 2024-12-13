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
	"github.com/openmeterio/openmeter/openmeter/registry"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionentitlement "github.com/openmeterio/openmeter/openmeter/subscription/adapters/entitlement"
	subscriptionrepo "github.com/openmeterio/openmeter/openmeter/subscription/repo"
	subscriptionservice "github.com/openmeterio/openmeter/openmeter/subscription/service"
)

var Subscription = wire.NewSet(
	NewSubscriptionService,
	NewSubscriptionWorkflowService,
	NewPLanSubscriptionAdapter,
)

func NewSubscriptionService(
	logger *slog.Logger,
	db *entdb.Client,
	productcatalogConfig config.ProductCatalogConfiguration,
	entitlementConfig config.EntitlementsConfiguration,
	featureConnector feature.FeatureConnector,
	entitlementRegistry *registry.Entitlement,
	customerService customer.Service,
	planService plan.Service,
) (subscription.Service, error) {
	// TODO: remove this check after enabled by default
	if db == nil {
		return nil, nil
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
	})

	return subscriptionService, nil
}

func NewSubscriptionWorkflowService(
	logger *slog.Logger,
	db *entdb.Client,
	customerService customer.Service,
	subscriptionService subscription.Service,
) (subscription.WorkflowService, error) {
	// TODO: remove this check after enabled by default
	if db == nil {
		return nil, nil
	}

	// TODO share the same repo with subscriptionService
	subscriptionRepo := subscriptionrepo.NewSubscriptionRepo(db)

	subscriptionWorkflowService := subscriptionservice.NewWorkflowService(subscriptionservice.WorkflowServiceConfig{
		Service:            subscriptionService,
		CustomerService:    customerService,
		TransactionManager: subscriptionRepo,
	})

	return subscriptionWorkflowService, nil
}

func NewPLanSubscriptionAdapter(
	logger *slog.Logger,
	db *entdb.Client,
	planService plan.Service,
) plansubscription.Adapter {
	return plansubscription.NewPlanSubscriptionAdapter(plansubscription.PlanSubscriptionAdapterConfig{
		PlanService: planService,
		Logger:      logger.With("subsystem", "subscription.plan.adapter"),
	})
}
