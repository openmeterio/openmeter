package service

import (
	"context"
	"fmt"

	"github.com/samber/lo"

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

// TODO: we can get rid of this if plan implements subscription.Plan or if we just use plain productcatalog.Plan
func PlanFromPlanInput(input plan.CreatePlanInput) (subscription.Plan, error) {
	p := input.Plan

	// We need to cheat a bit as plan validation fails without key and reference
	// There isn't a meaningful type to what we're using here
	// TODO: we could either
	// 1. redifine this partial type in `productcatalog` (though its only meaningful here)
	// 2. define the type here and figure out how to reuse the validations etc...
	// 3. accept the fact that we're cheating and remove this comment

	if !lo.IsEmpty(p.Key) || !lo.IsEmpty(p.Version) {
		// Let's safeguard ourselves
		return nil, fmt.Errorf("plan key and version must be empty")
	}

	// Let's set the fields for the validation to pass
	p.Key = "cheat"
	p.Version = 1

	if err := p.Validate(); err != nil {
		return nil, models.ErrorWithFieldPrefix(models.NewFieldSelectorGroup(
			models.NewFieldSelector("plan")), err)
	}

	// Let's unset the fields
	p.Key = ""
	p.Version = 0

	return &plansubscription.Plan{
		Plan: p,
	}, nil
}

func PlanFromPlan(p plan.Plan) subscription.Plan {
	return &plansubscription.Plan{
		Plan: p.AsProductCatalogPlan(),
		Ref:  &p.NamespacedID,
	}
}
