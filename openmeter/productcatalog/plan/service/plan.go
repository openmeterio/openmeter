package service

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func (s service) ListPlans(ctx context.Context, params plan.ListPlansInput) (pagination.PagedResponse[plan.Plan], error) {
	fn := func(ctx context.Context) (pagination.PagedResponse[plan.Plan], error) {
		if err := params.Validate(); err != nil {
			return pagination.PagedResponse[plan.Plan]{}, fmt.Errorf("invalid list Plans params: %w", err)
		}

		return s.adapter.ListPlans(ctx, params)
	}

	return transaction.Run(ctx, s.adapter, fn)
}

func (s service) expandFeatures(ctx context.Context, phases []plan.Phase) error {
	if len(phases) > 0 {
		return nil
	}

	for _, phase := range phases {
		rateCardFeatures := make(map[string]*feature.Feature)

		for _, rateCard := range phase.RateCards {
			if rateCardFeature := rateCard.Feature(); rateCardFeature != nil {
				rateCardFeatures[rateCardFeature.Key] = rateCardFeature
			}
		}

		featureList, err := s.feature.ListFeatures(ctx, feature.ListFeaturesParams{
			IDsOrKeys: lo.Keys(rateCardFeatures),
			Namespace: phase.Namespace,
			Page: pagination.Page{
				PageSize:   len(rateCardFeatures),
				PageNumber: 1,
			},
		})
		if err != nil {
			return fmt.Errorf("failed to list Features for RateCard: %w", err)
		}

		// Update features in-place or return error if
		visited := make(map[string]struct{})
		for _, feat := range featureList.Items {
			if rcFeat, ok := rateCardFeatures[feat.Key]; ok {
				*rcFeat = feat

				visited[feat.Key] = struct{}{}
			}
		}

		if len(rateCardFeatures) != len(visited) {
			missing, r := lo.Difference(lo.Keys(rateCardFeatures), lo.Keys(visited))
			missing = append(missing, r...)

			return fmt.Errorf("non-existing Features: %+v", missing)
		}
	}

	return nil
}

func (s service) CreatePlan(ctx context.Context, params plan.CreatePlanInput) (*plan.Plan, error) {
	fn := func(ctx context.Context) (*plan.Plan, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid create Plan params: %w", err)
		}

		logger := s.logger.With(
			"operation", "create",
			"namespace", params.Namespace,
			"plan.key", params.Key,
		)

		logger.Debug("creating Plan")

		if err := s.expandFeatures(ctx, params.Phases); err != nil {
			return nil, fmt.Errorf("failed to get Feature for RateCards: %w", err)
		}

		p, err := s.adapter.CreatePlan(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to create Plan: %w", err)
		}

		logger.With("plan.id", p.ID).Debug("Plan created")

		return p, nil
	}

	return transaction.Run(ctx, s.adapter, fn)
}

func (s service) DeletePlan(ctx context.Context, params plan.DeletePlanInput) error {
	fn := func(ctx context.Context) (interface{}, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid delete Plan params: %w", err)
		}

		logger := s.logger.With(
			"operation", "delete",
			"namespace", params.Namespace,
			"plan.id", params.ID,
		)

		// TODO(chrisgacsal): add check which makes sure that Plans with active Subscriptions are not deleted.

		logger.Debug("deleting Plan")

		err := s.adapter.DeletePlan(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to delete Plan: %w", err)
		}

		logger.Debug("Plan deleted")

		return nil, nil
	}

	_, err := transaction.Run(ctx, s.adapter, fn)

	return err
}

func (s service) GetPlan(ctx context.Context, params plan.GetPlanInput) (*plan.Plan, error) {
	fn := func(ctx context.Context) (*plan.Plan, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid get Plan params: %w", err)
		}

		logger := s.logger.With(
			"operation", "get",
			"namespace", params.Namespace,
			"plan.id", params.ID,
			"plan.key", params.Key,
			"plan.version", params.Version,
		)

		logger.Debug("fetching Plan")

		p, err := s.adapter.GetPlan(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to get Plan: %w", err)
		}

		logger.Debug("Plan fetched")

		return p, nil
	}

	return transaction.Run(ctx, s.adapter, fn)
}

func (s service) UpdatePlan(ctx context.Context, params plan.UpdatePlanInput) (*plan.Plan, error) {
	fn := func(ctx context.Context) (*plan.Plan, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid update Plan params: %w", err)
		}

		logger := s.logger.With(
			"operation", "update",
			"namespace", params.Namespace,
			"plan.id", params.ID,
		)

		logger.Debug("updating Plan")

		if params.Phases != nil {
			if err := s.expandFeatures(ctx, *params.Phases); err != nil {
				return nil, fmt.Errorf("failed to get Feature for RateCards: %w", err)
			}
		}

		p, err := s.adapter.GetPlan(ctx, plan.GetPlanInput{
			NamespacedID: models.NamespacedID{
				Namespace: params.Namespace,
				ID:        params.ID,
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

		logger.Debug("updating plan")

		// NOTE(chrisgacsal): we only allow updating the state of the Plan via Publish/Archive,
		// therefore the EffectivePeriod attribute must be zeroed before updating the Plan.
		params.EffectivePeriod = plan.EffectivePeriod{}

		p, err = s.adapter.UpdatePlan(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to udpate Plan: %w", err)
		}

		logger.Debug("Plan updated")

		return p, nil
	}

	return transaction.Run(ctx, s.adapter, fn)
}

func (s service) PublishPlan(ctx context.Context, params plan.PublishPlanInput) (*plan.Plan, error) {
	fn := func(ctx context.Context) (*plan.Plan, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid publish Plan params: %w", err)
		}

		logger := s.logger.With(
			"operation", "publish",
			"namespace", params.Namespace,
			"plan.id", params.ID,
		)

		logger.Debug("publishing Plan")

		p, err := s.adapter.GetPlan(ctx, plan.GetPlanInput{
			NamespacedID: models.NamespacedID{
				Namespace: params.Namespace,
				ID:        params.ID,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get Plan: %w", err)
		}

		allowedPlanStatuses := []plan.PlanStatus{plan.DraftStatus, plan.ScheduledStatus}
		planStatus := p.Status()
		if !lo.Contains(allowedPlanStatuses, p.Status()) {
			return nil, fmt.Errorf("only Plans in %+v can be published/rescheduled, but it has %s state", allowedPlanStatuses, planStatus)
		}

		// TODO(chrisgacsal): in order to ensure that there are not time gaps where no active version of a Plan is available
		// the EffectivePeriod must be validated/updated with the surrounding Plans(N-1, N+1) if they exist.
		// If updating the EffectivePeriod for surrounding Plans violates constraints, return an validation error,
		// otherwise adjust their schedule accordingly.
		// IMPORTANT: this should be an optional action which must be only performed with the users consent as it has side-effects.
		// In other words, modify the surrounding Plans only if the user is allowed it otherwise return a validation error
		// in case the lifecycle of the Plan is not continuous (there are time gaps between versions).

		input := plan.UpdatePlanInput{
			NamespacedID: models.NamespacedID{
				Namespace: p.Namespace,
				ID:        p.ID,
			},
		}

		if params.EffectiveFrom != nil {
			input.EffectiveFrom = lo.ToPtr(params.EffectiveFrom.UTC())
		}

		if params.EffectiveTo != nil {
			input.EffectiveTo = lo.ToPtr(params.EffectiveTo.UTC())
		}

		p, err = s.adapter.UpdatePlan(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to publish Plan: %w", err)
		}

		logger.Debug("Plan published")

		return p, nil
	}

	return transaction.Run(ctx, s.adapter, fn)
}

func (s service) ArchivePlan(ctx context.Context, params plan.ArchivePlanInput) (*plan.Plan, error) {
	fn := func(ctx context.Context) (*plan.Plan, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid archive Plan params: %w", err)
		}

		logger := s.logger.With(
			"operation", "archive",
			"namespace", params.Namespace,
			"plan.id", params.ID,
		)

		logger.Debug("archiving Plan")

		p, err := s.adapter.GetPlan(ctx, plan.GetPlanInput{
			NamespacedID: models.NamespacedID{
				Namespace: params.Namespace,
				ID:        params.ID,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get Plan: %w", err)
		}

		activeStatuses := []plan.PlanStatus{plan.ActiveStatus}
		status := p.Status()
		if !lo.Contains(activeStatuses, status) {
			return nil, fmt.Errorf("only Plans in %+v can be archived, but it is in %s state", activeStatuses, status)
		}

		// TODO(chrisgacsal): in order to ensure that there are not time gaps where no active version of a Plan is available
		// the EffectivePeriod must be validated/updated with the next Plan(N+1) if exists.
		// If updating the EffectivePeriod for next Plan violates constraints, return validation error, otherwise adjust
		// their schedule accordingly.
		// IMPORTANT: this should be an optional action which must be only performed with the users consent as it has side-effects.
		// In other words, modify the surrounding Plans only if the user is allowed it otherwise return a validation error
		// in case the lifecycle of the Plan is not continuous (there are time gaps between versions).

		p, err = s.adapter.UpdatePlan(ctx, plan.UpdatePlanInput{
			NamespacedID: models.NamespacedID{
				Namespace: p.Namespace,
				ID:        p.ID,
			},
			EffectivePeriod: plan.EffectivePeriod{
				EffectiveTo: lo.ToPtr(params.EffectiveTo.UTC()),
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to archive Plan: %w", err)
		}

		logger.Debug("Plan archived")

		return p, nil
	}

	return transaction.Run(ctx, s.adapter, fn)
}

func (s service) NextPlan(ctx context.Context, params plan.NextPlanInput) (*plan.Plan, error) {
	fn := func(ctx context.Context) (*plan.Plan, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid next version Plan params: %w", err)
		}

		logger := s.logger.With(
			"operation", "next",
			"namespace", params.Namespace,
			"plan.id", params.ID,
			"plan.key", params.Key,
			"plan.version", params.Version,
		)

		logger.Debug("creating new version of a Plan")

		// Fetch all version of a plan to find the one to be used as source and also to calculate the next version number.
		allVersions, err := s.adapter.ListPlans(ctx, plan.ListPlansInput{
			Page: pagination.Page{
				PageSize:   1000,
				PageNumber: 1,
			},
			OrderBy:        plan.OrderByVersion,
			Order:          plan.OrderAsc,
			Namespaces:     []string{params.Namespace},
			Keys:           []string{params.Key},
			IncludeDeleted: true,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list all versions of the Plan: %w", err)
		}

		if len(allVersions.Items) == 0 {
			return nil, fmt.Errorf("no versions available for this plan")
		}

		// Generate source plan filter from input parameters
		planFilter := func() func(plan plan.Plan) bool {
			switch {
			case params.ID != "":
				return func(p plan.Plan) bool {
					return p.Namespace == params.Namespace && p.ID == params.ID
				}
			case params.Key != "" && params.Version == 0:
				return func(p plan.Plan) bool {
					return p.Namespace == params.Namespace && p.Key == params.Key && p.Status() == plan.ActiveStatus
				}
			default:
				return func(p plan.Plan) bool {
					return p.Namespace == params.Namespace && p.Key == params.Key && p.Version == params.Version
				}
			}
		}()

		var sourcePlan *plan.Plan

		nextVersion := 1
		for _, p := range allVersions.Items {
			if sourcePlan == nil && planFilter(p) {
				sourcePlan = &p
			}

			if p.Version >= nextVersion {
				nextVersion = p.Version + 1
			}
		}

		if sourcePlan == nil {
			return nil, fmt.Errorf("no versions available for plan to use as source for next draft version")
		}

		nextPlan, err := s.adapter.CreatePlan(ctx, plan.CreatePlanInput{
			NamespacedModel: models.NamespacedModel{
				Namespace: sourcePlan.Namespace,
			},
			Key:         sourcePlan.Key,
			Version:     nextVersion,
			Name:        sourcePlan.Name,
			Description: sourcePlan.Description,
			Metadata:    sourcePlan.Metadata,
			Currency:    sourcePlan.Currency,
			Phases:      sourcePlan.Phases,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create new version of a Plan: %w", err)
		}

		return nextPlan, nil
	}

	return transaction.Run(ctx, s.adapter, fn)
}
