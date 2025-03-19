package httpdriver

import (
	"errors"
	"fmt"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
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
	case productcatalog.DraftStatus:
		status = api.PlanStatusDraft
	case productcatalog.ActiveStatus:
		status = api.PlanStatusActive
	case productcatalog.ArchivedStatus:
		status = api.PlanStatusArchived
	case productcatalog.ScheduledStatus:
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

	if len(p.Discounts) > 0 {
		discounts := make([]api.Discount, 0, len(p.Discounts))

		for _, discount := range p.Discounts {
			percentage, err := discount.AsPercentage()
			if err != nil {
				return resp, fmt.Errorf("failed to cast Discount: %w", err)
			}

			d := api.Discount{
				RateCards:  lo.EmptyableToPtr(percentage.RateCards),
				Percentage: percentage.Percentage,
				Type:       api.DiscountPercentageTypePercentage,
			}

			discounts = append(discounts, d)
		}

		resp.Discounts = lo.ToPtr(discounts)
	}

	resp.RateCards = make([]api.RateCard, 0, len(p.RateCards))
	for _, rateCard := range p.RateCards {
		rc, err := FromRateCard(rateCard)
		if err != nil {
			return resp, fmt.Errorf("failed to cast RateCard: %w", err)
		}

		resp.RateCards = append(resp.RateCards, rc)
	}

	return resp, nil
}

func FromRateCard(r productcatalog.RateCard) (api.RateCard, error) {
	var err error

	resp := api.RateCard{}

	switch r.Type() {
	case productcatalog.FlatFeeRateCardType:
		rc, ok := r.(*plan.FlatFeeRateCard)
		if !ok {
			return resp, errors.New("failed to cast FlatFeeRateCard")
		}

		var tmpl api.RateCardEntitlement
		if rc.EntitlementTemplate != nil {
			tmpl, err = FromEntitlementTemplate(*rc.EntitlementTemplate)
			if err != nil {
				return resp, fmt.Errorf("failed to cast EntitlementTemplate: %w", err)
			}
		}

		var featureKey *string
		if rc.Feature() != nil {
			featureKey = &rc.Feature().Key
		}

		var billingCadence *string
		if rc.BillingCadence != nil {
			billingCadence = lo.ToPtr(rc.BillingCadence.ISOString().String())
		}

		var price *api.FlatPriceWithPaymentTerm
		if rc.Price != nil {
			flatPrice, err := rc.Price.AsFlat()
			if err != nil {
				return resp, fmt.Errorf("failed to cast FlatPrice: %w", err)
			}

			price = &api.FlatPriceWithPaymentTerm{
				Amount:      flatPrice.Amount.String(),
				PaymentTerm: lo.ToPtr(FromPaymentTerm(flatPrice.PaymentTerm)),
				Type:        api.FlatPriceWithPaymentTermTypeFlat,
			}
		}

		var taxConfig *api.TaxConfig
		if rc.TaxConfig != nil {
			taxConfig = lo.ToPtr(FromTaxConfig(*rc.TaxConfig))
		}

		err = resp.FromRateCardFlatFee(api.RateCardFlatFee{
			BillingCadence:      billingCadence,
			Description:         rc.Description,
			EntitlementTemplate: lo.EmptyableToPtr(tmpl),
			FeatureKey:          featureKey,
			Key:                 rc.Key(),
			Metadata:            lo.EmptyableToPtr(api.Metadata(rc.Metadata)),
			Name:                rc.Name,
			Price:               price,
			TaxConfig:           taxConfig,
			Type:                api.RateCardFlatFeeTypeFlatFee,
		})
		if err != nil {
			return resp, fmt.Errorf("failed to cast FlatPriceRateCard: %w", err)
		}
	case productcatalog.UsageBasedRateCardType:
		rc, ok := r.(*plan.UsageBasedRateCard)
		if !ok {
			return resp, errors.New("failed to cast UsageBasedRateCard")
		}

		var tmpl api.RateCardEntitlement
		if rc.EntitlementTemplate != nil {
			tmpl, err = FromEntitlementTemplate(*rc.EntitlementTemplate)
			if err != nil {
				return resp, fmt.Errorf("failed to cast EntitlementTemplate: %w", err)
			}
		}

		var featureKey *string
		if rc.Feature() != nil {
			featureKey = &rc.Feature().Key
		}

		var price api.RateCardUsageBasedPrice
		if rc.Price != nil {
			switch rc.Price.Type() {
			case productcatalog.FlatPriceType:
				flatPrice, err := rc.Price.AsFlat()
				if err != nil {
					return resp, fmt.Errorf("failed to cast FlatPrice: %w", err)
				}

				err = price.FromFlatPriceWithPaymentTerm(api.FlatPriceWithPaymentTerm{
					Amount:      flatPrice.Amount.String(),
					PaymentTerm: lo.ToPtr(FromPaymentTerm(flatPrice.PaymentTerm)),
					Type:        api.FlatPriceWithPaymentTermTypeFlat,
				})
				if err != nil {
					return resp, fmt.Errorf("failed to cast FlatPrice: %w", err)
				}
			case productcatalog.UnitPriceType:
				unitPrice, err := rc.Price.AsUnit()
				if err != nil {
					return resp, fmt.Errorf("failed to cast UnitPrice: %w", err)
				}

				var minimumAmount *string
				if unitPrice.MinimumAmount != nil {
					minimumAmount = lo.ToPtr(unitPrice.MinimumAmount.String())
				}

				var maximumAmount *string
				if unitPrice.MaximumAmount != nil {
					maximumAmount = lo.ToPtr(unitPrice.MaximumAmount.String())
				}

				err = price.FromUnitPriceWithCommitments(api.UnitPriceWithCommitments{
					Amount:        unitPrice.Amount.String(),
					MinimumAmount: minimumAmount,
					MaximumAmount: maximumAmount,
					Type:          api.UnitPriceWithCommitmentsTypeUnit,
				})
				if err != nil {
					return resp, fmt.Errorf("failed to cast UnitPrice: %w", err)
				}
			case productcatalog.TieredPriceType:
				tieredPrice, err := rc.Price.AsTiered()
				if err != nil {
					return resp, fmt.Errorf("failed to cast TieredPrice: %w", err)
				}

				var minimumAmount *string
				if tieredPrice.MinimumAmount != nil {
					minimumAmount = lo.ToPtr(tieredPrice.MinimumAmount.String())
				}

				var maximumAmount *string
				if tieredPrice.MaximumAmount != nil {
					maximumAmount = lo.ToPtr(tieredPrice.MaximumAmount.String())
				}

				err = price.FromTieredPriceWithCommitments(api.TieredPriceWithCommitments{
					Type:          api.TieredPriceWithCommitmentsTypeTiered,
					Mode:          api.TieredPriceMode(tieredPrice.Mode),
					MinimumAmount: minimumAmount,
					MaximumAmount: maximumAmount,
					Tiers: lo.Map(tieredPrice.Tiers, func(t productcatalog.PriceTier, _ int) api.PriceTier {
						var upToAmount *api.Numeric
						if t.UpToAmount != nil {
							upToAmount = lo.ToPtr(t.UpToAmount.String())
						}

						var unitPrice *api.UnitPrice
						if t.UnitPrice != nil {
							unitPrice = &api.UnitPrice{
								Type:   api.UnitPriceTypeUnit,
								Amount: t.UnitPrice.Amount.String(),
							}
						}

						var flatPrice *api.FlatPrice
						if t.FlatPrice != nil {
							flatPrice = &api.FlatPrice{
								Type:   api.FlatPriceTypeFlat,
								Amount: t.FlatPrice.Amount.String(),
							}
						}

						return api.PriceTier{
							UpToAmount: upToAmount,
							UnitPrice:  unitPrice,
							FlatPrice:  flatPrice,
						}
					}),
				})
				if err != nil {
					return resp, fmt.Errorf("failed to cast TieredPrice: %w", err)
				}
			}
		}

		var taxConfig *api.TaxConfig
		if rc.TaxConfig != nil {
			taxConfig = lo.ToPtr(FromTaxConfig(*rc.TaxConfig))
		}

		err = resp.FromRateCardUsageBased(api.RateCardUsageBased{
			Type:                api.RateCardUsageBasedTypeUsageBased,
			BillingCadence:      rc.BillingCadence.ISOString().String(),
			Description:         rc.Description,
			EntitlementTemplate: lo.EmptyableToPtr(tmpl),
			FeatureKey:          featureKey,
			Key:                 rc.Key(),
			Metadata:            lo.EmptyableToPtr(api.Metadata(rc.Metadata)),
			Name:                rc.Name,
			Price:               &price,
			TaxConfig:           taxConfig,
		})
		if err != nil {
			return resp, fmt.Errorf("failed to cast UsageBasedRateCard: %w", err)
		}
	default:
		return resp, fmt.Errorf("invalid RateCard type: %s", r.Type())
	}

	return resp, nil
}

func FromTaxConfig(c productcatalog.TaxConfig) api.TaxConfig {
	var stripe *api.StripeTaxConfig

	if c.Stripe != nil {
		stripe = &api.StripeTaxConfig{
			Code: c.Stripe.Code,
		}
	}

	return api.TaxConfig{
		Stripe:   stripe,
		Behavior: (*api.TaxBehavior)(c.Behavior),
	}
}

func FromPaymentTerm(t productcatalog.PaymentTermType) api.PricePaymentTerm {
	switch t {
	case productcatalog.InArrearsPaymentTerm:
		return api.PricePaymentTermInArrears
	case productcatalog.InAdvancePaymentTerm:
		fallthrough
	default:
		return api.PricePaymentTermInAdvance
	}
}

func FromEntitlementTemplate(t productcatalog.EntitlementTemplate) (api.RateCardEntitlement, error) {
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
			Metadata:                lo.EmptyableToPtr(api.Metadata(metered.Metadata)),
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
			Metadata: lo.EmptyableToPtr(api.Metadata(static.Metadata)),
			Type:     api.RateCardStaticEntitlementTypeStatic,
			Config:   static.Config,
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
			Metadata: lo.EmptyableToPtr(api.Metadata(boolean.Metadata)),
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

	discounts := lo.FromPtrOr(a.Discounts, nil)
	if len(discounts) > 0 {
		phase.Discounts = make([]productcatalog.Discount, 0, len(discounts))

		for _, discount := range discounts {
			switch discount.Type {
			case api.DiscountPercentageTypePercentage:
				percentageDiscount := productcatalog.PercentageDiscount{
					Percentage: discount.Percentage,
					RateCards:  lo.FromPtrOr(discount.RateCards, nil),
				}

				phaseDiscount := productcatalog.NewDiscountFrom(percentageDiscount)
				if err = phaseDiscount.Validate(); err != nil {
					return phase, fmt.Errorf("invalid Discount: %w", err)
				}

				phase.Discounts = append(phase.Discounts, phaseDiscount)
			}
		}
	}

	if len(a.RateCards) > 0 {
		phase.RateCards = make([]productcatalog.RateCard, 0, len(a.RateCards))

		for _, rc := range a.RateCards {
			rateCard, err := AsRateCard(rc)
			if err != nil {
				return phase, fmt.Errorf("failed to cast RateCard: %w", err)
			}

			phase.RateCards = append(phase.RateCards, rateCard)
		}
	}

	return phase, nil
}

func AsRateCard(r api.RateCard) (productcatalog.RateCard, error) {
	rType, err := r.Discriminator()
	if err != nil {
		return nil, fmt.Errorf("failed to cast type: %w", err)
	}

	switch rType {
	case string(productcatalog.FlatFeeRateCardType):
		flat, err := r.AsRateCardFlatFee()
		if err != nil {
			return nil, fmt.Errorf("failed to cast FlatFeeRateCard: %w", err)
		}

		flatRateCard, err := AsFlatFeeRateCard(flat)
		if err != nil {
			return nil, fmt.Errorf("failed to cast FlatFeeRateCard: %w", err)
		}

		return &flatRateCard, nil
	case string(productcatalog.UsageBasedRateCardType):
		usage, err := r.AsRateCardUsageBased()
		if err != nil {
			return nil, fmt.Errorf("failed to cast FlatFeeRateCard: %w", err)
		}

		usageBasedRateCard, err := AsUsageBasedRateCard(usage)
		if err != nil {
			return nil, fmt.Errorf("failed to cast FlatFeeRateCard: %w", err)
		}

		return &usageBasedRateCard, nil
	default:
		return nil, fmt.Errorf("invalid type: %s", rType)
	}
}

func AsFlatFeeRateCard(flat api.RateCardFlatFee) (productcatalog.FlatFeeRateCard, error) {
	var err error

	rc := productcatalog.FlatFeeRateCard{
		RateCardMeta: productcatalog.RateCardMeta{
			Key:         flat.Key,
			Name:        flat.Name,
			Description: flat.Description,
			Metadata:    lo.FromPtrOr(flat.Metadata, nil),
		},
	}

	if flat.BillingCadence != nil {
		isoString := isodate.String(*flat.BillingCadence)
		rc.BillingCadence, err = isoString.ParsePtrOrNil()
		if err != nil {
			return rc, fmt.Errorf("failed to cast BillingCadence: %w", err)
		}
	}

	if flat.FeatureKey != nil {
		rc.RateCardMeta.Feature = &feature.Feature{
			Key: *flat.FeatureKey,
		}
	}

	if flat.EntitlementTemplate != nil {
		tmpl, err := AsEntitlementTemplate(*flat.EntitlementTemplate, rc.BillingCadence)
		if err != nil {
			return productcatalog.FlatFeeRateCard{}, fmt.Errorf("failed to cast EntitlementTemplate: %w", err)
		}

		rc.EntitlementTemplate = tmpl
	}

	if flat.TaxConfig != nil {
		rc.TaxConfig = lo.ToPtr(AsTaxConfig(*flat.TaxConfig))
	}

	if flat.Price != nil {
		amount, err := decimal.NewFromString(flat.Price.Amount)
		if err != nil {
			return rc, fmt.Errorf("failed to cast Price Amount to decimal: %w", err)
		}

		var paymentTerm productcatalog.PaymentTermType
		if flat.Price.PaymentTerm != nil {
			switch *flat.Price.PaymentTerm {
			case api.PricePaymentTermInArrears:
				paymentTerm = productcatalog.InArrearsPaymentTerm
			case api.PricePaymentTermInAdvance:
				paymentTerm = productcatalog.InAdvancePaymentTerm
			default:
				paymentTerm = productcatalog.DefaultPaymentTerm
			}
		} else {
			paymentTerm = productcatalog.DefaultPaymentTerm
		}

		rc.Price = productcatalog.NewPriceFrom(productcatalog.FlatPrice{
			Amount:      amount,
			PaymentTerm: paymentTerm,
		})
	}

	return rc, nil
}

func AsUsageBasedRateCard(usage api.RateCardUsageBased) (productcatalog.UsageBasedRateCard, error) {
	var err error

	rc := productcatalog.UsageBasedRateCard{
		RateCardMeta: productcatalog.RateCardMeta{
			Key:         usage.Key,
			Name:        usage.Name,
			Description: usage.Description,
			Metadata:    lo.FromPtrOr(usage.Metadata, nil),
		},
	}

	isoString := isodate.String(usage.BillingCadence)
	rc.BillingCadence, err = isoString.Parse()
	if err != nil {
		return rc, fmt.Errorf("failed to cast BillingCadence: %w", err)
	}

	if usage.FeatureKey != nil {
		rc.RateCardMeta.Feature = &feature.Feature{
			Key: *usage.FeatureKey,
		}
	}

	if usage.EntitlementTemplate != nil {
		tmpl, err := AsEntitlementTemplate(*usage.EntitlementTemplate, &rc.BillingCadence)
		if err != nil {
			return rc, fmt.Errorf("failed to cast EntitlementTemplate: %w", err)
		}

		rc.EntitlementTemplate = tmpl
	}

	if usage.TaxConfig != nil {
		rc.TaxConfig = lo.ToPtr(AsTaxConfig(*usage.TaxConfig))
	}

	if usage.Price != nil {
		price, err := AsPrice(*usage.Price)
		if err != nil {
			return rc, fmt.Errorf("failed to cast Price: %w", err)
		}

		rc.Price = price
	}

	return rc, nil
}

func AsPrice(p api.RateCardUsageBasedPrice) (*productcatalog.Price, error) {
	var price *productcatalog.Price

	usagePriceType, err := p.Discriminator()
	if err != nil {
		return nil, fmt.Errorf("failed to cast type: %w", err)
	}

	switch usagePriceType {
	case string(api.FlatPriceTypeFlat):
		flat, err := p.AsFlatPriceWithPaymentTerm()
		if err != nil {
			return nil, fmt.Errorf("failed to cast FlatPrice: %w", err)
		}

		flatPrice := productcatalog.FlatPrice{}

		flatPrice.Amount, err = decimal.NewFromString(flat.Amount)
		if err != nil {
			return nil, fmt.Errorf("failed to cast Amount of FlatPrice to decimal: %w", err)
		}

		if flat.PaymentTerm != nil {
			switch *flat.PaymentTerm {
			case api.PricePaymentTermInArrears:
				flatPrice.PaymentTerm = productcatalog.InArrearsPaymentTerm
			case api.PricePaymentTermInAdvance:
				flatPrice.PaymentTerm = productcatalog.InAdvancePaymentTerm
			default:
				flatPrice.PaymentTerm = productcatalog.DefaultPaymentTerm
			}
		}

		price = productcatalog.NewPriceFrom(flatPrice)
	case string(api.UnitPriceTypeUnit):
		unit, err := p.AsUnitPriceWithCommitments()
		if err != nil {
			return nil, fmt.Errorf("failed to cast UnitPrice: %w", err)
		}

		unitPrice := productcatalog.UnitPrice{}

		unitPrice.Amount, err = decimal.NewFromString(unit.Amount)
		if err != nil {
			return price, fmt.Errorf("failed to cast Amount of UnitPrice to decimal: %w", err)
		}

		if unit.MinimumAmount != nil {
			minimumAmount, err := decimal.NewFromString(*unit.MinimumAmount)
			if err != nil {
				return nil, fmt.Errorf("failed to cast MinimumAmount of UnitPrice to decimal: %w", err)
			}

			unitPrice.MinimumAmount = &minimumAmount
		}

		if unit.MaximumAmount != nil {
			maximumAmount, err := decimal.NewFromString(*unit.MaximumAmount)
			if err != nil {
				return nil, fmt.Errorf("failed to cast MaximumAmount of UnitPrice to decimal: %w", err)
			}

			unitPrice.MaximumAmount = &maximumAmount
		}

		price = productcatalog.NewPriceFrom(unitPrice)
	case string(api.TieredPriceWithCommitmentsTypeTiered):
		tiered, err := p.AsTieredPriceWithCommitments()
		if err != nil {
			return price, fmt.Errorf("failed to cast TieredPrice: %w", err)
		}

		tieredPrice := productcatalog.TieredPrice{
			Tiers: nil,
		}

		tieredPrice.Mode, err = productcatalog.NewTieredPriceMode(string(tiered.Mode))
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
			tieredPrice.Tiers = make([]productcatalog.PriceTier, 0, len(tiered.Tiers))
			for _, tier := range tiered.Tiers {
				priceTier, err := AsPriceTier(tier)
				if err != nil {
					return price, fmt.Errorf("failed to cast PriceTier: %w", err)
				}

				tieredPrice.Tiers = append(tieredPrice.Tiers, priceTier)
			}
		}

		price = productcatalog.NewPriceFrom(tieredPrice)
	default:
		return price, fmt.Errorf("invalid Price type for UsageBasedRateCard: %s", usagePriceType)
	}

	return price, nil
}

func AsPriceTier(t api.PriceTier) (productcatalog.PriceTier, error) {
	tier := productcatalog.PriceTier{
		UpToAmount: nil,
		FlatPrice:  nil,
		UnitPrice:  nil,
	}

	if t.UpToAmount != nil {
		upToAmount, err := decimal.NewFromString(*t.UpToAmount)
		if err != nil {
			return tier, fmt.Errorf("invalid UpToAmount for PriceTier: %w", err)
		}

		tier.UpToAmount = &upToAmount
	}

	if t.FlatPrice != nil {
		amount, err := decimal.NewFromString(t.FlatPrice.Amount)
		if err != nil {
			return tier, fmt.Errorf("invalid Amount for FlatPrice component in PriceTier: %w", err)
		}

		tier.FlatPrice = &productcatalog.PriceTierFlatPrice{
			Amount: amount,
		}
	}

	if t.UnitPrice != nil {
		amount, err := decimal.NewFromString(t.UnitPrice.Amount)
		if err != nil {
			return tier, fmt.Errorf("invalid Amount for UnitPrice component in PriceTier: %w", err)
		}

		tier.UnitPrice = &productcatalog.PriceTierUnitPrice{
			Amount: amount,
		}
	}

	return tier, nil
}

func AsEntitlementTemplate(e api.RateCardEntitlement, billingCadence *isodate.Period) (*productcatalog.EntitlementTemplate, error) {
	tmpl := &productcatalog.EntitlementTemplate{}

	eType, err := e.Discriminator()
	if err != nil {
		return nil, fmt.Errorf("failed to cast EntitlementTemplate type: %w", err)
	}

	switch eType {
	case string(api.RateCardMeteredEntitlementTypeMetered):
		metered, err := e.AsRateCardMeteredEntitlement()
		if err != nil {
			return nil, fmt.Errorf("failed to cast Metered EntitlementTemplate: %w", err)
		}

		var usagePeriod isodate.Period

		if metered.UsagePeriod != nil {
			usagePeriodISO := isodate.String(lo.FromPtr(metered.UsagePeriod))

			if usagePeriod, err = usagePeriodISO.Parse(); err != nil {
				return nil, fmt.Errorf("failed to cast UsagePeriod for Metered EntitlementTemplate: %w", err)
			}
		}

		if usagePeriod.IsZero() {
			if billingCadence == nil || billingCadence.IsZero() {
				return nil, models.NewGenericValidationError(
					fmt.Errorf("missing UsagePeriod for Metered EntitlementTemplate where cannot infer from BillingCadence"),
				)
			}

			usagePeriod = *billingCadence
		}

		meteredTemplate := productcatalog.MeteredEntitlementTemplate{
			Metadata:                lo.FromPtrOr(metered.Metadata, nil),
			IsSoftLimit:             lo.FromPtrOr(metered.IsSoftLimit, false),
			IssueAfterReset:         metered.IssueAfterReset,
			IssueAfterResetPriority: metered.IssueAfterResetPriority,
			PreserveOverageAtReset:  metered.PreserveOverageAtReset,
			UsagePeriod:             usagePeriod,
		}

		tmpl = productcatalog.NewEntitlementTemplateFrom(meteredTemplate)
	case string(api.RateCardStaticEntitlementTypeStatic):
		static, err := e.AsRateCardStaticEntitlement()
		if err != nil {
			return tmpl, fmt.Errorf("failed to cast Static EntitlementTemplate: %w", err)
		}

		staticTemplate := productcatalog.StaticEntitlementTemplate{
			Metadata: lo.FromPtrOr(static.Metadata, nil),
			Config:   static.Config,
		}

		tmpl = productcatalog.NewEntitlementTemplateFrom(staticTemplate)
	case string(api.RateCardBooleanEntitlementTypeBoolean):
		boolean, err := e.AsRateCardBooleanEntitlement()
		if err != nil {
			return tmpl, fmt.Errorf("failed to cast Boolean EntitlementTemplate: %w", err)
		}

		booleanTemplate := productcatalog.BooleanEntitlementTemplate{
			Metadata: lo.FromPtrOr(boolean.Metadata, nil),
		}

		tmpl = productcatalog.NewEntitlementTemplateFrom(booleanTemplate)
	default:
		return nil, fmt.Errorf("invalid EntitlementTemplate type: %s", eType)
	}

	return tmpl, nil
}

func AsTaxConfig(c api.TaxConfig) productcatalog.TaxConfig {
	tc := productcatalog.TaxConfig{
		Behavior: (*productcatalog.TaxBehavior)(c.Behavior),
	}

	if c.Stripe != nil {
		tc.Stripe = &productcatalog.StripeTaxConfig{
			Code: c.Stripe.Code,
		}
	}

	return tc
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
