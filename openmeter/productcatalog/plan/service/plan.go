package service

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/pkg/clock"
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

func (s service) expandFeatures(ctx context.Context, namespace string, rateCards *productcatalog.RateCards) error {
	if rateCards == nil || len(*rateCards) == 0 {
		return nil
	}

	rateCardFeatures := make(map[string]*feature.Feature)
	rateCardFeatureKeys := make([]string, 0)
	for _, rateCard := range *rateCards {
		if rateCardFeature := rateCard.Feature(); rateCardFeature != nil {
			rateCardFeatures[rateCardFeature.Key] = rateCardFeature
			rateCardFeatureKeys = append(rateCardFeatureKeys, rateCardFeature.Key)
		}
	}

	if len(rateCardFeatureKeys) == 0 {
		return nil
	}

	featureList, err := s.feature.ListFeatures(ctx, feature.ListFeaturesParams{
		IDsOrKeys: rateCardFeatureKeys,
		Namespace: namespace,
		Page: pagination.Page{
			PageSize:   len(rateCardFeatures),
			PageNumber: 1,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to list Features for RateCards: %w", err)
	}

	// Update features in-place or return error if
	visited := make([]string, 0)
	for _, feat := range featureList.Items {
		if rcFeat, ok := rateCardFeatures[feat.Key]; ok {
			*rcFeat = feat

			visited = append(visited, feat.Key)
		}
	}

	if len(rateCardFeatures) != len(visited) {
		missing, r := lo.Difference(rateCardFeatureKeys, visited)
		missing = append(missing, r...)

		return productcatalog.NewValidationError(fmt.Errorf("non-existing Features: %+v", missing))
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

		// Check if there is already a Plan with the same Key
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

		// If there are Plan versions with the same Key do:
		// * check their statuses to ensure that new plan with the same Key is created only
		//   if there is no version in Draft status
		// * calculate the version number for the new Plan based by incrementing the last version
		if len(allVersions.Items) >= 0 {
			for _, p := range allVersions.Items {
				if p.DeletedAt == nil && p.Status() == productcatalog.DraftStatus {
					return nil, productcatalog.NewValidationError(
						fmt.Errorf("only a single draft version is allowed for Plan"),
					)
				}

				if p.Version >= params.Version {
					params.Version = p.Version + 1
				}
			}
		}

		logger.Debug("creating Plan")

		if len(params.Phases) > 0 {
			for _, phase := range params.Phases {
				if err := s.expandFeatures(ctx, params.Namespace, &phase.RateCards); err != nil {
					return nil, fmt.Errorf("failed to expand Features for RateCards in PlanPhase: %w", err)
				}
			}
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

		logger.Debug("deleting Plan")

		p, err := s.adapter.GetPlan(ctx, plan.GetPlanInput{
			NamespacedID: models.NamespacedID{
				Namespace: params.Namespace,
				ID:        params.ID,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get Plan: %w", err)
		}

		allowedPlanStatuses := []productcatalog.PlanStatus{
			productcatalog.ArchivedStatus,
			productcatalog.ScheduledStatus,
			productcatalog.DraftStatus,
		}
		planStatus := p.Status()
		if !lo.Contains(allowedPlanStatuses, p.Status()) {
			return nil, productcatalog.NewValidationError(
				fmt.Errorf("only Plans in %+v can be deleted, but it has %s state", allowedPlanStatuses, planStatus),
			)
		}

		err = s.adapter.DeletePlan(ctx, params)
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

		if params.Phases != nil && len(*params.Phases) > 0 {
			for _, phase := range *params.Phases {
				if err := s.expandFeatures(ctx, params.Namespace, &phase.RateCards); err != nil {
					return nil, fmt.Errorf("failed to expand Features for RateCards in PlanPhase: %w", err)
				}
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

		allowedPlanStatuses := []productcatalog.PlanStatus{
			productcatalog.DraftStatus,
			productcatalog.ScheduledStatus,
		}
		planStatus := p.Status()
		if !lo.Contains(allowedPlanStatuses, p.Status()) {
			return nil, productcatalog.NewValidationError(
				fmt.Errorf("only Plans in %+v can be updated, but it has %s state", allowedPlanStatuses, planStatus),
			)
		}

		logger.Debug("updating plan")

		// NOTE(chrisgacsal): we only allow updating the state of the Plan via Publish/Archive,
		// therefore the EffectivePeriod attribute must be zeroed before updating the Plan.
		params.EffectivePeriod = productcatalog.EffectivePeriod{}

		p, err = s.adapter.UpdatePlan(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to udpate Plan: %w", err)
		}

		logger.Debug("Plan updated")

		return p, nil
	}

	return transaction.Run(ctx, s.adapter, fn)
}

// PublishPlan
// TODO(chrisgacsal): add support for scheduling Plan versions in the future.
// In order to ensure that there are not time gaps where no active version of a Plan is available
// the EffectivePeriod must be validated/updated with the surrounding Plans(N-1, N+1) if they exist.
// If updating the EffectivePeriod for surrounding Plans violates constraints, return an validation error,
// otherwise adjust their schedule accordingly.
// IMPORTANT: this might need to be an optional action which must be only performed with the users consent as it has side-effects.
// In other words, modify the surrounding Plans only if the user is allowed it otherwise return a validation error
// in case the lifecycle of the Plan is not continuous (there are time gaps between versions).
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

		pp, err := p.AsProductCatalogPlan(clock.Now())
		if err != nil {
			return nil, fmt.Errorf("failed to convert Plan to ProductCatalog Plan: %w", err)
		}

		// First, let's validate that the Subscription can successfully be created from this Plan

		if err := pp.ValidForCreatingSubscriptions(); err != nil {
			return nil, productcatalog.NewValidationError(fmt.Errorf("invalid Plan for creating subscriptions: %w", err))
		}

		// Second, let's validate that the plan status and the version history is correct
		allowedPlanStatuses := []productcatalog.PlanStatus{
			productcatalog.DraftStatus,
			productcatalog.ScheduledStatus,
		}
		planStatus := pp.Status()
		if !lo.Contains(allowedPlanStatuses, pp.Status()) {
			return nil, productcatalog.NewValidationError(
				fmt.Errorf("invalid Plan: only Plans in %+v can be published/rescheduled, but it has %s state", allowedPlanStatuses, planStatus),
			)
		}

		// Find and archive Plan version with plan.ActiveStatus if there is one. Only perform lookup if
		// the Plan to be published has higher version then 1 meaning that it has previous versions,
		// otherwise skip this step.
		if pp.Version > 1 {
			activePlan, err := s.adapter.GetPlan(ctx, plan.GetPlanInput{
				NamespacedID: models.NamespacedID{
					Namespace: params.Namespace,
				},
				Key: pp.Key,
			})
			if err != nil {
				if !plan.IsNotFound(err) {
					return nil, fmt.Errorf("failed to get Plan with active status: %w", err)
				}
			}

			if activePlan != nil && params.EffectiveFrom != nil {
				_, err = s.ArchivePlan(ctx, plan.ArchivePlanInput{
					NamespacedID: models.NamespacedID{
						Namespace: activePlan.Namespace,
						ID:        activePlan.ID,
					},
					EffectiveTo: lo.FromPtr(params.EffectiveFrom),
				})
				if err != nil {
					return nil, fmt.Errorf("failed to archive plan with active status: %w", err)
				}
			}
		}

		// Publish new Plan version

		input := plan.UpdatePlanInput{
			NamespacedID: params.NamespacedID,
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

		activeStatuses := []productcatalog.PlanStatus{productcatalog.ActiveStatus}
		status := p.Status()
		if !lo.Contains(activeStatuses, status) {
			return nil, productcatalog.NewValidationError(
				fmt.Errorf("only Plans in %+v can be archived, but it is in %s state", activeStatuses, status),
			)
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
			EffectivePeriod: productcatalog.EffectivePeriod{
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
			OrderBy:        plan.OrderByVersion,
			Order:          plan.OrderAsc,
			Namespaces:     []string{params.Namespace},
			IDs:            []string{params.ID},
			Keys:           []string{params.Key},
			IncludeDeleted: true,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list all versions of the Plan: %w", err)
		}

		if len(allVersions.Items) == 0 {
			return nil, productcatalog.NewValidationError(
				fmt.Errorf("no versions available for this plan"),
			)
		}

		// Generate source plan filter from input parameters

		// planFilterFunc is a filter function which returns tuple where the first boolean means that
		// there is a match while the second tells the caller to stop further invocations as there is an exact match.
		type planFilterFunc func(plan plan.Plan) (match bool, stop bool)

		sourcePlanFilterFunc := func() planFilterFunc {
			switch {
			case params.ID != "":
				return func(p plan.Plan) (match bool, stop bool) {
					if p.Namespace == params.Namespace && p.ID == params.ID {
						return true, true
					}

					return false, false
				}
			case params.Key != "" && params.Version == 0:
				return func(p plan.Plan) (match bool, stop bool) {
					return p.Namespace == params.Namespace && p.Key == params.Key, false
				}
			default:
				return func(p plan.Plan) (match bool, stop bool) {
					if p.Namespace == params.Namespace && p.Key == params.Key && p.Version == params.Version {
						return true, true
					}

					return false, false
				}
			}
		}()

		var sourcePlan *plan.Plan

		nextVersion := 1
		var match, stop bool
		for _, p := range allVersions.Items {
			if p.DeletedAt == nil && p.Status() == productcatalog.DraftStatus {
				return nil, productcatalog.NewValidationError(
					fmt.Errorf("only a single draft version is allowed for Plan"),
				)
			}

			if !stop {
				match, stop = sourcePlanFilterFunc(p)
				if match {
					sourcePlan = &p
				}
			}

			if p.Version >= nextVersion {
				nextVersion = p.Version + 1
			}
		}

		if sourcePlan == nil {
			return nil, productcatalog.NewValidationError(
				fmt.Errorf("no versions available for plan to use as source for next draft version"),
			)
		}

		nextPlan, err := s.adapter.CreatePlan(ctx, plan.CreatePlanInput{
			NamespacedModel: models.NamespacedModel{
				Namespace: sourcePlan.Namespace,
			},
			Plan: productcatalog.Plan{
				PlanMeta: productcatalog.PlanMeta{
					Key:         sourcePlan.Key,
					Version:     nextVersion,
					Name:        sourcePlan.Name,
					Description: sourcePlan.Description,
					Metadata:    sourcePlan.Metadata,
					Currency:    sourcePlan.Currency,
				},
				Phases: func() []productcatalog.Phase {
					var phases []productcatalog.Phase

					for _, phase := range sourcePlan.Phases {
						phases = append(phases, productcatalog.Phase{
							PhaseMeta: phase.PhaseMeta,
							RateCards: phase.RateCards,
						})
					}

					return phases
				}(),
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create new version of a Plan: %w", err)
		}

		return nextPlan, nil
	}

	return transaction.Run(ctx, s.adapter, fn)
}
