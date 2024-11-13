package httpdriver

import (
	"fmt"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/rickb777/period"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/pkg/datex"
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
		Metadata:      lo.EmptyableToPtr(p.Metadata),
		Name:          p.Name,
		UpdatedAt:     p.UpdatedAt,
		Version:       p.Version,
	}

	if len(p.Phases) > 0 {
		resp.Phases = make([]api.PlanPhase, 0, len(p.Phases))

		for _, phase := range p.Phases {
			planPhase, err := FromPlanPhase(phase)
			if err != nil {
				return resp, fmt.Errorf("failed to cast Plan: %w", err)
			}

			resp.Phases = append(resp.Phases, planPhase)
		}
	}

	var status api.PlanStatus
	switch p.Status() {
	case plan.DraftStatus:
		status = api.PlanStatusDraft
	case plan.ActiveStatus:
		status = api.PlanStatusActive
	case plan.ArchivedStatus:
		status = api.PlanStatusArchived
	case plan.ScheduledStatus:
		status = api.PlanStatusScheduled
	default:
		return resp, fmt.Errorf("invalid PlanStatus: %s", p.Status())
	}

	resp.Status = status

	return resp, nil
}

func FromPlanPhase(p plan.Phase) (api.PlanPhase, error) {
	resp := api.PlanPhase{
		CreatedAt:   p.CreatedAt,
		DeletedAt:   p.DeletedAt,
		Description: p.Description,
		Key:         p.Key,
		Metadata:    lo.EmptyableToPtr(p.Metadata),
		Name:        p.Name,
		StartAfter:  lo.ToPtr(p.StartAfter.ISOString().String()),
		UpdatedAt:   p.UpdatedAt,
	}

	if len(p.Discounts) > 0 {
		discounts := make([]api.Discount, 0, len(p.Discounts))

		for _, discount := range p.Discounts {
			percentage, err := discount.AsPercentage()
			if err != nil {
				return resp, fmt.Errorf("failed to cast Discount: %w", err)
			}

			d := api.Discount{
				RateCards:  lo.ToPtr(percentage.RateCards),
				Percentage: float32(percentage.Percentage.InexactFloat64()),
				Type:       api.DiscountPercentageTypePercentage,
			}

			discounts = append(discounts, d)
		}

		resp.Discounts = lo.ToPtr(discounts)
	}

	if len(p.RateCards) > 0 {
		resp.RateCards = make([]api.RateCard, 0, len(p.RateCards))

		for _, rateCard := range p.RateCards {
			rc, err := FromRateCard(rateCard)
			if err != nil {
				return resp, fmt.Errorf("failed to cast RateCard: %w", err)
			}

			resp.RateCards = append(resp.RateCards, rc)
		}
	}

	return resp, nil
}

func FromRateCard(r plan.RateCard) (api.RateCard, error) {
	resp := api.RateCard{}

	switch r.Type() {
	case plan.FlatFeeRateCardType:
		rc, err := r.AsFlatFee()
		if err != nil {
			return resp, fmt.Errorf("failed to cast FlatFeeRateCard: %w", err)
		}

		var tmpl api.RateCardEntitlement
		if rc.EntitlementTemplate != nil {
			tmpl, err = FromEntitlementTemplate(*rc.EntitlementTemplate)
			if err != nil {
				return resp, fmt.Errorf("failed to cast EntitlementTemplate: %w", err)
			}
		}

		var featureKey *string
		if rc.Feature != nil {
			featureKey = &rc.Feature.Key
		}

		flatPrice, err := rc.Price.AsFlat()
		if err != nil {
			return resp, fmt.Errorf("failed to cast FlatPrice: %w", err)
		}

		var taxConfig *api.TaxConfig
		if rc.TaxConfig != nil {
			taxConfig = lo.ToPtr(FromTaxConfig(*rc.TaxConfig))
		}

		err = resp.FromRateCardFlatFee(api.RateCardFlatFee{
			BillingCadence:      lo.ToPtr(rc.BillingCadence.ISOString().String()),
			CreatedAt:           rc.CreatedAt,
			DeletedAt:           rc.DeletedAt,
			Description:         rc.Description,
			EntitlementTemplate: lo.EmptyableToPtr(tmpl),
			FeatureKey:          featureKey,
			Key:                 rc.Key,
			Metadata:            lo.ToPtr(rc.Metadata),
			Name:                rc.Name,
			Price: &api.FlatPriceWithPaymentTerm{
				Amount:      flatPrice.Amount.String(),
				PaymentTerm: lo.ToPtr(FromPaymentTerm(flatPrice.PaymentTerm)),
				Type:        api.FlatPriceWithPaymentTermTypeFlat,
			},
			TaxConfig: taxConfig,
			Type:      api.RateCardFlatFeeTypeFlatFee,
			UpdatedAt: rc.UpdatedAt,
		})
		if err != nil {
			return resp, fmt.Errorf("failed to cast FlatPriceRateCard: %w", err)
		}
	case plan.UsageBasedRateCardType:
	default:
		return resp, fmt.Errorf("invalid type: %s", r.Type())
	}

	return resp, nil
}

func FromTaxConfig(c plan.TaxConfig) api.TaxConfig {
	var stripe *api.StripeTaxConfig

	if c.Stripe != nil {
		stripe = &api.StripeTaxConfig{
			Code: c.Stripe.Code,
		}
	}

	return api.TaxConfig{
		Stripe: stripe,
	}
}

func FromPaymentTerm(t plan.PaymentTermType) api.PricePaymentTerm {
	switch t {
	case plan.InArrearsPaymentTerm:
		return api.PricePaymentTermInArrears
	case plan.InAdvancePaymentTerm:
		fallthrough
	default:
		return api.PricePaymentTermInAdvance
	}
}

func FromEntitlementTemplate(t plan.EntitlementTemplate) (api.RateCardEntitlement, error) {
	result := api.RateCardEntitlement{}

	switch t.Type() {
	case entitlement.EntitlementTypeMetered:
		metered, err := t.AsMetered()
		if err != nil {
			return result, fmt.Errorf("failed to cast Metered EntitlementTemplate: %w", err)
		}

		err = result.FromRateCardMeteredEntitlement(api.RateCardMeteredEntitlement{
			IsSoftLimit:             lo.ToPtr(metered.IsSoftLimit),
			IssueAfterReset:         metered.IssueAfterReset,
			IssueAfterResetPriority: metered.IssueAfterResetPriority,
			Metadata:                lo.ToPtr(metered.Metadata),
			PreserveOverageAtReset:  metered.PreserveOverageAtReset,
			Type:                    api.RateCardMeteredEntitlementTypeMetered,
			UsagePeriod:             lo.ToPtr(metered.UsagePeriod.ISOString().String()),
		})
		if err != nil {
			return result, fmt.Errorf("failed to cast Metered EntitlementTemplate: %w", err)
		}
	case entitlement.EntitlementTypeStatic:
		static, err := t.AsStatic()
		if err != nil {
			return result, fmt.Errorf("failed to cast Static EntitlementTemplate: %w", err)
		}

		err = result.FromRateCardStaticEntitlement(api.RateCardStaticEntitlement{
			Metadata: lo.ToPtr(static.Metadata),
			Type:     api.RateCardStaticEntitlementTypeStatic,
		})
		if err != nil {
			return result, fmt.Errorf("failed to cast Metered EntitlementTemplate: %w", err)
		}
	case entitlement.EntitlementTypeBoolean:
		boolean, err := t.AsBoolean()
		if err != nil {
			return result, fmt.Errorf("failed to cast Static EntitlementTemplate: %w", err)
		}

		err = result.FromRateCardBooleanEntitlement(api.RateCardBooleanEntitlement{
			Metadata: lo.ToPtr(boolean.Metadata),
			Type:     api.RateCardBooleanEntitlementTypeBoolean,
		})
		if err != nil {
			return result, fmt.Errorf("failed to cast Boolean EntitlementTemplate: %w", err)
		}
	default:
		return result, fmt.Errorf("invalid type: %s", t.Type())
	}

	return result, nil
}

func AsCreatePlanRequest(a api.PlanCreate, namespace string) (CreatePlanRequest, error) {
	var err error

	req := CreatePlanRequest{
		NamespacedModel: models.NamespacedModel{
			Namespace: namespace,
		},
		Key:         a.Key,
		Name:        a.Name,
		Description: a.Description,
		Metadata:    lo.FromPtrOr(a.Metadata, nil),
	}

	req.Currency = currency.Code(a.Currency)
	if err = req.Currency.Validate(); err != nil {
		return req, fmt.Errorf("invalid CurrencyCode: %w", err)
	}

	if len(a.Phases) > 0 {
		req.Phases = make([]plan.Phase, 0, len(a.Phases))

		for _, phase := range a.Phases {
			planPhase, err := AsPlanPhase(phase, namespace, "")
			if err != nil {
				return req, fmt.Errorf("failed to cast PlanPhase: %w", err)
			}

			req.Phases = append(req.Phases, planPhase)
		}
	}

	return req, nil
}

func AsPlanPhase(a api.PlanPhase, namespace, phaseID string) (plan.Phase, error) {
	var err error

	phase := plan.Phase{
		NamespacedID: models.NamespacedID{
			Namespace: namespace,
			ID:        phaseID,
		},
		ManagedModel: models.ManagedModel{
			CreatedAt: a.CreatedAt,
			UpdatedAt: a.UpdatedAt,
			DeletedAt: a.DeletedAt,
		},
		Key:         a.Key,
		Name:        a.Name,
		Description: a.Description,
		Metadata:    lo.FromPtrOr(a.Metadata, nil),
	}

	phase.StartAfter, err = datex.ISOString(lo.FromPtrOr(a.StartAfter, plan.DefaultStartAfter)).Parse()
	if err != nil {
		return phase, fmt.Errorf("failed to cast StartAfter date to period: %w", err)
	}

	discounts := lo.FromPtrOr(a.Discounts, nil)
	if len(discounts) > 0 {
		phase.Discounts = make([]plan.Discount, 0, len(discounts))

		for _, discount := range discounts {
			switch discount.Type {
			case api.DiscountPercentageTypePercentage:
				percentageDiscount := plan.PercentageDiscount{
					Percentage: decimal.NewFromFloat(float64(discount.Percentage)),
					RateCards:  lo.FromPtrOr(discount.RateCards, nil),
				}

				phaseDiscount := plan.NewDiscountFrom(percentageDiscount)
				if err = phaseDiscount.Validate(); err != nil {
					return phase, fmt.Errorf("invalid Discount: %w", err)
				}

				phase.Discounts = append(phase.Discounts, phaseDiscount)
			}
		}
	}

	if len(a.RateCards) > 0 {
		phase.RateCards = make([]plan.RateCard, 0, len(a.RateCards))

		for _, rc := range a.RateCards {
			rateCard, err := AsRateCard(rc, namespace)
			if err != nil {
				return phase, fmt.Errorf("failed to cast RateCard: %w", err)
			}

			phase.RateCards = append(phase.RateCards, rateCard)
		}
	}

	return phase, nil
}

func AsRateCard(r api.RateCard, namespace string) (plan.RateCard, error) {
	rType, err := r.Discriminator()
	if err != nil {
		return plan.RateCard{}, fmt.Errorf("failed to cast type: %w", err)
	}

	switch rType {
	case string(plan.FlatFeeRateCardType):
		flat, err := r.AsRateCardFlatFee()
		if err != nil {
			return plan.RateCard{}, fmt.Errorf("failed to cast FlatFeeRateCard: %w", err)
		}

		flatRateCard, err := AsFlatFeeRateCard(flat, namespace)
		if err != nil {
			return plan.RateCard{}, fmt.Errorf("failed to cast FlatFeeRateCard: %w", err)
		}

		return plan.NewRateCardFrom(flatRateCard), nil
	case string(plan.UsageBasedRateCardType):
		usage, err := r.AsRateCardUsageBased()
		if err != nil {
			return plan.RateCard{}, fmt.Errorf("failed to cast FlatFeeRateCard: %w", err)
		}

		usageBasedRateCard, err := AsUsageBasedRateCard(usage, namespace)
		if err != nil {
			return plan.RateCard{}, fmt.Errorf("failed to cast FlatFeeRateCard: %w", err)
		}

		return plan.NewRateCardFrom(usageBasedRateCard), nil
	default:
		return plan.RateCard{}, fmt.Errorf("invalid type: %s", rType)
	}
}

func AsFlatFeeRateCard(flat api.RateCardFlatFee, namespace string) (plan.FlatFeeRateCard, error) {
	var err error

	flatRateCard := plan.FlatFeeRateCard{
		RateCardMeta: plan.RateCardMeta{
			NamespacedID: models.NamespacedID{
				Namespace: namespace,
			},
			ManagedModel: models.ManagedModel{
				CreatedAt: flat.CreatedAt,
				UpdatedAt: flat.UpdatedAt,
				DeletedAt: flat.DeletedAt,
			},
			Key:         flat.Key,
			Name:        flat.Name,
			Description: flat.Description,
			Metadata:    lo.FromPtrOr(flat.Metadata, nil),
		},
	}

	if flat.FeatureKey != nil {
		flatRateCard.Feature = &feature.Feature{
			Key: *flat.FeatureKey,
		}
	}

	if flat.EntitlementTemplate != nil {
		tmpl, err := AsEntitlementTemplate(*flat.EntitlementTemplate)
		if err != nil {
			return plan.FlatFeeRateCard{}, fmt.Errorf("failed to cast EntitlementTemplate: %w", err)
		}

		flatRateCard.EntitlementTemplate = lo.ToPtr(tmpl)
	}

	if flat.TaxConfig != nil {
		flatRateCard.TaxConfig = lo.ToPtr(AsTaxConfig(*flat.TaxConfig))
	}

	if flat.BillingCadence != nil {
		isoString := datex.ISOString(*flat.BillingCadence)
		flatRateCard.BillingCadence, err = isoString.ParsePtrOrNil()
		if err != nil {
			return flatRateCard, fmt.Errorf("failed to cast BillingCadence: %w", err)
		}
	}

	amount, err := decimal.NewFromString(flat.Price.Amount)
	if err != nil {
		return flatRateCard, fmt.Errorf("failed to cast Price Amount to decimal: %w", err)
	}

	var paymentTerm plan.PaymentTermType
	if flat.Price.PaymentTerm != nil {
		switch *flat.Price.PaymentTerm {
		case api.PricePaymentTermInArrears:
			paymentTerm = plan.InArrearsPaymentTerm
		case api.PricePaymentTermInAdvance:
			paymentTerm = plan.InAdvancePaymentTerm
		default:
			paymentTerm = plan.DefaultPaymentTerm
		}
	}

	flatPrice := plan.FlatPrice{
		Amount:      amount,
		PaymentTerm: paymentTerm,
	}
	flatRateCard.Price = plan.NewPriceFrom(flatPrice)

	return flatRateCard, nil
}

func AsUsageBasedRateCard(usage api.RateCardUsageBased, namespace string) (plan.UsageBasedRateCard, error) {
	var err error

	usageRateCard := plan.UsageBasedRateCard{
		RateCardMeta: plan.RateCardMeta{
			NamespacedID: models.NamespacedID{
				Namespace: namespace,
			},
			ManagedModel: models.ManagedModel{
				CreatedAt: usage.CreatedAt,
				UpdatedAt: usage.UpdatedAt,
				DeletedAt: usage.DeletedAt,
			},
			Key:         usage.Key,
			Name:        usage.Name,
			Description: usage.Description,
			Metadata:    lo.FromPtrOr(usage.Metadata, nil),
		},
	}

	if usage.FeatureKey != nil {
		usageRateCard.Feature = &feature.Feature{
			Key: *usage.FeatureKey,
		}
	}

	if usage.EntitlementTemplate != nil {
		tmpl, err := AsEntitlementTemplate(*usage.EntitlementTemplate)
		if err != nil {
			return usageRateCard, fmt.Errorf("failed to cast EntitlementTemplate: %w", err)
		}

		usageRateCard.EntitlementTemplate = lo.ToPtr(tmpl)
	}

	if usage.TaxConfig != nil {
		usageRateCard.TaxConfig = lo.ToPtr(AsTaxConfig(*usage.TaxConfig))
	}

	isoString := datex.ISOString(usage.BillingCadence)
	usageRateCard.BillingCadence, err = isoString.Parse()
	if err != nil {
		return usageRateCard, fmt.Errorf("failed to cast BillingCadence: %w", err)
	}

	if usage.Price != nil {
		price, err := AsPrice(*usage.Price)
		if err != nil {
			return usageRateCard, fmt.Errorf("failed to cast Price: %w", err)
		}

		usageRateCard.Price = lo.ToPtr(price)
	}

	return usageRateCard, nil
}

func AsPrice(p api.RateCardUsageBasedPrice) (plan.Price, error) {
	var price plan.Price

	usagePriceType, err := p.Discriminator()
	if err != nil {
		return price, fmt.Errorf("failed to cast type: %w", err)
	}

	switch usagePriceType {
	case string(api.FlatPriceTypeFlat):
		flat, err := p.AsFlatPriceWithPaymentTerm()
		if err != nil {
			return price, fmt.Errorf("failed to cast FlatPrice: %w", err)
		}

		flatPrice := plan.FlatPrice{}

		flatPrice.Amount, err = decimal.NewFromString(flat.Amount)
		if err != nil {
			return price, fmt.Errorf("failed to cast Amount of FlatPrice to decimal: %w", err)
		}

		if flat.PaymentTerm != nil {
			switch *flat.PaymentTerm {
			case api.PricePaymentTermInArrears:
				flatPrice.PaymentTerm = plan.InArrearsPaymentTerm
			case api.PricePaymentTermInAdvance:
				flatPrice.PaymentTerm = plan.InAdvancePaymentTerm
			default:
				flatPrice.PaymentTerm = plan.DefaultPaymentTerm
			}
		}

		price = plan.NewPriceFrom(flatPrice)
	case string(api.UnitPriceTypeUnit):
		unit, err := p.AsUnitPriceWithCommitments()
		if err != nil {
			return price, fmt.Errorf("failed to cast UnitPrice: %w", err)
		}

		unitPrice := plan.UnitPrice{}

		unitPrice.Amount, err = decimal.NewFromString(unit.Amount)
		if err != nil {
			return price, fmt.Errorf("failed to cast Amount of UnitPrice to decimal: %w", err)
		}

		if unit.MinimumAmount != nil {
			minimumAmount, err := decimal.NewFromString(*unit.MinimumAmount)
			if err != nil {
				return price, fmt.Errorf("failed to cast MinimumAmount of UnitPrice to decimal: %w", err)
			}

			unitPrice.MinimumAmount = &minimumAmount
		}

		if unit.MaximumAmount != nil {
			maximumAmount, err := decimal.NewFromString(*unit.MaximumAmount)
			if err != nil {
				return price, fmt.Errorf("failed to cast MaximumAmount of UnitPrice to decimal: %w", err)
			}

			unitPrice.MaximumAmount = &maximumAmount
		}

		price = plan.NewPriceFrom(unitPrice)
	case string(api.TieredPriceWithCommitmentsTypeTiered):
		tiered, err := p.AsTieredPriceWithCommitments()
		if err != nil {
			return price, fmt.Errorf("failed to cast TieredPrice: %w", err)
		}

		tieredPrice := plan.TieredPrice{
			Tiers: nil,
		}

		tieredPrice.Mode, err = plan.NewTieredPriceMode(string(tiered.Mode))
		if err != nil {
			return price, fmt.Errorf("failed to cast TieredPriceMode: %w", err)
		}

		if tiered.MinimumAmount != nil {
			minimumAmount, err := decimal.NewFromString(*tiered.MinimumAmount)
			if err != nil {
				return price, fmt.Errorf("failed to cast MaximumAmount of UnitPrice to decimal: %w", err)
			}

			tieredPrice.MinimumAmount = &minimumAmount
		}

		if tiered.MaximumAmount != nil {
			maximumAmount, err := decimal.NewFromString(*tiered.MaximumAmount)
			if err != nil {
				return price, fmt.Errorf("failed to cast MaximumAmount of UnitPrice to decimal: %w", err)
			}

			tieredPrice.MaximumAmount = &maximumAmount
		}

		if len(tiered.Tiers) > 0 {
			tieredPrice.Tiers = make([]plan.PriceTier, 0, len(tiered.Tiers))
			for _, tier := range tiered.Tiers {
				priceTier, err := AsPriceTier(tier)
				if err != nil {
					return price, fmt.Errorf("failed to cast PriceTier: %w", err)
				}

				tieredPrice.Tiers = append(tieredPrice.Tiers, priceTier)
			}
		}

		price = plan.NewPriceFrom(tieredPrice)
	default:
		return price, fmt.Errorf("invalid Price type for UsageBasedRateCard: %s", usagePriceType)
	}

	return price, nil
}

func AsPriceTier(t api.PriceTier) (plan.PriceTier, error) {
	tier := plan.PriceTier{
		UpToAmount: nil,
		FlatPrice:  nil,
		UnitPrice:  nil,
	}

	if t.UpToAmount != nil {
		tier.UpToAmount = lo.ToPtr(decimal.NewFromFloat(*t.UpToAmount))
	}

	if t.FlatPrice != nil {
		amount, err := decimal.NewFromString(t.FlatPrice.Amount)
		if err != nil {
			return tier, fmt.Errorf("invalid Amount for FlatPrice component in PriceTier: %w", err)
		}

		tier.FlatPrice = &plan.PriceTierFlatPrice{
			Amount: amount,
		}
	}

	if t.UnitPrice != nil {
		amount, err := decimal.NewFromString(t.UnitPrice.Amount)
		if err != nil {
			return tier, fmt.Errorf("invalid Amount for UnitPrice component in PriceTier: %w", err)
		}

		tier.UnitPrice = &plan.PriceTierUnitPrice{
			Amount: amount,
		}
	}

	return tier, nil
}

func AsEntitlementTemplate(e api.RateCardEntitlement) (plan.EntitlementTemplate, error) {
	tmpl := plan.EntitlementTemplate{}

	eType, err := e.Discriminator()
	if err != nil {
		return tmpl, fmt.Errorf("failed to cast EntitlementTemplate type: %w", err)
	}

	switch eType {
	case string(api.RateCardMeteredEntitlementTypeMetered):
		metered, err := e.AsRateCardMeteredEntitlement()
		if err != nil {
			return tmpl, fmt.Errorf("failed to cast Metered EntitlementTemplate: %w", err)
		}

		usagePeriodISO := datex.ISOString(lo.FromPtrOr(metered.UsagePeriod, ""))
		usagePeriod, err := usagePeriodISO.Parse()
		if err != nil {
			return tmpl, fmt.Errorf("failed to cast UsagePeriod for Metered EntitlementTemplate: %w", err)
		}

		meteredTemplate := plan.MeteredEntitlementTemplate{
			Metadata:                lo.FromPtrOr(metered.Metadata, nil),
			IsSoftLimit:             lo.FromPtrOr(metered.IsSoftLimit, false),
			IssueAfterReset:         metered.IssueAfterReset,
			IssueAfterResetPriority: metered.IssueAfterResetPriority,
			PreserveOverageAtReset:  metered.PreserveOverageAtReset,
			UsagePeriod:             usagePeriod,
		}

		tmpl = plan.NewEntitlementTemplateFrom(meteredTemplate)
	case string(api.RateCardStaticEntitlementTypeStatic):
		static, err := e.AsRateCardStaticEntitlement()
		if err != nil {
			return tmpl, fmt.Errorf("failed to cast Static EntitlementTemplate: %w", err)
		}

		staticTemplate := plan.StaticEntitlementTemplate{
			Metadata: lo.FromPtrOr(static.Metadata, nil),
			Config:   static.Config,
		}

		tmpl = plan.NewEntitlementTemplateFrom(staticTemplate)
	case string(api.RateCardBooleanEntitlementTypeBoolean):
		boolean, err := e.AsRateCardMeteredEntitlement()
		if err != nil {
			return tmpl, fmt.Errorf("failed to cast Boolean EntitlementTemplate: %w", err)
		}

		booleanTemplate := plan.BooleanEntitlementTemplate{
			Metadata: lo.FromPtrOr(boolean.Metadata, nil),
		}

		tmpl = plan.NewEntitlementTemplateFrom(booleanTemplate)
	default:
		return plan.EntitlementTemplate{}, fmt.Errorf("invalid EntitlementTemplate type: %s", eType)
	}

	return tmpl, nil
}

func AsTaxConfig(c api.TaxConfig) plan.TaxConfig {
	tc := plan.TaxConfig{}

	if c.Stripe != nil {
		tc.Stripe = &plan.StripeTaxConfig{
			Code: c.Stripe.Code,
		}
	}

	return tc
}

func AsUpdatePlanRequest(a api.PlanUpdate, namespace string, planID string) (UpdatePlanRequest, error) {
	req := UpdatePlanRequest{
		NamespacedID: models.NamespacedID{
			Namespace: namespace,
			ID:        planID,
		},
		Name:        a.Name,
		Description: a.Description,
		Metadata:    a.Metadata,
	}

	if a.Phases != nil && *a.Phases != nil {
		phases := make([]plan.Phase, 0, len(*a.Phases))
		if len(*a.Phases) > 0 {
			for _, phase := range *a.Phases {
				planPhase, err := AsPlanPhase(phase, namespace, "")
				if err != nil {
					return req, fmt.Errorf("failed to cast Plan Phase from HTTP update request: %w", err)
				}

				phases = append(phases, planPhase)
			}
		}

		req.Phases = &phases
	}

	return req, nil
}

func AsCreatePhaseRequest(a api.PlanPhaseCreate, namespace, planID string) (CreatePhaseRequest, error) {
	var err error

	req := CreatePhaseRequest{
		NamespacedModel: models.NamespacedModel{
			Namespace: namespace,
		},
		Key:         a.Key,
		Name:        a.Name,
		Description: a.Description,
		Metadata:    lo.FromPtrOr(a.Metadata, nil),
		StartAfter: datex.Period{
			Period: period.Period{},
		},
		PlanID: planID,
	}

	req.StartAfter, err = datex.ISOString(lo.FromPtrOr(a.StartAfter, plan.DefaultStartAfter)).Parse()
	if err != nil {
		return req, fmt.Errorf("failed to parse StartAfter period: %w", err)
	}

	if len(a.RateCards) > 0 {
		req.RateCards = make([]plan.RateCard, 0, len(a.RateCards))

		for _, rc := range a.RateCards {
			rateCard, err := AsRateCard(rc, namespace)
			if err != nil {
				return req, fmt.Errorf("failed to cast RateCard from HTTP create request: %w", err)
			}

			req.RateCards = append(req.RateCards, rateCard)
		}
	}

	return req, nil
}

func AsUpdatePhaseRequest(a api.PlanPhaseUpdate, namespace, planID, phaseKey string) (UpdatePhaseRequest, error) {
	req := UpdatePhaseRequest{
		NamespacedID: models.NamespacedID{
			Namespace: namespace,
		},
		Key:         phaseKey,
		Name:        a.Name,
		Description: a.Description,
		Metadata:    a.Metadata,
		PlanID:      planID,
	}

	if a.StartAfter != nil {
		startAfterISO := datex.ISOString(*a.StartAfter)

		startAfter, err := startAfterISO.ParsePtrOrNil()
		if err != nil {
			return req, fmt.Errorf("failed to parse StartAfter period: %w", err)
		}

		req.StartAfter = startAfter
	}

	if a.RateCards != nil && *a.RateCards != nil {
		phases := make([]plan.RateCard, 0, len(*a.RateCards))
		if len(*a.RateCards) > 0 {
			for _, phase := range *a.RateCards {
				planPhase, err := AsRateCard(phase, namespace)
				if err != nil {
					return req, fmt.Errorf("failed to cast RateCard from HTTP update request: %w", err)
				}

				phases = append(phases, planPhase)
			}
		}

		req.RateCards = &phases
	}

	return req, nil
}
