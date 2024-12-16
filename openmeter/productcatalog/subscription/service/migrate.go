package service

import (
	"context"
	"fmt"

	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
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
		return def, &models.GenericUserError{
			Message: fmt.Sprintf("Subscription %s has no plan, cannot be migrated", request.ID.ID),
		}
	}

	// Let's fetch the version of the plan we should migrate to
	plan, err := s.PlanAdapter.GetVersion(ctx, request.ID.Namespace, plansubscription.PlanRefInput{
		Key:     sub.PlanRef.Key,
		Version: &request.TargetVersion,
	})
	if err != nil {
		return def, err
	}

	// Then let's create the subscription from the plan
	curr, new, err := s.WorkflowService.ChangeToPlan(ctx, request.ID, subscription.ChangeSubscriptionWorkflowInput{
		ActiveFrom:     clock.Now(),
		AnnotatedModel: sub.AnnotatedModel,
		Name:           sub.Name,
		Description:    sub.Description,
	}, plan)
	if err != nil {
		return def, err
	}

	return plansubscription.SubscriptionChangeResponse{
		Current: curr,
		New:     new,
	}, nil
}
