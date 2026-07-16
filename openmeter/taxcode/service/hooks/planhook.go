package hooks

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type (
	PlanHook     = models.ServiceHook[taxcode.TaxCode]
	NoopPlanHook = models.NoopServiceHook[taxcode.TaxCode]
)

type PlanHookConfig struct {
	PlanService plan.Service
}

func (e PlanHookConfig) Validate() error {
	if e.PlanService == nil {
		return fmt.Errorf("plan service is required")
	}

	return nil
}

var _ models.ServiceHook[taxcode.TaxCode] = (*planHook)(nil)

type planHook struct {
	NoopPlanHook

	planService plan.Service
}

func NewPlanHook(config PlanHookConfig) (PlanHook, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid plan hook config: %w", err)
	}

	return &planHook{
		planService: config.PlanService,
	}, nil
}

func (e *planHook) PreDelete(ctx context.Context, tc *taxcode.TaxCode) error {
	affectedPlans, err := e.planService.ListPlans(ctx, plan.ListPlansInput{
		Namespaces: []string{tc.Namespace},
		Status: []productcatalog.PlanStatus{
			productcatalog.PlanStatusActive,
			productcatalog.PlanStatusArchived,
			productcatalog.PlanStatusDraft,
			productcatalog.PlanStatusScheduled,
			productcatalog.PlanStatusInvalid,
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

	var errs []error

	for _, affectedPlan := range affectedPlans.Items {
		for _, phase := range affectedPlan.Phases {
			for _, rateCard := range phase.RateCards {
				taxCodeID := rateCard.AsMeta().TaxCodeReference()
				if taxCodeID == nil || *taxCodeID != tc.ID {
					continue
				}

				errs = append(errs, taxcode.NewTaxCodeReferencedByRateCardError(tc.ID, rateCard.Key()))
			}
		}
	}

	if len(affectedPlans.Items) > 0 && len(errs) == 0 {
		return fmt.Errorf("plan %s matched tax code filter but no rate card references tax code %s", affectedPlans.Items[0].ID, tc.ID)
	}

	return errors.Join(errs...)
}
