package plansubscription

import (
	"context"
	"log/slog"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Adapter interface {
	// GetPlan returns the plan with the given key and version with all it's dependent resources.
	//
	// If the Plan is Not Found, it should return a PlanNotFoundError.
	GetVersion(ctx context.Context, namespace string, ref subscription.PlanRefInput) (subscription.Plan, error)
}

type PlanSubscriptionAdapterConfig struct {
	PlanService plan.Service
	Logger      *slog.Logger
}

type adapter struct {
	PlanSubscriptionAdapterConfig
}

var _ Adapter = &adapter{}

func NewPlanSubscriptionAdapter(config PlanSubscriptionAdapterConfig) Adapter {
	return &adapter{config}
}

func (a *adapter) GetVersion(ctx context.Context, namespace string, ref subscription.PlanRefInput) (subscription.Plan, error) {
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

	return &Plan{
		Plan: *p,
	}, nil
}
