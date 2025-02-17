package service

import (
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
)

type Config struct {
	// TODO: WorkflowService and this can probably be merged
	WorkflowService     subscription.WorkflowService
	SubscriptionService subscription.Service
	PlanService         plan.Service
	Logger              *slog.Logger
	CustomerService     customer.Service
}

type service struct {
	Config
}

func New(c Config) plansubscription.PlanSubscriptionService {
	return &service{
		Config: c,
	}
}
