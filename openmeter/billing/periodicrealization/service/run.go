package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/periodicrealization"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (s *service) GetOrCreateRunAt(ctx context.Context, input periodicrealization.GetOrCreateRunAtInput) (periodicrealization.Run, error) {
	if err := input.Validate(); err != nil {
		return periodicrealization.Run{}, err
	}

	// Locked so that two schedulers cannot both find the namespace empty.
	return s.withLockedNamespace(ctx, input.Namespace, func(ctx context.Context) (periodicrealization.Run, error) {
		// TODO: temporary until run chaining is figured out. A namespace that
		// already has a run should get the existing run for the period, or a new
		// one linked to its previous run.
		hasRuns, err := s.adapter.HasRuns(ctx, input.Namespace)
		if err != nil {
			return periodicrealization.Run{}, err
		}

		if hasRuns {
			return periodicrealization.Run{}, models.NewGenericNotImplementedError(
				fmt.Errorf("namespace %s already has a periodic realization run: scheduling further runs is not supported yet", input.Namespace),
			)
		}

		periodTo := periodicrealization.PeriodTo(input.AsOf)

		return s.adapter.CreateRun(ctx, periodicrealization.CreateRunInput{
			Namespace:    input.Namespace,
			PeriodTo:     periodTo,
			RealizeAfter: periodicrealization.RealizeAfter(periodTo),
		})
	})
}
