package plans

import (
	"fmt"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/labels"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
)

var unsupportedV3PriceTypes = map[productcatalog.PriceType]struct{}{
	productcatalog.DynamicPriceType: {},
	productcatalog.PackagePriceType: {},
}

func hasUnsupportedV3Price(p plan.Plan) bool {
	for _, phase := range p.Phases {
		for _, rc := range phase.RateCards {
			price := rc.AsMeta().Price
			if price == nil {
				continue
			}

			if _, unsupported := unsupportedV3PriceTypes[price.Type()]; unsupported {
				return true
			}
		}
	}

	return false
}

func FromPlan(p plan.Plan) (api.BillingPlan, error) {
	validationIssues, _ := p.AsProductCatalogPlan().ValidationErrors()

	resp := api.BillingPlan{
		BillingCadence:   p.BillingCadence.String(),
		CreatedAt:        lo.ToPtr(p.CreatedAt),
		Currency:         p.Currency.String(),
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
			result.BillingCadence = lo.ToPtr(bc.ISOString().String())
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

		result.BillingCadence = lo.ToPtr(bc.ISOString().String())

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
			Amount: flat.Amount.String(),
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
			Amount: unit.Amount.String(),
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
		return result, models.NewGenericConflictError(fmt.Errorf("dynamic price is not supported in v3 API"))

	case productcatalog.PackagePriceType:
		return result, models.NewGenericConflictError(fmt.Errorf("package price is not supported in v3 API"))

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
			tier.UpToAmount = lo.ToPtr(t.UpToAmount.String())
		}

		if t.FlatPrice != nil {
			tier.FlatPrice = &api.BillingPriceFlat{
				Amount: t.FlatPrice.Amount.String(),
				Type:   api.BillingPriceFlatType("flat"),
			}
		}

		if t.UnitPrice != nil {
			tier.UnitPrice = &api.BillingPriceUnit{
				Amount: t.UnitPrice.Amount.String(),
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
		result.Usage = lo.ToPtr(d.Usage.Quantity.String())
	}

	return result
}

func fromBillingCommitments(c productcatalog.Commitments) *api.BillingSpendCommitments {
	if c.MinimumAmount == nil && c.MaximumAmount == nil {
		return nil
	}

	result := &api.BillingSpendCommitments{}

	if c.MinimumAmount != nil {
		result.MinimumAmount = lo.ToPtr(c.MinimumAmount.String())
	}

	if c.MaximumAmount != nil {
		result.MaximumAmount = lo.ToPtr(c.MaximumAmount.String())
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

func toUpdatePlanInput(ns string, planID string, body api.UpsertPlanRequest) (plan.UpdatePlanInput, error) {
	req := plan.UpdatePlanInput{
		NamespacedID: models.NamespacedID{
			Namespace: ns,
			ID:        planID,
		},
		Name:            &body.Name,
		Description:     body.Description,
		ProRatingConfig: lo.ToPtr(toProRatingConfig(body.ProRatingEnabled)),
	}

	if body.Labels != nil {
		m := labels.ToMetadata(body.Labels)
		req.Metadata = &m
	}

	phases := make([]productcatalog.Phase, 0, len(body.Phases))
	for _, phase := range body.Phases {
		p, err := toPlanPhase(phase)
		if err != nil {
			return req, fmt.Errorf("failed to convert phase: %w", err)
		}

		phases = append(phases, p)
	}

	req.Phases = &phases

	return req, nil
}

func toCreatePlanInput(ns string, body api.CreatePlanRequest) (plan.CreatePlanInput, error) {
	req := plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: ns,
		},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Key:             body.Key,
				Name:            body.Name,
				Description:     body.Description,
				Metadata:        labels.ToMetadata(body.Labels),
				ProRatingConfig: toProRatingConfig(body.ProRatingEnabled),
			},
		},
	}

	cur := currency.Code(body.Currency)
	if err := cur.Validate(); err != nil {
		return req, fmt.Errorf("invalid currency: %w", err)
	}

	req.Currency = cur

	billingCadence, err := datetime.ISODurationString(body.BillingCadence).Parse()
	if err != nil {
		return req, fmt.Errorf("invalid billing cadence: %w", err)
	}

	req.BillingCadence = billingCadence

	if len(body.Phases) > 0 {
		req.Phases = make([]productcatalog.Phase, 0, len(body.Phases))

		for _, phase := range body.Phases {
			p, err := toPlanPhase(phase)
			if err != nil {
				return req, fmt.Errorf("failed to convert phase: %w", err)
			}

			req.Phases = append(req.Phases, p)
		}
	}

	return req, nil
}

func toProRatingConfig(enabled *bool) productcatalog.ProRatingConfig {
	if enabled == nil || *enabled {
		return productcatalog.ProRatingConfig{
			Enabled: true,
			Mode:    productcatalog.ProRatingModeProratePrices,
		}
	}

	return productcatalog.ProRatingConfig{
		Enabled: false,
	}
}

func toPlanPhase(p api.BillingPlanPhase) (productcatalog.Phase, error) {
	phase := productcatalog.Phase{
		PhaseMeta: productcatalog.PhaseMeta{
			Key:         p.Key,
			Name:        p.Name,
			Description: p.Description,
			Metadata:    labels.ToMetadata(p.Labels),
		},
	}

	var err error

	phase.Duration, err = (*datetime.ISODurationString)(p.Duration).ParsePtrOrNil()
	if err != nil {
		return phase, fmt.Errorf("invalid duration: %w", err)
	}

	if len(p.RateCards) > 0 {
		phase.RateCards = make(productcatalog.RateCards, 0, len(p.RateCards))

		for _, rc := range p.RateCards {
			rateCard, err := toRateCard(rc)
			if err != nil {
				return phase, fmt.Errorf("failed to convert rate card %q: %w", rc.Key, err)
			}

			phase.RateCards = append(phase.RateCards, rateCard)
		}
	}

	return phase, nil
}

func toRateCard(rc api.BillingRateCard) (productcatalog.RateCard, error) {
	priceType, err := rc.Price.Discriminator()
	if err != nil {
		return nil, fmt.Errorf("failed to read price type: %w", err)
	}

	meta := productcatalog.RateCardMeta{
		Key:         rc.Key,
		Name:        rc.Name,
		Description: rc.Description,
		Metadata:    labels.ToMetadata(rc.Labels),
	}

	if rc.Feature != nil {
		meta.FeatureID = &rc.Feature.Id
	}

	if rc.TaxConfig != nil {
		meta.TaxConfig = toBillingTaxConfig(*rc.TaxConfig)
	}

	if rc.Discounts != nil {
		discounts, err := toBillingDiscounts(*rc.Discounts)
		if err != nil {
			return nil, fmt.Errorf("failed to convert discounts: %w", err)
		}

		meta.Discounts = discounts
	}

	switch priceType {
	case "free", "flat":
		price, err := toBillingPrice(rc.Price, rc.PaymentTerm)
		if err != nil {
			return nil, fmt.Errorf("failed to convert price: %w", err)
		}

		meta.Price = price

		flatRC := &productcatalog.FlatFeeRateCard{
			RateCardMeta: meta,
		}

		if rc.BillingCadence != nil {
			bc, err := datetime.ISODurationString(*rc.BillingCadence).Parse()
			if err != nil {
				return nil, fmt.Errorf("invalid billing cadence: %w", err)
			}

			flatRC.BillingCadence = &bc
		}

		return flatRC, nil

	case "unit", "graduated", "volume":
		if rc.BillingCadence == nil {
			return nil, fmt.Errorf("billing cadence is required for usage-based rate card %q", rc.Key)
		}

		bc, err := datetime.ISODurationString(*rc.BillingCadence).Parse()
		if err != nil {
			return nil, fmt.Errorf("invalid billing cadence: %w", err)
		}

		price, err := toBillingPriceWithCommitments(rc.Price, rc.Commitments)
		if err != nil {
			return nil, fmt.Errorf("failed to convert price: %w", err)
		}

		meta.Price = price

		return &productcatalog.UsageBasedRateCard{
			RateCardMeta:   meta,
			BillingCadence: bc,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported price type: %s", priceType)
	}
}

func toBillingPrice(p api.BillingPrice, paymentTerm *api.BillingPricePaymentTerm) (*productcatalog.Price, error) {
	disc, err := p.Discriminator()
	if err != nil {
		return nil, fmt.Errorf("failed to read price type: %w", err)
	}

	switch disc {
	case "free":
		return nil, nil

	case "flat":
		flat, err := p.AsBillingPriceFlat()
		if err != nil {
			return nil, fmt.Errorf("failed to read flat price: %w", err)
		}

		amount, err := decimal.NewFromString(flat.Amount)
		if err != nil {
			return nil, fmt.Errorf("invalid flat price amount: %w", err)
		}

		term := productcatalog.DefaultPaymentTerm
		if paymentTerm != nil {
			term = productcatalog.PaymentTermType(*paymentTerm)
		}

		return productcatalog.NewPriceFrom(productcatalog.FlatPrice{
			Amount:      amount,
			PaymentTerm: term,
		}), nil

	default:
		return nil, fmt.Errorf("toBillingPrice does not handle price type %q", disc)
	}
}

func toBillingPriceWithCommitments(p api.BillingPrice, commitments *api.BillingSpendCommitments) (*productcatalog.Price, error) {
	disc, err := p.Discriminator()
	if err != nil {
		return nil, fmt.Errorf("failed to read price type: %w", err)
	}

	c, err := parseCommitments(commitments)
	if err != nil {
		return nil, err
	}

	switch disc {
	case "unit":
		unit, err := p.AsBillingPriceUnit()
		if err != nil {
			return nil, fmt.Errorf("failed to read unit price: %w", err)
		}

		amount, err := decimal.NewFromString(unit.Amount)
		if err != nil {
			return nil, fmt.Errorf("invalid unit price amount: %w", err)
		}

		return productcatalog.NewPriceFrom(productcatalog.UnitPrice{
			Amount:      amount,
			Commitments: c,
		}), nil

	case "graduated":
		grad, err := p.AsBillingPriceGraduated()
		if err != nil {
			return nil, fmt.Errorf("failed to read graduated price: %w", err)
		}

		tiers, err := toBillingPriceTiers(grad.Tiers)
		if err != nil {
			return nil, fmt.Errorf("failed to convert graduated tiers: %w", err)
		}

		return productcatalog.NewPriceFrom(productcatalog.TieredPrice{
			Mode:        productcatalog.GraduatedTieredPrice,
			Tiers:       tiers,
			Commitments: c,
		}), nil

	case "volume":
		vol, err := p.AsBillingPriceVolume()
		if err != nil {
			return nil, fmt.Errorf("failed to read volume price: %w", err)
		}

		tiers, err := toBillingPriceTiers(vol.Tiers)
		if err != nil {
			return nil, fmt.Errorf("failed to convert volume tiers: %w", err)
		}

		return productcatalog.NewPriceFrom(productcatalog.TieredPrice{
			Mode:        productcatalog.VolumeTieredPrice,
			Tiers:       tiers,
			Commitments: c,
		}), nil

	default:
		return nil, fmt.Errorf("unsupported usage-based price type: %s", disc)
	}
}

func parseCommitments(c *api.BillingSpendCommitments) (productcatalog.Commitments, error) {
	if c == nil {
		return productcatalog.Commitments{}, nil
	}

	result := productcatalog.Commitments{}

	if c.MinimumAmount != nil {
		min, err := decimal.NewFromString(*c.MinimumAmount)
		if err != nil {
			return result, fmt.Errorf("invalid minimum amount: %w", err)
		}

		result.MinimumAmount = &min
	}

	if c.MaximumAmount != nil {
		max, err := decimal.NewFromString(*c.MaximumAmount)
		if err != nil {
			return result, fmt.Errorf("invalid maximum amount: %w", err)
		}

		result.MaximumAmount = &max
	}

	return result, nil
}

func toBillingPriceTiers(tiers []api.BillingPriceTier) ([]productcatalog.PriceTier, error) {
	result := make([]productcatalog.PriceTier, 0, len(tiers))

	for _, t := range tiers {
		tier := productcatalog.PriceTier{}

		if t.UpToAmount != nil {
			amount, err := decimal.NewFromString(*t.UpToAmount)
			if err != nil {
				return nil, fmt.Errorf("invalid tier up-to amount: %w", err)
			}

			tier.UpToAmount = &amount
		}

		if t.FlatPrice != nil {
			amount, err := decimal.NewFromString(t.FlatPrice.Amount)
			if err != nil {
				return nil, fmt.Errorf("invalid tier flat price amount: %w", err)
			}

			tier.FlatPrice = &productcatalog.PriceTierFlatPrice{Amount: amount}
		}

		if t.UnitPrice != nil {
			amount, err := decimal.NewFromString(t.UnitPrice.Amount)
			if err != nil {
				return nil, fmt.Errorf("invalid tier unit price amount: %w", err)
			}

			tier.UnitPrice = &productcatalog.PriceTierUnitPrice{Amount: amount}
		}

		result = append(result, tier)
	}

	return result, nil
}

func toBillingTaxConfig(tc api.BillingRateCardTaxConfig) *productcatalog.TaxConfig {
	result := &productcatalog.TaxConfig{
		TaxCodeID: &tc.Code.Id,
	}

	if tc.Behavior != nil {
		result.Behavior = lo.ToPtr(productcatalog.TaxBehavior(*tc.Behavior))
	}

	return result
}

func fromPlanAddon(a planaddon.PlanAddon) (api.PlanAddon, error) {
	validationIssues, _ := a.AsProductCatalogPlanAddon().ValidationErrors()

	return api.PlanAddon{
		Id:               a.ID,
		Addon:            api.AddonReferenceItem{Id: a.Addon.ID},
		FromPlanPhase:    a.PlanAddonConfig.FromPlanPhase,
		MaxQuantity:      a.PlanAddonConfig.MaxQuantity,
		CreatedAt:        lo.ToPtr(a.CreatedAt),
		UpdatedAt:        lo.ToPtr(a.UpdatedAt),
		DeletedAt:        a.DeletedAt,
		Labels:           labels.FromMetadata(a.Metadata),
		ValidationErrors: fromValidationErrors(validationIssues),
	}, nil
}

func toBillingDiscounts(d api.BillingRateCardDiscounts) (productcatalog.Discounts, error) {
	result := productcatalog.Discounts{}

	if d.Percentage != nil {
		result.Percentage = &productcatalog.PercentageDiscount{
			Percentage: models.NewPercentage(float64(*d.Percentage)),
		}
	}

	if d.Usage != nil {
		qty, err := decimal.NewFromString(*d.Usage)
		if err != nil {
			return result, fmt.Errorf("invalid usage discount quantity: %w", err)
		}

		result.Usage = &productcatalog.UsageDiscount{
			Quantity: qty,
		}
	}

	return result, nil
}
