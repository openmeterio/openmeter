package service

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func (s service) ListPhases(ctx context.Context, params plan.ListPhasesInput) (pagination.PagedResponse[plan.Phase], error) {
	fn := func(ctx context.Context) (pagination.PagedResponse[plan.Phase], error) {
		if err := params.Validate(); err != nil {
			return pagination.PagedResponse[plan.Phase]{}, fmt.Errorf("invalid list PlanPhases params: %w", err)
		}

		return s.adapter.ListPhases(ctx, params)
	}

	return transaction.Run(ctx, s.adapter, fn)
}

func (s service) CreatePhase(ctx context.Context, params plan.CreatePhaseInput) (*plan.Phase, error) {
	fn := func(ctx context.Context) (*plan.Phase, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid create PlanPhase params: %w", err)
		}

		logger := s.logger.With(
			"operation", "create",
			"namespace", params.Namespace,
			"phase.key", params.Key,
		)

		p, err := s.adapter.GetPlan(ctx, plan.GetPlanInput{
			NamespacedID: models.NamespacedID{
				Namespace: params.Namespace,
				ID:        params.PlanID,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get Plan: %w", err)
		}

		allowedPlanStatuses := []plan.PlanStatus{plan.DraftStatus, plan.ScheduledStatus}
		planStatus := p.Status()
		if !lo.Contains(allowedPlanStatuses, p.Status()) {
			return nil, fmt.Errorf("only Plans in %+v can be updated, but it has %s state", allowedPlanStatuses, planStatus)
		}

		for _, planPhase := range p.Phases {
			if planPhase.StartAfter == params.StartAfter {
				return nil, fmt.Errorf("there is already a PlanPhase with the same StartAfter perdiod: %q", planPhase.Key)
			}
		}

		logger.Debug("creating PlanPhase")

		if len(params.RateCards) > 0 {
			if err := s.expandFeatures(ctx, params.Namespace, &params.RateCards); err != nil {
				return nil, fmt.Errorf("failed to expand Features for RateCards in PlanPhase: %w", err)
			}
		}

		phase, err := s.adapter.CreatePhase(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to create PlanPhase: %w", err)
		}

		logger.With("phase.id", phase.ID).Debug("PlanPhase created")

		return phase, nil
	}

	return transaction.Run(ctx, s.adapter, fn)
}

func (s service) DeletePhase(ctx context.Context, params plan.DeletePhaseInput) error {
	fn := func(ctx context.Context) (interface{}, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid delete PlanPhase params: %w", err)
		}

		logger := s.logger.With(
			"operation", "delete",
			"namespace", params.Namespace,
			"phase.key", params.Key,
		)

		logger.Debug("deleting PlanPhase")

		p, err := s.adapter.GetPlan(ctx, plan.GetPlanInput{
			NamespacedID: models.NamespacedID{
				Namespace: params.Namespace,
				ID:        params.PlanID,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get Plan: %w", err)
		}

		allowedPlanStatuses := []plan.PlanStatus{plan.DraftStatus, plan.ScheduledStatus, plan.ArchivedStatus}
		planStatus := p.Status()
		if !lo.Contains(allowedPlanStatuses, p.Status()) {
			return nil, fmt.Errorf("only Plans in %+v can be updated, but it has %s state", allowedPlanStatuses, planStatus)
		}

		err = s.adapter.DeletePhase(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to delete PlanPhase: %w", err)
		}

		logger.Debug("PlanPhase deleted")

		return nil, nil
	}
	_, err := transaction.Run(ctx, s.adapter, fn)

	return err
}

func (s service) GetPhase(ctx context.Context, params plan.GetPhaseInput) (*plan.Phase, error) {
	fn := func(ctx context.Context) (*plan.Phase, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid get PlanPhase params: %w", err)
		}

		logger := s.logger.With(
			"operation", "get",
			"namespace", params.Namespace,
			"phase.id", params.ID,
			"phase.key", params.Key,
		)

		logger.Debug("fetching Plan")

		phase, err := s.adapter.GetPhase(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to get PlanPhase: %w", err)
		}

		logger.Debug("PlanPhase fetched")

		return phase, nil
	}

	return transaction.Run(ctx, s.adapter, fn)
}

func (s service) UpdatePhase(ctx context.Context, params plan.UpdatePhaseInput) (*plan.Phase, error) {
	fn := func(ctx context.Context) (*plan.Phase, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid update PlanPhase params: %w", err)
		}

		logger := s.logger.With(
			"operation", "update",
			"namespace", params.Namespace,
			"plan.id", params.ID,
		)

		p, err := s.adapter.GetPlan(ctx, plan.GetPlanInput{
			NamespacedID: models.NamespacedID{
				Namespace: params.Namespace,
				ID:        params.PlanID,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get Plan: %w", err)
		}

		allowedPlanStatuses := []plan.PlanStatus{plan.DraftStatus, plan.ScheduledStatus}
		planStatus := p.Status()
		if !lo.Contains(allowedPlanStatuses, p.Status()) {
			return nil, fmt.Errorf("only Plans in %+v can be updated, but it has %s state", allowedPlanStatuses, planStatus)
		}

		if params.StartAfter != nil {
			for _, planPhase := range p.Phases {
				if planPhase.StartAfter == *params.StartAfter {
					return nil, fmt.Errorf("there is already a PlanPhase with the same StartAfter perdiod: %q", planPhase.Key)
				}
			}
		}

		logger.Debug("updating PlanPhase")

		phase, err := s.adapter.UpdatePhase(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to udpate PlanPhase: %w", err)
		}

		logger.Debug("PlanPhase updated")

		return phase, nil
	}

	return transaction.Run(ctx, s.adapter, fn)
}
