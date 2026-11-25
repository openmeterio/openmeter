package service

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing/periodicrealization"
)

func (s *service) GetOrCreateRunAt(ctx context.Context, input periodicrealization.GetOrCreateRunAtInput) (periodicrealization.Run, error) {
	if err := input.Validate(); err != nil {
		return periodicrealization.Run{}, err
	}

	// Locked so that two schedulers cannot both chain a run to the same
	// previous run.
	return s.withLockedNamespace(ctx, input.Namespace, func(ctx context.Context) (periodicrealization.Run, error) {
		// AsOf is an exclusive bound, like a service period end: it asks for the
		// run that covers usage before that instant. An AsOf that is exactly a
		// period end therefore belongs to the run ending there, not to the next
		// one. Charge runs often end at UTC midnight, and such a run holds only
		// usage of the day that just ended.
		//
		// A run covers everything since the previous run's period ended, not
		// only its own period. So a time in a period that never got a run
		// belongs to the next run that exists, and no run is ever created
		// behind an existing one.
		covering, err := s.adapter.GetRunCovering(ctx, input)
		if err != nil {
			return periodicrealization.Run{}, err
		}

		if covering != nil {
			return *covering, nil
		}

		// Every existing run ended before AsOf, so the new run goes to the end
		// of the chain.
		periodTo := periodicrealization.PeriodTo(input.AsOf)

		createInput := periodicrealization.CreateRunInput{
			Namespace:    input.Namespace,
			PeriodTo:     periodTo,
			RealizeAfter: periodicrealization.RealizeAfter(periodTo),
		}

		latest, err := s.adapter.GetLatestRun(ctx, input.Namespace)
		if err != nil {
			return periodicrealization.Run{}, err
		}

		if latest != nil {
			createInput.PreviousRunID = &latest.ID
		}

		return s.adapter.CreateRun(ctx, createInput)
	})
}
