package testutils

import (
	"context"
	"log/slog"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/subscription/service"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/models"
)

// TODO: we can get rid of this
type PlanSubscriptionAdapter interface {
	// GetPlan returns the plan for the Ref with all it's dependent resources.
	//
	// If the Plan is Not Found, it should return a PlanNotFoundError.
	GetVersion(ctx context.Context, namespace string, ref plansubscription.PlanRefInput) (subscription.Plan, error)

	// Converts a plan.CreatePlanInput to a subscription.Plan.
	FromInput(ctx context.Context, namespace string, input plan.CreatePlanInput) (subscription.Plan, error)
}

type PlanSubscriptionAdapterConfig struct {
	PlanService plan.Service
	Logger      *slog.Logger
}

type adapter struct {
	PlanSubscriptionAdapterConfig
}

var _ PlanSubscriptionAdapter = &adapter{}

func NewPlanSubscriptionAdapter(config PlanSubscriptionAdapterConfig) PlanSubscriptionAdapter {
	return &adapter{config}
}

func (a *adapter) GetVersion(ctx context.Context, namespace string, ref plansubscription.PlanRefInput) (subscription.Plan, error) {
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

	pp, err := p.AsProductCatalogPlan(clock.Now())
	if err != nil {
		return nil, err
	}

	return &plansubscription.Plan{
		Plan: pp,
		Ref:  &p.NamespacedID,
	}, nil
}

func (a *adapter) FromInput(ctx context.Context, namespace string, input plan.CreatePlanInput) (subscription.Plan, error) {
	return service.PlanFromPlanInput(input)
}
