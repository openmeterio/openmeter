package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func (s service) ListPlanAddons(ctx context.Context, params planaddon.ListPlanAddonsInput) (pagination.PagedResponse[planaddon.PlanAddon], error) {
	fn := func(ctx context.Context) (pagination.PagedResponse[planaddon.PlanAddon], error) {
		if err := params.Validate(); err != nil {
			return pagination.PagedResponse[planaddon.PlanAddon]{}, fmt.Errorf("invalid list plan add-on assignment params: %w", err)
		}

		return s.adapter.ListPlanAddons(ctx, params)
	}

	return fn(ctx)
}

func (s service) CreatePlanAddon(ctx context.Context, params planaddon.CreatePlanAddonInput) (*planaddon.PlanAddon, error) {
	fn := func(ctx context.Context) (*planaddon.PlanAddon, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid create plan add-on assignment params [namespace=%s plan.id=%s addon.id=%s]: %w",
				params.Namespace, params.PlanID, params.AddonID, err)
		}

		logger := s.logger.With(
			"operation", "create",
			"namespace", params.Namespace,
			"plan.id", params.PlanID,
			"addon.id", params.AddonID,
		)

		// Check whether plan add-on assignment already exists or not
		planAddons, err := s.ListPlanAddons(ctx, planaddon.ListPlanAddonsInput{
			Namespaces: []string{params.Namespace},
			PlanIDs:    []string{params.PlanID},
			AddonIDs:   []string{params.AddonID},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get plan add-on assignment: %w", err)
		}

		if len(planAddons.Items) > 0 {
			return nil, models.NewGenericConflictError(
				fmt.Errorf("plan add-on assignment already exists [namespace=%s plan.id=%s addon.id=%s]: %w",
					params.Namespace, params.PlanID, params.AddonID, err),
			)
		}

		logger.Debug("validating plan add-on assignment")

		p, err := s.plan.GetPlan(ctx, plan.GetPlanInput{
			NamespacedID: models.NamespacedID{
				Namespace: params.Namespace,
				ID:        params.PlanID,
			},
		})
		if err != nil {
			if notFound := &(plan.NotFoundError{}); errors.As(err, &notFound) {
				return nil, models.NewGenericNotFoundError(err)
			}

			return nil, fmt.Errorf("failed to get plan [namespace=%s plan.id=%s]: %w",
				params.Namespace, params.PlanID, err)
		}

		a, err := s.addon.GetAddon(ctx, addon.GetAddonInput{
			NamespacedID: models.NamespacedID{
				Namespace: params.Namespace,
				ID:        params.AddonID,
			},
		})
		if err != nil {
			if notFound := &(plan.NotFoundError{}); errors.As(err, &notFound) {
				return nil, models.NewGenericNotFoundError(err)
			}

			return nil, fmt.Errorf("failed to get add-on [namespace=%s addon.id=%s]: %w",
				params.Namespace, params.AddonID, err)
		}

		planAddonAssignment := productcatalog.PlanAddon{
			PlanAddonMeta: productcatalog.PlanAddonMeta{
				Metadata:    params.Metadata,
				Annotations: params.Annotations,
				PlanAddonConfig: productcatalog.PlanAddonConfig{
					FromPlanPhase: params.FromPlanPhase,
					MaxQuantity:   params.MaxQuantity,
				},
			},
			Plan:  p.AsProductCatalogPlan2(),
			Addon: a.AsProductCatalogAddon(),
		}

		if err = planAddonAssignment.Validate(); err != nil {
			return nil, models.NewGenericValidationError(
				fmt.Errorf("invalid plan add-on assignment [namespace=%s plan.id=%s addon.id=%s]: %w",
					params.Namespace, params.PlanID, params.AddonID, err),
			)
		}

		logger.Debug("creating plan add-on assignment")

		planAddon, err := s.adapter.CreatePlanAddon(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to create plan add-on assignment [namespace=%s plan.id=%s addon.id=%s]: %w",
				params.Namespace, params.PlanID, params.AddonID, err)
		}

		logger.With("planaddon.id", planAddon.ID).Debug("plan add-on assignment created")

		// Emit created event
		event := planaddon.NewPlanAddonCreateEvent(ctx, planAddon)
		if err = s.publisher.Publish(ctx, event); err != nil {
			return nil, fmt.Errorf("failed to publish plan add-on assignment created event [namespace=%s plan.id=%s addon.id=%s]: %w",
				params.Namespace, params.PlanID, params.AddonID, err)
		}

		return planAddon, nil
	}

	return transaction.Run(ctx, s.adapter, fn)
}

func (s service) DeletePlanAddon(ctx context.Context, params planaddon.DeletePlanAddonInput) error {
	fn := func(ctx context.Context) error {
		if err := params.Validate(); err != nil {
			return fmt.Errorf("invalid delete plan add-on assignment params [namespace=%s plan.id=%s addon.id=%s]: %w",
				params.Namespace, params.PlanID, params.AddonID, err)
		}

		logger := s.logger.With(
			"operation", "delete",
			"namespace", params.Namespace,
			"planaddon.id", params.ID,
		)

		logger.Debug("deleting plan add-on assignment")

		// Get the plan add-on assignment to check whether it is already deleted or not
		planAddon, err := s.adapter.GetPlanAddon(ctx, planaddon.GetPlanAddonInput{
			NamespacedModel: models.NamespacedModel{
				Namespace: params.Namespace,
			},
			ID:           params.ID,
			PlanIDOrKey:  params.PlanID,
			AddonIDOrKey: params.AddonID,
		})
		if err != nil {
			if notFound := &(planaddon.NotFoundError{}); errors.As(err, &notFound) {
				return models.NewGenericNotFoundError(err)
			}

			return fmt.Errorf("failed to get plan add-on assignment [namespace=%s plan.id=%s addon.id=%s]: %w",
				params.Namespace, params.PlanID, params.AddonID, err)
		}

		if planAddon.DeletedAt != nil {
			return nil
		}

		// Delete the plan add-on assignment
		err = s.adapter.DeletePlanAddon(ctx, params)
		if err != nil {
			return fmt.Errorf("failed to delete plan add-on assignment [namespace=%s plan.id=%s addon.id=%s]: %w",
				params.Namespace, params.PlanID, params.AddonID, err)
		}

		logger.Debug("plan add-on assignment deleted")

		// Get the deleted add-on to emit the event
		planAddon, err = s.adapter.GetPlanAddon(ctx, planaddon.GetPlanAddonInput{
			NamespacedModel: models.NamespacedModel{
				Namespace: params.Namespace,
			},
			ID: planAddon.ID,
		})
		if err != nil {
			if notFound := &(planaddon.NotFoundError{}); errors.As(err, &notFound) {
				return models.NewGenericNotFoundError(err)
			}

			return fmt.Errorf("failed to get deleted plan add-on assignment: %w", err)
		}

		// Emit deleted event
		event := planaddon.NewPlanAddonDeleteEvent(ctx, planAddon)
		if err = s.publisher.Publish(ctx, event); err != nil {
			return fmt.Errorf("failed to publish plan add-on assignment deleted event: %w", err)
		}

		return nil
	}

	return transaction.RunWithNoValue(ctx, s.adapter, fn)
}

func (s service) GetPlanAddon(ctx context.Context, params planaddon.GetPlanAddonInput) (*planaddon.PlanAddon, error) {
	fn := func(ctx context.Context) (*planaddon.PlanAddon, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid get plan add-on assignment params [namespace=%s planaddon.id=%s plan.id=%s addon.id=%s]: %w",
				params.Namespace, params.ID, params.PlanIDOrKey, params.AddonIDOrKey, err)
		}

		logger := s.logger.With(
			"operation", "get",
			"namespace", params.Namespace,
			"planaddon.id", params.ID,
			"plan.id", params.PlanIDOrKey,
			"addon.id", params.AddonIDOrKey,
		)

		logger.Debug("fetching plan add-on assignment")

		planAddon, err := s.adapter.GetPlanAddon(ctx, params)
		if err != nil {
			if notFound := &(planaddon.NotFoundError{}); errors.As(err, &notFound) {
				return nil, models.NewGenericNotFoundError(err)
			}

			return nil, fmt.Errorf("failed to get plan add-on assignment [namespace=%s planaddon.id=%s plan.id=%s addon.id=%s]: %w",
				params.Namespace, params.ID, params.PlanIDOrKey, params.AddonIDOrKey, err)
		}

		logger.Debug("plan add-on assignment fetched")

		return planAddon, nil
	}

	return fn(ctx)
}

func (s service) UpdatePlanAddon(ctx context.Context, params planaddon.UpdatePlanAddonInput) (*planaddon.PlanAddon, error) {
	fn := func(ctx context.Context) (*planaddon.PlanAddon, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid update plan add-on assignment params [namespace=%s plan.id=%s addon.id=%s]: %w",
				params.Namespace, params.PlanID, params.AddonID, err)
		}

		logger := s.logger.With(
			"operation", "update",
			"namespace", params.Namespace,
			"planaddon.id", params.ID,
			"plan.id", params.PlanID,
			"addon.id", params.AddonID,
		)

		logger.Debug("updating plan add-on assignment")

		planAddon, err := s.adapter.GetPlanAddon(ctx, planaddon.GetPlanAddonInput{
			NamespacedModel: models.NamespacedModel{
				Namespace: params.Namespace,
			},
			ID:           params.ID,
			PlanIDOrKey:  params.PlanID,
			AddonIDOrKey: params.AddonID,
		})
		if err != nil {
			if notFound := &(planaddon.NotFoundError{}); errors.As(err, &notFound) {
				return nil, models.NewGenericNotFoundError(err)
			}

			return nil, fmt.Errorf("failed to get plan add-on assignment [namespace=%s plan.id=%s addon.id=%s]: %w",
				params.Namespace, params.PlanID, params.AddonID, err)
		}

		logger.Debug("validating plan add-on assignment")

		p, err := s.plan.GetPlan(ctx, plan.GetPlanInput{
			NamespacedID: models.NamespacedID{
				Namespace: params.Namespace,
				ID:        planAddon.Plan.ID,
			},
		})
		if err != nil {
			if err != nil {
				if notFound := &(plan.NotFoundError{}); errors.As(err, &notFound) {
					return nil, models.NewGenericNotFoundError(err)
				}

				return nil, fmt.Errorf("failed to get plan [namespace=%s plan.id=%s]: %w",
					params.Namespace, params.PlanID, err)
			}
		}

		a, err := s.addon.GetAddon(ctx, addon.GetAddonInput{
			NamespacedID: models.NamespacedID{
				Namespace: params.Namespace,
				ID:        planAddon.Addon.ID,
			},
		})
		if err != nil {
			if notFound := &(addon.NotFoundError{}); errors.As(err, &notFound) {
				return nil, models.NewGenericNotFoundError(err)
			}

			return nil, fmt.Errorf("failed to get add-on [namespace=%s addon.id=%s]: %w",
				params.Namespace, params.AddonID, err)
		}

		planAddonAssignment := productcatalog.PlanAddon{
			PlanAddonMeta: productcatalog.PlanAddonMeta{
				Metadata:    lo.FromPtr(params.Metadata),
				Annotations: lo.FromPtr(params.Annotations),
				PlanAddonConfig: productcatalog.PlanAddonConfig{
					FromPlanPhase: lo.FromPtr(params.FromPlanPhase),
					MaxQuantity:   params.MaxQuantity,
				},
			},
			Plan:  p.AsProductCatalogPlan2(),
			Addon: a.AsProductCatalogAddon(),
		}

		if err = planAddonAssignment.Validate(); err != nil {
			return nil, models.NewGenericValidationError(
				fmt.Errorf("invalid plan add-on assignment [namespace=%s plan.id=%s addon.id=%s]: %w",
					params.Namespace, params.PlanID, params.AddonID, err),
			)
		}

		planAddon, err = s.adapter.UpdatePlanAddon(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to udpate plan add-on assignment [namespace=%s plan.id=%s addon.id=%s]: %w",
				params.Namespace, params.PlanID, params.AddonID, err)
		}

		logger.Debug("plan add-on assignment updated")

		// Emit updated event
		event := planaddon.NewPlanAddonUpdateEvent(ctx, planAddon)
		if err = s.publisher.Publish(ctx, event); err != nil {
			return nil, fmt.Errorf("failed to publish plan add-on assignment updated event [namespace=%s plan.id=%s addon.id=%s]: %w",
				params.Namespace, params.PlanID, params.AddonID, err)
		}

		return planAddon, nil
	}

	return transaction.Run(ctx, s.adapter, fn)
}
