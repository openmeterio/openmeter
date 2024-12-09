package subscriptionplan

import (
	"context"
	"log/slog"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type PlanSubscriptionAdapterConfig struct {
	PlanService plan.Service
	Logger      *slog.Logger
}

type PlanSubscriptionAdapter struct {
	PlanSubscriptionAdapterConfig
}

var _ subscription.PlanAdapter = &PlanSubscriptionAdapter{}

func NewSubscriptionPlanAdapter(config PlanSubscriptionAdapterConfig) subscription.PlanAdapter {
	return &PlanSubscriptionAdapter{config}
}

func (a *PlanSubscriptionAdapter) GetVersion(ctx context.Context, namespace string, ref subscription.PlanRefInput) (subscription.Plan, error) {
	planKey := ref.Key
	version := defaultx.WithDefault(ref.Version, 0) // plan service treats 0 as special case

	p, err := a.PlanService.GetPlan(ctx, plan.GetPlanInput{
		NamespacedID: models.NamespacedID{
			Namespace: namespace,
		},
		Key:     planKey,
		Version: version,
	})

	if _, ok := lo.ErrorsAs[plan.NotFoundError](err); ok {
		return nil, subscription.PlanNotFoundError{
			Key:     planKey,
			Version: version,
		}
	} else if err != nil {
		return nil, err
	}

	if p == nil {
		return nil, subscription.PlanNotFoundError{
			Key:     planKey,
			Version: version,
		}
	}

	return &SubscriptionPlan{
		Plan: *p,
	}, nil
}
