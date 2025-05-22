package service

import (
	"fmt"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Config struct {
	// TODO: WorkflowService and this can probably be merged
	WorkflowService     subscriptionworkflow.Service
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

func (s *service) removePhasesBeforeStartingPhase(p *plan.Plan, startingPhase string) error {
	for idx, phase := range p.Phases {
		if phase.Key == startingPhase {
			// Let's filter out the phases before the starting phase
			p.Phases = p.Phases[idx:]
			break
		}

		if idx == len(p.Phases)-1 {
			return models.NewGenericValidationError(
				fmt.Errorf("starting phase %s not found in plan %s@%d", startingPhase, p.Key, p.Version),
			)
		}
	}

	return nil
}
