package plans

import (
	"fmt"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
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
		RateCards:   make([]api.BillingRateCard, 0, len(p.RateCards)),
	}

	for _, rc := range p.RateCards {
		billingRC, err := fromRateCard(rc)
		if err != nil {
			return phase, fmt.Errorf("failed to convert rate card %q: %w", rc.Key(), err)
		}

		phase.RateCards = append(phase.RateCards, billingRC)
	}

	return phase, nil
}

func fromRateCard(rc productcatalog.RateCard) (api.BillingRateCard, error) {
	meta := rc.AsMeta()

	result := api.BillingRateCard{
		Key:         meta.Key,
		Name:        meta.Name,
		Description: meta.Description,
		Discounts:   fromBillingDiscounts(meta.Discounts),
		TaxConfig:   fromBillingTaxConfig(meta.TaxConfig, meta.TaxCode),
	}

	if meta.FeatureID != nil {
		result.Feature = &api.FeatureReferenceItem{
			Id: *meta.FeatureID,
		}
	}

	switch rc.Type() {
	case productcatalog.FlatFeeRateCardType:
		if bc := rc.GetBillingCadence(); bc != nil {
			result.BillingCadence = lo.ToPtr(api.ISO8601Duration(bc.ISOString().String()))
		}

		if meta.Price != nil {
			flatPrice, err := meta.Price.AsFlat()
			if err != nil {
				return result, fmt.Errorf("failed to read flat price: %w", err)
			}

			result.PaymentTerm = lo.ToPtr(api.BillingPricePaymentTerm(flatPrice.PaymentTerm))
		}

	case productcatalog.UsageBasedRateCardType:
		bc := rc.GetBillingCadence()
		if bc == nil {
			return result, fmt.Errorf("usage-based rate card %q missing billing cadence", meta.Key)
		}

		result.BillingCadence = lo.ToPtr(api.ISO8601Duration(bc.ISOString().String()))

		if meta.Price != nil {
			result.Commitments = fromBillingCommitments(meta.Price.GetCommitments())
		}

	default:
		return result, fmt.Errorf("unknown rate card type: %s", rc.Type())
	}

	price, err := fromBillingPrice(meta.Price)
	if err != nil {
		return result, fmt.Errorf("failed to convert price: %w", err)
	}

	result.Price = price

	return result, nil
}

func fromBillingPrice(p *productcatalog.Price) (api.BillingPrice, error) {
	var result api.BillingPrice

	if p == nil {
		if err := result.FromBillingPriceFree(api.BillingPriceFree{
			Type: api.BillingPriceFreeType("free"),
		}); err != nil {
			return result, fmt.Errorf("failed to set free price: %w", err)
		}

		return result, nil
	}

	switch p.Type() {
	case productcatalog.FlatPriceType:
		flat, err := p.AsFlat()
		if err != nil {
			return result, fmt.Errorf("failed to read flat price: %w", err)
		}

		if err = result.FromBillingPriceFlat(api.BillingPriceFlat{
			Amount: api.Numeric(flat.Amount.String()),
			Type:   api.BillingPriceFlatType("flat"),
		}); err != nil {
			return result, fmt.Errorf("failed to set flat price: %w", err)
		}

	case productcatalog.UnitPriceType:
		unit, err := p.AsUnit()
		if err != nil {
			return result, fmt.Errorf("failed to read unit price: %w", err)
		}

		if err = result.FromBillingPriceUnit(api.BillingPriceUnit{
			Amount: api.Numeric(unit.Amount.String()),
			Type:   api.BillingPriceUnitType("unit"),
		}); err != nil {
			return result, fmt.Errorf("failed to set unit price: %w", err)
		}

	case productcatalog.TieredPriceType:
		tiered, err := p.AsTiered()
		if err != nil {
			return result, fmt.Errorf("failed to read tiered price: %w", err)
		}

		tiers := fromBillingPriceTiers(tiered.Tiers)

		switch tiered.Mode {
		case productcatalog.GraduatedTieredPrice:
			if err = result.FromBillingPriceGraduated(api.BillingPriceGraduated{
				Tiers: tiers,
				Type:  api.BillingPriceGraduatedType("graduated"),
			}); err != nil {
				return result, fmt.Errorf("failed to set graduated price: %w", err)
			}

		case productcatalog.VolumeTieredPrice:
			if err = result.FromBillingPriceVolume(api.BillingPriceVolume{
				Tiers: tiers,
				Type:  api.BillingPriceVolumeType("volume"),
			}); err != nil {
				return result, fmt.Errorf("failed to set volume price: %w", err)
			}

		default:
			return result, fmt.Errorf("unknown tiered price mode: %s", tiered.Mode)
		}

	case productcatalog.DynamicPriceType:
		return result, fmt.Errorf("dynamic price is not supported in v3 API")

	case productcatalog.PackagePriceType:
		return result, fmt.Errorf("package price is not supported in v3 API")

	default:
		return result, fmt.Errorf("unknown price type: %s", p.Type())
	}

	return result, nil
}

func fromBillingPriceTiers(tiers []productcatalog.PriceTier) []api.BillingPriceTier {
	result := make([]api.BillingPriceTier, 0, len(tiers))

	for _, t := range tiers {
		tier := api.BillingPriceTier{}

		if t.UpToAmount != nil {
			tier.UpToAmount = lo.ToPtr(api.Numeric(t.UpToAmount.String()))
		}

		if t.FlatPrice != nil {
			tier.FlatPrice = &api.BillingPriceFlat{
				Amount: api.Numeric(t.FlatPrice.Amount.String()),
				Type:   api.BillingPriceFlatType("flat"),
			}
		}

		if t.UnitPrice != nil {
			tier.UnitPrice = &api.BillingPriceUnit{
				Amount: api.Numeric(t.UnitPrice.Amount.String()),
				Type:   api.BillingPriceUnitType("unit"),
			}
		}

		result = append(result, tier)
	}

	return result
}

func fromBillingTaxConfig(c *productcatalog.TaxConfig, tc *taxcode.TaxCode) *api.BillingRateCardTaxConfig {
	if c == nil || tc == nil {
		return nil
	}

	result := &api.BillingRateCardTaxConfig{
		Code: api.TaxCodeReferenceItem{
			Id: tc.ID,
		},
	}

	if c.Behavior != nil {
		result.Behavior = lo.ToPtr(api.BillingTaxBehavior(*c.Behavior))
	}

	return result
}

func fromBillingDiscounts(d productcatalog.Discounts) *api.BillingRateCardDiscounts {
	if d.Percentage == nil && d.Usage == nil {
		return nil
	}

	result := &api.BillingRateCardDiscounts{}

	if d.Percentage != nil {
		pct := float32(d.Percentage.Percentage.InexactFloat64())
		result.Percentage = &pct
	}

	if d.Usage != nil {
		result.Usage = lo.ToPtr(api.Numeric(d.Usage.Quantity.String()))
	}

	return result
}

func fromBillingCommitments(c productcatalog.Commitments) *api.BillingSpendCommitments {
	if c.MinimumAmount == nil && c.MaximumAmount == nil {
		return nil
	}

	result := &api.BillingSpendCommitments{}

	if c.MinimumAmount != nil {
		result.MinimumAmount = lo.ToPtr(api.Numeric(c.MinimumAmount.String()))
	}

	if c.MaximumAmount != nil {
		result.MaximumAmount = lo.ToPtr(api.Numeric(c.MaximumAmount.String()))
	}

	return result
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
