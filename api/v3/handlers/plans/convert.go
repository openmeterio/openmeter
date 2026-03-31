package plans

import (
	"fmt"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/pkg/models"
)

func FromPlan(p plan.Plan) (api.BillingPlan, error) {
	validationIssues, _ := p.AsProductCatalogPlan().ValidationErrors()

	resp := api.BillingPlan{
		BillingCadence:   api.ISO8601Duration(p.BillingCadence.String()),
		CreatedAt:        lo.ToPtr(p.CreatedAt),
		Currency:         api.CurrencyCode(p.Currency.String()),
		DeletedAt:        p.DeletedAt,
		Description:      p.Description,
		EffectiveFrom:    p.EffectiveFrom,
		EffectiveTo:      p.EffectiveTo,
		Id:               p.ID,
		Key:              p.Key,
		Name:             p.Name,
		UpdatedAt:        lo.ToPtr(p.UpdatedAt),
		Version:          p.Version,
		ProRatingEnabled: lo.ToPtr(p.ProRatingConfig.Enabled),
		ValidationErrors: fromValidationErrors(validationIssues),
	}

	var status api.BillingPlanStatus
	switch p.Status() {
	case productcatalog.PlanStatusDraft:
		status = api.BillingPlanStatusDraft
	case productcatalog.PlanStatusActive:
		status = api.BillingPlanStatusActive
	case productcatalog.PlanStatusArchived:
		status = api.BillingPlanStatusArchived
	case productcatalog.PlanStatusScheduled:
		status = api.BillingPlanStatusScheduled
	default:
		return resp, fmt.Errorf("invalid PlanStatus: %s", p.Status())
	}

	resp.Status = status

	resp.Phases = make([]api.BillingPlanPhase, 0, len(p.Phases))
	for _, phase := range p.Phases {
		billingPhase, err := fromPlanPhase(phase)
		if err != nil {
			return resp, fmt.Errorf("failed to convert plan phase: %w", err)
		}

		resp.Phases = append(resp.Phases, billingPhase)
	}

	return resp, nil
}

func fromPlanPhase(p plan.Phase) (api.BillingPlanPhase, error) {
	phase := api.BillingPlanPhase{
		Description: p.Description,
		Duration:    (*api.ISO8601Duration)(p.Duration.ISOStringPtrOrNil()),
		Key:         p.Key,
		Name:        p.Name,
		// TODO: convert rate cards to BillingRateCard
		RateCards: make([]api.BillingRateCard, 0, len(p.RateCards)),
	}

	return phase, nil
}

func fromValidationErrors(issues models.ValidationIssues) *[]api.ProductCatalogValidationError {
	if len(issues) == 0 {
		return nil
	}

	result := make([]api.ProductCatalogValidationError, 0, len(issues))
	for _, issue := range issues {
		result = append(result, api.ProductCatalogValidationError{
			Code:    string(issue.Code()),
			Field:   issue.Field().JSONPath(),
			Message: issue.Message(),
		})
	}

	return &result
}
