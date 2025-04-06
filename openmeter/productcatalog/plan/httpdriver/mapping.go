package httpdriver

import (
	"fmt"

	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/http"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/pkg/isodate"
	"github.com/openmeterio/openmeter/pkg/models"
)

func FromPlan(p plan.Plan) (api.Plan, error) {
	resp := api.Plan{
		CreatedAt:     p.CreatedAt,
		Currency:      p.Currency.String(),
		DeletedAt:     p.DeletedAt,
		Description:   p.Description,
		EffectiveFrom: p.EffectiveFrom,
		EffectiveTo:   p.EffectiveTo,
		Id:            p.ID,
		Key:           p.Key,
		Metadata:      lo.EmptyableToPtr(api.Metadata(p.Metadata)),
		Name:          p.Name,
		UpdatedAt:     p.UpdatedAt,
		Version:       p.Version,
		Alignment: &api.Alignment{
			BillablesMustAlign: lo.ToPtr(p.Alignment.BillablesMustAlign),
		},
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
		Metadata:    lo.EmptyableToPtr(api.Metadata(p.Metadata)),
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
				Key:         a.Key,
				Name:        a.Name,
				Description: a.Description,
				Metadata:    lo.FromPtrOr(a.Metadata, nil),
				Alignment: productcatalog.Alignment{
					BillablesMustAlign: func() bool {
						if a.Alignment != nil {
							if a.Alignment.BillablesMustAlign != nil {
								return *a.Alignment.BillablesMustAlign
							}
						}
						return true
					}(),
				},
			},
			Phases: nil,
		},
	}

	req.Currency = currency.Code(a.Currency)
	if err = req.Currency.Validate(); err != nil {
		return req, fmt.Errorf("invalid CurrencyCode: %w", err)
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

func AsPlanPhase(a api.PlanPhase) (productcatalog.Phase, error) {
	var err error

	phase := productcatalog.Phase{
		PhaseMeta: productcatalog.PhaseMeta{
			Key:         a.Key,
			Name:        a.Name,
			Description: a.Description,
			Metadata:    lo.FromPtrOr(a.Metadata, nil),
		},
	}

	phase.Duration, err = (*isodate.String)(a.Duration).ParsePtrOrNil()
	if err != nil {
		return phase, fmt.Errorf("failed to cast duration to period: %w", err)
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
		Name:        lo.ToPtr(a.Name),
		Description: a.Description,
		Metadata:    (*models.Metadata)(a.Metadata),
	}

	if a.Alignment != nil {
		if a.Alignment.BillablesMustAlign != nil {
			req.AlignmentUpdate.BillablesMustAlign = a.Alignment.BillablesMustAlign
		}
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
