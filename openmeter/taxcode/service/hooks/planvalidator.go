package hooks

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type (
	PlanValidatorHook     = models.ServiceHook[taxcode.TaxCode]
	NoopPlanValidatorHook = models.NoopServiceHook[taxcode.TaxCode]
)

type PlanValidatorHookConfig struct {
	PlanService plan.Service
}

func (e PlanValidatorHookConfig) Validate() error {
	if e.PlanService == nil {
		return fmt.Errorf("plan service is required")
	}

	return nil
}

var _ models.ServiceHook[taxcode.TaxCode] = (*planValidatorHook)(nil)

type planValidatorHook struct {
	NoopPlanValidatorHook

	planService plan.Service
}

func NewPlanValidatorHook(config PlanValidatorHookConfig) (PlanValidatorHook, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid plan validator hook config: %w", err)
	}

	return &planValidatorHook{
		planService: config.PlanService,
	}, nil
}

func (e *planValidatorHook) PreDelete(ctx context.Context, tc *taxcode.TaxCode) error {
	affectedPlans, err := e.planService.ListPlans(ctx, plan.ListPlansInput{
		Status: []productcatalog.PlanStatus{
			productcatalog.PlanStatusActive,
			productcatalog.PlanStatusArchived,
			productcatalog.PlanStatusDraft,
			productcatalog.PlanStatusScheduled,
		},
		TaxCodes: &filter.FilterString{
			In: &[]string{
				tc.ID,
			},
		},
		Page: pagination.Page{
			PageSize:   5,
			PageNumber: 1,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to list plans: %w", err)
	}

	if len(affectedPlans.Items) > 0 {
		planIDs := lo.Map(affectedPlans.Items, func(item plan.Plan, _ int) string {
			return item.ID
		})
		return taxcode.NewTaxCodeReferencedByPlanError(tc.ID, planIDs)
	}

	return nil
}
