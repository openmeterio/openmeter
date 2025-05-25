package service

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/models"
)

// TODO: this method is mostly redundant if the APIs are matched
func (s *service) getPlanByVersion(ctx context.Context, namespace string, ref plansubscription.PlanRefInput) (*plan.Plan, error) {
	planKey := ref.Key
	version := defaultx.WithDefault(ref.Version, 0) // plan service treats 0 as special case

	p, err := s.PlanService.GetPlan(ctx, plan.GetPlanInput{
		NamespacedID: models.NamespacedID{
			Namespace: namespace,
		},
		Key:     planKey,
		Version: version,
	})

	if plan.IsNotFound(err) {
		return nil, subscription.NewPlanNotFoundError(planKey, version)
	} else if err != nil {
		return nil, err
	}

	if p == nil {
		return nil, subscription.NewPlanNotFoundError(planKey, version)
	}

	return p, nil
}

func PlanFromPlan(p plan.Plan) subscription.Plan {
	return &plansubscription.Plan{
		Plan: p.AsProductCatalogPlan(),
		Ref:  &p.NamespacedID,
	}
}
