package common

import (
	"log/slog"

	"github.com/google/wire"

	"github.com/openmeterio/openmeter/app/config"
	billingsubscription "github.com/openmeterio/openmeter/openmeter/billing/subscription"
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
	NewSubscriptionServices,
	BillingSubscriptionValidator,
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
	productcatalogConfig config.ProductCatalogConfiguration,
	entitlementConfig config.EntitlementsConfiguration,
	featureConnector feature.FeatureConnector,
	entitlementRegistry *registry.Entitlement,
	customerService customer.Service,
	planService plan.Service,
	eventPublisher eventbus.Publisher,
	billingSubscriptionValidator *billingsubscription.Validator,
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

	validators := []subscription.SubscriptionValidator{}
	if billingSubscriptionValidator != nil {
		validators = append(validators, billingSubscriptionValidator)
	}

	subscriptionService := subscriptionservice.New(subscriptionservice.ServiceConfig{
		SubscriptionRepo:      subscriptionRepo,
		SubscriptionPhaseRepo: subscriptionPhaseRepo,
		SubscriptionItemRepo:  subscriptionItemRepo,
		CustomerService:       customerService,
		EntitlementAdapter:    subscriptionEntitlementAdapter,
		TransactionManager:    subscriptionRepo,
		Publisher:             eventPublisher,
		Validators:            validators,
		Logger:                logger.With("subsystem", "subscription.service"),
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
		Logger:              logger.With("subsystem", "subscription.change.service"),
	})

	return SubscriptionServiceWithWorkflow{
		Service:                 subscriptionService,
		WorkflowService:         subscriptionWorkflowService,
		PlanSubscriptionService: planSubscriptionService,
	}, nil
}
