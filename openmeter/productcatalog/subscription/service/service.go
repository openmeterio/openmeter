package service

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/datetime"
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

func (s *service) zeroPhasesBeforeStartingPhase(p *plan.Plan, startingPhase string) error {
	nPhases := make([]plan.Phase, 0, len(p.Phases))

	reachedStartingPhase := false

	for idx, phase := range p.Phases {
		if phase.Key == startingPhase {
			reachedStartingPhase = true
		}

		if !reachedStartingPhase {
			// Instead of deleting the earlier phases, we set their length to 0
			phase.Duration = lo.ToPtr(datetime.ISODurationFromDuration(time.Duration(0)))
		}

		if idx == len(p.Phases)-1 && !reachedStartingPhase {
			return models.NewGenericValidationError(
				fmt.Errorf("starting phase %s not found in plan %s@%d", startingPhase, p.Key, p.Version),
			)
		}

		nPhases = append(nPhases, phase)
	}

	p.Phases = nPhases

	return nil
}
