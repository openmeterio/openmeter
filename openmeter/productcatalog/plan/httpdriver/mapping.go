package httpdriver

import (
	"fmt"

	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/http"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
)

func FromPlan(p plan.Plan) (api.Plan, error) {
	validationIssues, _ := p.AsProductCatalogPlan().ValidationErrors()

	resp := api.Plan{
		CreatedAt:        p.CreatedAt,
		Currency:         p.Currency.String(),
		DeletedAt:        p.DeletedAt,
		Description:      p.Description,
		EffectiveFrom:    p.EffectiveFrom,
		EffectiveTo:      p.EffectiveTo,
		Id:               p.ID,
		Key:              p.Key,
		Metadata:         lo.EmptyableToPtr(api.Metadata(p.Metadata)),
		Name:             p.Name,
		UpdatedAt:        p.UpdatedAt,
		Version:          p.Version,
		BillingCadence:   p.BillingCadence.String(),
		ProRatingConfig:  fromProRatingConfig(p.ProRatingConfig),
		ValidationErrors: http.FromValidationErrors(validationIssues),
	}

	resp.Phases = make([]api.PlanPhase, 0, len(p.Phases))
	for _, phase := range p.Phases {
		planPhase, err := FromPlanPhase(phase)
		if err != nil {
			return resp, fmt.Errorf("failed to cast Plan: %w", err)
		}

		resp.Phases = append(resp.Phases, planPhase)
	}

	var status api.PlanStatus
	switch p.Status() {
	case productcatalog.PlanStatusDraft:
		status = api.PlanStatusDraft
	case productcatalog.PlanStatusActive:
		status = api.PlanStatusActive
	case productcatalog.PlanStatusArchived:
		status = api.PlanStatusArchived
	case productcatalog.PlanStatusScheduled:
		status = api.PlanStatusScheduled
	default:
		return resp, fmt.Errorf("invalid PlanStatus: %s", p.Status())
	}

	resp.Status = status

	return resp, nil
}

func FromPlanPhase(p plan.Phase) (api.PlanPhase, error) {
	resp := api.PlanPhase{
		Description: p.Description,
		Key:         p.Key,
		Metadata:    http.FromMetadata(p.Metadata),
		Name:        p.Name,
		Duration:    (*string)(p.Duration.ISOStringPtrOrNil()),
	}

	resp.RateCards = make([]api.RateCard, 0, len(p.RateCards))
	for _, rateCard := range p.RateCards {
		rc, err := http.FromRateCard(rateCard)
		if err != nil {
			return resp, fmt.Errorf("failed to cast RateCard: %w", err)
		}

		resp.RateCards = append(resp.RateCards, rc)
	}

	return resp, nil
}

func AsCreatePlanRequest(a api.PlanCreate, namespace string) (CreatePlanRequest, error) {
	var err error

	req := CreatePlanRequest{
		NamespacedModel: models.NamespacedModel{
			Namespace: namespace,
		},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Key:             a.Key,
				Name:            a.Name,
				Description:     a.Description,
				Metadata:        lo.FromPtr(a.Metadata),
				ProRatingConfig: asProRatingConfig(a.ProRatingConfig),
			},
			Phases: nil,
		},
	}

	req.Currency = currency.Code(a.Currency)
	if err = req.Currency.Validate(); err != nil {
		return req, fmt.Errorf("invalid CurrencyCode: %w", err)
	}

	req.PlanMeta.BillingCadence, err = datetime.ISODurationString(a.BillingCadence).Parse()
	if err != nil {
		return req, fmt.Errorf("invalid BillingCadence: %w", err)
	}

	if len(a.Phases) > 0 {
		req.Phases = make([]productcatalog.Phase, 0, len(a.Phases))

		for _, phase := range a.Phases {
			planPhase, err := AsPlanPhase(phase)
			if err != nil {
				return req, fmt.Errorf("failed to cast PlanPhase: %w", err)
			}

			req.Phases = append(req.Phases, planPhase)
		}
	}

	return req, nil
}

// fromProRatingConfig converts domain ProRatingConfig to API ProRatingConfig
func fromProRatingConfig(p productcatalog.ProRatingConfig) *api.ProRatingConfig {
	return &api.ProRatingConfig{
		Enabled: p.Enabled,
		Mode:    api.ProRatingMode(p.Mode),
	}
}

// asProRatingConfig converts API ProRatingConfig to domain ProRatingConfig
func asProRatingConfig(p *api.ProRatingConfig) productcatalog.ProRatingConfig {
	if p == nil {
		// Return default configuration when not provided
		return productcatalog.ProRatingConfig{
			Enabled: true,
			Mode:    productcatalog.ProRatingModeProratePrices,
		}
	}

	return productcatalog.ProRatingConfig{
		Enabled: p.Enabled,
		Mode:    productcatalog.ProRatingMode(p.Mode),
	}
}

func AsPlanPhase(a api.PlanPhase) (productcatalog.Phase, error) {
	var err error

	phase := productcatalog.Phase{
		PhaseMeta: productcatalog.PhaseMeta{
			Key:         a.Key,
			Name:        a.Name,
			Description: a.Description,
			Metadata:    lo.FromPtr(a.Metadata),
		},
	}

	phase.Duration, err = (*datetime.ISODurationString)(a.Duration).ParsePtrOrNil()
	if err != nil {
		return phase, models.NewGenericValidationError(fmt.Errorf("invalid duration: failed to cast to period: %w", err))
	}

	phase.RateCards, err = http.AsRateCards(a.RateCards)
	if err != nil {
		return phase, err
	}

	return phase, nil
}

func AsUpdatePlanRequest(a api.PlanReplaceUpdate, namespace string, planID string) (UpdatePlanRequest, error) {
	req := UpdatePlanRequest{
		NamespacedID: models.NamespacedID{
			Namespace: namespace,
			ID:        planID,
		},
		Name:            lo.ToPtr(a.Name),
		Description:     a.Description,
		Metadata:        (*models.Metadata)(a.Metadata),
		ProRatingConfig: lo.ToPtr(asProRatingConfig(a.ProRatingConfig)),
	}

	if a.BillingCadence != "" {
		billingCadence, err := datetime.ISODurationString(a.BillingCadence).Parse()
		if err != nil {
			return req, models.NewGenericValidationError(fmt.Errorf("invalid billingCadence: %w", err))
		}

		req.BillingCadence = lo.ToPtr(billingCadence)
	}

	phases := make([]productcatalog.Phase, 0, len(a.Phases))
	for _, phase := range a.Phases {
		planPhase, err := AsPlanPhase(phase)
		if err != nil {
			return req, fmt.Errorf("failed to cast Plan Phase from HTTP update request: %w", err)
		}

		phases = append(phases, planPhase)
	}
	req.Phases = &phases

	return req, nil
}
