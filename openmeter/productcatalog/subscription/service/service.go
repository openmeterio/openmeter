package service

import (
	"log/slog"

	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
)

type Config struct {
	// TODO: WorkflowService and this can probably be merged
	WorkflowService     subscription.WorkflowService
	SubscriptionService subscription.Service
	PlanAdapter         plansubscription.Adapter
	Logger              *slog.Logger
}

type service struct {
	Config
}

func New(c Config) plansubscription.PlanSubscriptionService {
	return &service{
		Config: c,
	}
}
