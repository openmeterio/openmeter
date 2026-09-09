package service

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (s *service) Migrate(ctx context.Context, request plansubscription.MigrateSubscriptionRequest) (plansubscription.SubscriptionChangeResponse, error) {
	var def plansubscription.SubscriptionChangeResponse

	// Let's fetch the current sub
	sub, err := s.SubscriptionService.Get(ctx, request.ID)
	if err != nil {
		return def, err
	}

	if sub.PlanRef == nil {
		return def, models.NewGenericValidationError(
			fmt.Errorf("subscription %s has no plan, cannot be migrated", request.ID.ID),
		)
	}

	// Let's fetch the version of the p we should migrate to
	p, err := s.getPlanByVersion(ctx, request.ID.Namespace, plansubscription.PlanRefInput{
		Key:     sub.PlanRef.Key,
		Version: request.TargetVersion,
	})
	if err != nil {
		return def, err
	}

	if p == nil {
		return def, fmt.Errorf("plan is nil")
	}

	if p.Version <= sub.PlanRef.Version {
		return def, models.NewGenericValidationError(
			fmt.Errorf("subscription %s is already at version %d, cannot migrate to version %d", request.ID.ID, sub.PlanRef.Version, p.Version),
		)
	}

	now := clock.Now()

	if p.DeletedAt != nil && !now.Before(*p.DeletedAt) {
		return def, models.NewGenericValidationError(
			fmt.Errorf("plan is deleted [namespace=%s, key=%s, version=%d, deleted_at=%s]",
				p.Namespace, p.Key, p.Version, p.DeletedAt),
		)
	}

	if !lo.Contains([]productcatalog.PlanStatus{
		productcatalog.PlanStatusActive,
		productcatalog.PlanStatusArchived,
	}, p.StatusAt(now)) {
		return def, models.NewGenericValidationError(
			fmt.Errorf("plan %s@%d is not active or archived at %s", p.Key, p.Version, now),
		)
	}

	if request.StartingPhase != nil {
		return def, models.NewGenericValidationError(fmt.Errorf("migration preserves the phase timeline; use subscription change to select a starting phase"))
	}
	if request.BillingAnchor != nil && !request.BillingAnchor.Equal(sub.BillingAnchor) {
		return def, models.NewGenericValidationError(fmt.Errorf("migration preserves the billing anchor; use subscription change to reset it"))
	}

	if request.RejectUnitConfig && p.HasUnitConfig() {
		return def, productcatalog.ErrUnitConfigNotRepresentable
	}

	pp := PlanFromPlan(*p)

	timing := lo.FromPtrOr(request.Timing, subscription.Timing{Enum: lo.ToPtr(subscription.TimingImmediate)})
	updated, err := s.WorkflowService.MigrateToPlan(ctx, subscriptionworkflow.MigrateSubscriptionWorkflowInput{
		SubscriptionID: request.ID,
		Plan:           pp,
		Timing:         timing,
	})
	if err != nil {
		return def, err
	}

	// Keep the existing response envelope: current is the pre-amendment snapshot,
	// next is the updated view of the same subscription.
	return plansubscription.SubscriptionChangeResponse{Current: sub, Next: updated}, nil
}
