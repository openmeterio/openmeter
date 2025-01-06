package service

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/clock"
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

	return p, nil
}

// TODO: we can get rid of this if plan implements subscription.Plan or if we just use plain productcatalog.Plan
func PlanFromPlanInput(input plan.CreatePlanInput) (subscription.Plan, error) {
	p := input.Plan

	if err := p.ValidForCreatingSubscriptions(); err != nil {
		return nil, &models.GenericUserError{Inner: fmt.Errorf("invalid plan: %v", err)}
	}

	return &plansubscription.Plan{
		Plan: p,
	}, nil
}

func PlanFromPlan(p plan.Plan) (subscription.Plan, error) {
	pp, err := p.AsProductCatalogPlan(clock.Now())
	if err != nil {
		return nil, err
	}

	return &plansubscription.Plan{
		Plan: pp,
		Ref:  &p.NamespacedID,
	}, nil
}
