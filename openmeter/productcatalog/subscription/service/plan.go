package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
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

	if p.Key != "" || p.Version != 0 {
		return nil, fmt.Errorf("plan key and version must be empty")
	}

	// Default settlement mode if not provided
	if p.SettlementMode == "" {
		p.SettlementMode = productcatalog.CreditThenInvoiceSettlementMode
	}

	if err := p.ValidateWith(productcatalog.ValidatePlanStructure()); err != nil {
		return nil, models.ErrorWithFieldPrefix(models.NewFieldSelectorGroup(
			models.NewFieldSelector("plan")), err)
	}

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
