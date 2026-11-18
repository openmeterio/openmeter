package http

import (
	"context"
	"errors"
	"fmt"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	apiv1models "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	catalogconvert "github.com/openmeterio/openmeter/openmeter/productcatalog/apiconvert"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
)

// FromRateCard maps a rate card to its v1 shape. taxCodes must hold every Tax Code the rate cards
// reference (see RateCards.TaxCodeIDs): a missing entry silently drops stripe.code from the output.
func FromRateCard(r productcatalog.RateCard, taxCodes map[string]*taxcode.TaxCode) (apiv1models.RateCard, error) {
	var err error

	resp := apiv1models.RateCard{}

	meta := r.AsMeta()

	var featureKey *string
	if meta.Feature != nil {
		featureKey = meta.Feature.Key
	}

	var tmpl apiv1models.RateCardEntitlement
	if meta.EntitlementTemplate != nil {
		tmpl, err = FromEntitlementTemplate(*meta.EntitlementTemplate)
		if err != nil {
			return resp, fmt.Errorf("failed to cast EntitlementTemplate: %w", err)
		}
	}

	var taxConfig *apiv1models.TaxConfig
	if meta.TaxConfig != nil {
		tc, err := ToAPITaxConfig(*meta.TaxConfig, taxCodes)
		if err != nil {
			return resp, fmt.Errorf("failed to map TaxConfig: %w", err)
		}

		taxConfig = lo.ToPtr(tc)
	}

	switch r.Type() {
	case productcatalog.FlatFeeRateCardType:
		var billingCadence *string
		if bc := r.GetBillingCadence(); bc != nil {
			billingCadence = lo.ToPtr(bc.ISOString().String())
		}

		var price *apiv1models.FlatPriceWithPaymentTerm
		if meta.Price != nil {
			flatPrice, err := meta.Price.AsFlat()
			if err != nil {
				return resp, fmt.Errorf("failed to cast FlatPrice: %w", err)
			}

			price = &apiv1models.FlatPriceWithPaymentTerm{
				Amount:      flatPrice.Amount.String(),
				PaymentTerm: lo.ToPtr(FromPaymentTerm(flatPrice.PaymentTerm)),
				Type:        apiv1models.FlatPriceWithPaymentTermTypeFlat,
			}
		}

		err = resp.FromRateCardFlatFee(apiv1models.RateCardFlatFee{
			BillingCadence:      billingCadence,
			Description:         meta.Description,
			EntitlementTemplate: lo.EmptyableToPtr(tmpl),
			FeatureKey:          featureKey,
			Key:                 meta.Key,
			Metadata:            FromMetadata(meta.Metadata),
			Name:                meta.Name,
			Price:               price,
			TaxConfig:           taxConfig,
			Type:                apiv1models.RateCardFlatFeeTypeFlatFee,
			Discounts:           FromDiscounts(meta.Discounts),
		})
		if err != nil {
			return resp, fmt.Errorf("failed to cast FlatPriceRateCard: %w", err)
		}
	case productcatalog.UsageBasedRateCardType:
		var price *apiv1models.RateCardUsageBasedPrice
		if meta.Price != nil {
			ubpPrice, err := FromRateCardUsageBasedPrice(*meta.Price)
			if err != nil {
				return resp, fmt.Errorf("failed to cast UsageBasedPrice: %w", err)
			}

			price = &ubpPrice
		}

		billingCadence := r.GetBillingCadence()
		if billingCadence == nil {
			return resp, errors.New("invalid UsageBasedRateCard: billing cadence must be set")
		}

		err = resp.FromRateCardUsageBased(apiv1models.RateCardUsageBased{
			Type:                apiv1models.RateCardUsageBasedTypeUsageBased,
			BillingCadence:      billingCadence.ISOString().String(),
			Description:         meta.Description,
			EntitlementTemplate: lo.EmptyableToPtr(tmpl),
			FeatureKey:          featureKey,
			Key:                 meta.Key,
			Metadata:            lo.EmptyableToPtr(apiv1models.Metadata(meta.Metadata)),
			Name:                meta.Name,
			Price:               price,
			TaxConfig:           taxConfig,
			Discounts:           FromDiscounts(meta.Discounts),
		})
		if err != nil {
			return resp, fmt.Errorf("failed to cast UsageBasedRateCard: %w", err)
		}
	default:
		return resp, fmt.Errorf("invalid RateCard type: %s", r.Type())
	}

	return resp, nil
}

func FromRateCardUsageBasedPrice(price productcatalog.Price) (apiv1models.RateCardUsageBasedPrice, error) {
	return catalogconvert.ToAPIRateCardUsageBasedPrice(price)
}

func FromTaxConfig(c productcatalog.TaxConfig) apiv1models.TaxConfig {
	return catalogconvert.ToAPITaxConfig(c)
}

// ToAPITaxConfig maps a Tax Code reference to the v1 shape. The stripe.code comes from the
// referenced Tax Code's current Stripe mapping, not the one the record was created with.
// taxCodes must contain every referenced Tax Code.
func ToAPITaxConfig(cfg productcatalog.TaxCodeConfig, taxCodes map[string]*taxcode.TaxCode) (apiv1models.TaxConfig, error) {
	out := apiv1models.TaxConfig{
		Behavior:  (*apiv1models.TaxBehavior)(cfg.Clone().Behavior),
		TaxCodeId: lo.EmptyableToPtr(cfg.TaxCodeID),
	}

	if cfg.TaxCodeID == "" {
		return out, nil
	}

	tc := taxCodes[cfg.TaxCodeID]
	if tc == nil {
		return apiv1models.TaxConfig{}, fmt.Errorf("tax code %s not found in taxCodes", cfg.TaxCodeID)
	}

	if m, ok := tc.GetAppMapping(app.AppTypeStripe); ok {
		out.Stripe = &apiv1models.StripeTaxConfig{Code: m.TaxCode}
	}

	return out, nil
}

func FromPaymentTerm(t productcatalog.PaymentTermType) apiv1models.PricePaymentTerm {
	return catalogconvert.ToAPIPricePaymentTerm(t)
}

func FromEntitlementTemplate(t productcatalog.EntitlementTemplate) (apiv1models.RateCardEntitlement, error) {
	result := apiv1models.RateCardEntitlement{}

	switch t.Type() {
	case entitlement.EntitlementTypeMetered:
		metered, err := t.AsMetered()
		if err != nil {
			return result, fmt.Errorf("failed to cast Metered EntitlementTemplate: %w", err)
		}

		err = result.FromRateCardMeteredEntitlement(apiv1models.RateCardMeteredEntitlement{
			IsSoftLimit:             lo.ToPtr(metered.IsSoftLimit),
			IssueAfterReset:         metered.IssueAfterReset,
			IssueAfterResetPriority: metered.IssueAfterResetPriority,
			Metadata:                lo.EmptyableToPtr(apiv1models.Metadata(metered.Metadata)),
			PreserveOverageAtReset:  metered.PreserveOverageAtReset,
			Type:                    apiv1models.RateCardMeteredEntitlementTypeMetered,
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

		err = result.FromRateCardStaticEntitlement(apiv1models.RateCardStaticEntitlement{
			Metadata: lo.EmptyableToPtr(apiv1models.Metadata(static.Metadata)),
			Type:     apiv1models.RateCardStaticEntitlementTypeStatic,
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

		err = result.FromRateCardBooleanEntitlement(apiv1models.RateCardBooleanEntitlement{
			Metadata: lo.EmptyableToPtr(apiv1models.Metadata(boolean.Metadata)),
			Type:     apiv1models.RateCardBooleanEntitlementTypeBoolean,
		})
		if err != nil {
			return result, fmt.Errorf("failed to cast Boolean EntitlementTemplate: %w", err)
		}
	default:
		return result, fmt.Errorf("invalid type: %s", t.Type())
	}

	return result, nil
}

func FromDiscounts(discounts productcatalog.Discounts) *apiv1models.Discounts {
	if discounts.IsEmpty() {
		return nil
	}

	discountsAPI := apiv1models.Discounts{}
	if discounts.Usage != nil {
		discountsAPI.Usage = &apiv1models.DiscountUsage{
			Quantity: discounts.Usage.Quantity.String(),
		}
	}

	if discounts.Percentage != nil {
		discountsAPI.Percentage = &apiv1models.DiscountPercentage{
			Percentage: discounts.Percentage.Percentage,
		}
	}

	return &discountsAPI
}

func AsRateCards(ctx context.Context, resolver productcatalog.TaxCodeBackfillResolver, namespace string, rcs []apiv1models.RateCard) (productcatalog.RateCards, error) {
	var rateCards productcatalog.RateCards

	if len(rcs) > 0 {
		rateCards = make(productcatalog.RateCards, 0, len(rcs))

		for _, rc := range rcs {
			rateCard, err := AsRateCard(ctx, resolver, namespace, rc)
			if err != nil {
				return nil, fmt.Errorf("failed to cast ratecard: %w", err)
			}

			rateCards = append(rateCards, rateCard)
		}
	}

	return rateCards, nil
}

func AsRateCard(ctx context.Context, resolver productcatalog.TaxCodeBackfillResolver, namespace string, r apiv1models.RateCard) (productcatalog.RateCard, error) {
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

		flatRateCard, err := AsFlatFeeRateCard(ctx, resolver, namespace, flat)
		if err != nil {
			return nil, fmt.Errorf("failed to cast FlatFeeRateCard: %w", err)
		}

		return &flatRateCard, nil
	case string(productcatalog.UsageBasedRateCardType):
		usage, err := r.AsRateCardUsageBased()
		if err != nil {
			return nil, fmt.Errorf("failed to cast UsageBasedRateCard: %w", err)
		}

		usageBasedRateCard, err := AsUsageBasedRateCard(ctx, resolver, namespace, usage)
		if err != nil {
			return nil, fmt.Errorf("failed to cast UsageBasedRateCard: %w", err)
		}

		return &usageBasedRateCard, nil
	default:
		return nil, fmt.Errorf("invalid type: %s", rType)
	}
}

func AsFlatFeeRateCard(ctx context.Context, resolver productcatalog.TaxCodeBackfillResolver, namespace string, flat apiv1models.RateCardFlatFee) (productcatalog.FlatFeeRateCard, error) {
	var err error

	rc := productcatalog.FlatFeeRateCard{
		RateCardMeta: productcatalog.RateCardMeta{
			Key:         flat.Key,
			Name:        flat.Name,
			Description: flat.Description,
			Metadata:    lo.FromPtr(flat.Metadata),
		},
	}

	if flat.BillingCadence != nil {
		isoString := datetime.ISODurationString(*flat.BillingCadence)
		rc.BillingCadence, err = isoString.ParsePtrOrNil()
		if err != nil {
			return rc, fmt.Errorf("failed to cast BillingCadence: %w", err)
		}
	}

	if flat.FeatureKey != nil {
		rc.RateCardMeta.Feature = productcatalog.NewFeatureReference(nil, flat.FeatureKey)
	}

	if flat.EntitlementTemplate != nil {
		tmpl, err := AsEntitlementTemplate(*flat.EntitlementTemplate, rc.BillingCadence)
		if err != nil {
			return productcatalog.FlatFeeRateCard{}, fmt.Errorf("failed to cast EntitlementTemplate: %w", err)
		}

		rc.EntitlementTemplate = tmpl
	}

	if flat.TaxConfig != nil {
		rc.TaxConfig, err = FromAPITaxConfig(ctx, resolver, namespace, *flat.TaxConfig)
		if err != nil {
			return rc, fmt.Errorf("failed to cast TaxConfig: %w", err)
		}
	}

	if flat.Price != nil {
		amount, err := decimal.NewFromString(flat.Price.Amount)
		if err != nil {
			return rc, fmt.Errorf("failed to cast Price Amount to decimal: %w", err)
		}

		var paymentTerm productcatalog.PaymentTermType
		if flat.Price.PaymentTerm != nil {
			switch *flat.Price.PaymentTerm {
			case apiv1models.PricePaymentTermInArrears:
				paymentTerm = productcatalog.InArrearsPaymentTerm
			case apiv1models.PricePaymentTermInAdvance:
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

	discounts, err := AsDiscounts(flat.Discounts)
	if err != nil {
		return rc, fmt.Errorf("failed to cast Discounts: %w", err)
	}

	rc.Discounts = discounts

	return rc, nil
}

func AsUsageBasedRateCard(ctx context.Context, resolver productcatalog.TaxCodeBackfillResolver, namespace string, usage apiv1models.RateCardUsageBased) (productcatalog.UsageBasedRateCard, error) {
	var err error

	rc := productcatalog.UsageBasedRateCard{
		RateCardMeta: productcatalog.RateCardMeta{
			Key:         usage.Key,
			Name:        usage.Name,
			Description: usage.Description,
			Metadata:    lo.FromPtr(usage.Metadata),
		},
	}

	isoString := datetime.ISODurationString(usage.BillingCadence)
	rc.BillingCadence, err = isoString.Parse()
	if err != nil {
		return rc, fmt.Errorf("failed to cast BillingCadence: %w", err)
	}

	if usage.FeatureKey != nil {
		rc.RateCardMeta.Feature = productcatalog.NewFeatureReference(nil, usage.FeatureKey)
	}

	if usage.EntitlementTemplate != nil {
		tmpl, err := AsEntitlementTemplate(*usage.EntitlementTemplate, &rc.BillingCadence)
		if err != nil {
			return rc, fmt.Errorf("failed to cast EntitlementTemplate: %w", err)
		}

		rc.EntitlementTemplate = tmpl
	}

	if usage.TaxConfig != nil {
		rc.TaxConfig, err = FromAPITaxConfig(ctx, resolver, namespace, *usage.TaxConfig)
		if err != nil {
			return rc, fmt.Errorf("failed to cast TaxConfig: %w", err)
		}
	}

	if usage.Price != nil {
		price, err := AsPrice(*usage.Price)
		if err != nil {
			return rc, fmt.Errorf("failed to cast Price: %w", err)
		}

		rc.Price = price
	}

	discounts, err := AsDiscounts(usage.Discounts)
	if err != nil {
		return rc, fmt.Errorf("failed to cast Discounts: %w", err)
	}

	rc.Discounts = discounts

	return rc, nil
}

func AsDiscounts(discounts *apiv1models.Discounts) (productcatalog.Discounts, error) {
	out := productcatalog.Discounts{}

	if discounts == nil {
		return out, nil
	}

	if discounts.Usage != nil {
		usageDiscount, err := AsUsageDiscount(*discounts.Usage)
		if err != nil {
			return out, fmt.Errorf("failed to cast UsageDiscount: %w", err)
		}

		out.Usage = &usageDiscount
	}

	if discounts.Percentage != nil {
		percentageDiscount := AsPercentageDiscount(*discounts.Percentage)
		out.Percentage = &percentageDiscount
	}

	return out, nil
}

func AsUsageDiscount(d apiv1models.DiscountUsage) (productcatalog.UsageDiscount, error) {
	quantity, err := decimal.NewFromString(d.Quantity)
	if err != nil {
		return productcatalog.UsageDiscount{}, fmt.Errorf("failed to parse Quantity of DiscountUsage: %w", err)
	}

	return productcatalog.UsageDiscount{
		Quantity: quantity,
	}, nil
}

func AsPercentageDiscount(d apiv1models.DiscountPercentage) productcatalog.PercentageDiscount {
	return productcatalog.PercentageDiscount{
		Percentage: d.Percentage,
	}
}

func AsPrice(p apiv1models.RateCardUsageBasedPrice) (*productcatalog.Price, error) {
	var price *productcatalog.Price

	usagePriceType, err := p.Discriminator()
	if err != nil {
		return nil, fmt.Errorf("failed to cast type: %w", err)
	}

	switch usagePriceType {
	case string(apiv1models.FlatPriceTypeFlat):
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
			case apiv1models.PricePaymentTermInArrears:
				flatPrice.PaymentTerm = productcatalog.InArrearsPaymentTerm
			case apiv1models.PricePaymentTermInAdvance:
				flatPrice.PaymentTerm = productcatalog.InAdvancePaymentTerm
			default:
				flatPrice.PaymentTerm = productcatalog.DefaultPaymentTerm
			}
		}

		price = productcatalog.NewPriceFrom(flatPrice)
	case string(apiv1models.UnitPriceTypeUnit):
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
	case string(apiv1models.TieredPriceWithCommitmentsTypeTiered):
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
	case string(apiv1models.DynamicPriceWithCommitmentsTypeDynamic):
		dynamic, err := p.AsDynamicPriceWithCommitments()
		if err != nil {
			return price, fmt.Errorf("failed to cast DynamicPrice: %w", err)
		}

		dynamicPrice := productcatalog.DynamicPrice{}

		if dynamic.Multiplier != nil {
			multiplier, err := decimal.NewFromString(*dynamic.Multiplier)
			if err != nil {
				return price, fmt.Errorf("failed to cast Multiplier of DynamicPrice to decimal: %w", err)
			}

			dynamicPrice.Multiplier = multiplier
		} else {
			dynamicPrice.Multiplier = decimal.NewFromInt(1)
		}

		// Commitments
		if dynamic.MinimumAmount != nil {
			minimumAmount, err := decimal.NewFromString(*dynamic.MinimumAmount)
			if err != nil {
				return price, fmt.Errorf("failed to cast MinimumAmount of DynamicPrice to decimal: %w", err)
			}

			dynamicPrice.MinimumAmount = &minimumAmount
		}

		if dynamic.MaximumAmount != nil {
			maximumAmount, err := decimal.NewFromString(*dynamic.MaximumAmount)
			if err != nil {
				return price, fmt.Errorf("failed to cast MaximumAmount of DynamicPrice to decimal: %w", err)
			}

			dynamicPrice.MaximumAmount = &maximumAmount
		}

		price = productcatalog.NewPriceFrom(dynamicPrice)
	case string(apiv1models.PackagePriceWithCommitmentsTypePackage):
		packagePriceAPI, err := p.AsPackagePriceWithCommitments()
		if err != nil {
			return price, fmt.Errorf("failed to cast PackagePrice: %w", err)
		}

		packagePrice := productcatalog.PackagePrice{}

		packagePrice.Amount, err = decimal.NewFromString(packagePriceAPI.Amount)
		if err != nil {
			return price, fmt.Errorf("failed to cast Amount of PackagePrice to decimal: %w", err)
		}

		packagePrice.QuantityPerPackage, err = decimal.NewFromString(packagePriceAPI.QuantityPerPackage)
		if err != nil {
			return price, fmt.Errorf("failed to cast QuantityPerPackage of PackagePrice to decimal: %w", err)
		}

		// Commitments
		if packagePriceAPI.MinimumAmount != nil {
			minimumAmount, err := decimal.NewFromString(*packagePriceAPI.MinimumAmount)
			if err != nil {
				return price, fmt.Errorf("failed to cast MinimumAmount of PackagePrice to decimal: %w", err)
			}

			packagePrice.MinimumAmount = &minimumAmount
		}

		if packagePriceAPI.MaximumAmount != nil {
			maximumAmount, err := decimal.NewFromString(*packagePriceAPI.MaximumAmount)
			if err != nil {
				return price, fmt.Errorf("failed to cast MaximumAmount of PackagePrice to decimal: %w", err)
			}

			packagePrice.MaximumAmount = &maximumAmount
		}

		price = productcatalog.NewPriceFrom(packagePrice)
	default:
		return price, fmt.Errorf("invalid Price type for UsageBasedRateCard: %s", usagePriceType)
	}

	return price, nil
}

func AsPriceTier(t apiv1models.PriceTier) (productcatalog.PriceTier, error) {
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

func AsEntitlementTemplate(e apiv1models.RateCardEntitlement, billingCadence *datetime.ISODuration) (*productcatalog.EntitlementTemplate, error) {
	tmpl := &productcatalog.EntitlementTemplate{}

	eType, err := e.Discriminator()
	if err != nil {
		return nil, fmt.Errorf("failed to cast EntitlementTemplate type: %w", err)
	}

	switch eType {
	case string(apiv1models.RateCardMeteredEntitlementTypeMetered):
		metered, err := e.AsRateCardMeteredEntitlement()
		if err != nil {
			return nil, fmt.Errorf("failed to cast Metered EntitlementTemplate: %w", err)
		}

		var usagePeriod datetime.ISODuration

		if metered.UsagePeriod != nil {
			usagePeriodISO := datetime.ISODurationString(lo.FromPtr(metered.UsagePeriod))

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
			Metadata:                lo.FromPtr(metered.Metadata),
			IsSoftLimit:             lo.FromPtr(metered.IsSoftLimit),
			IssueAfterReset:         metered.IssueAfterReset,
			IssueAfterResetPriority: metered.IssueAfterResetPriority,
			PreserveOverageAtReset:  metered.PreserveOverageAtReset,
			UsagePeriod:             usagePeriod,
		}

		tmpl = productcatalog.NewEntitlementTemplateFrom(meteredTemplate)
	case string(apiv1models.RateCardStaticEntitlementTypeStatic):
		static, err := e.AsRateCardStaticEntitlement()
		if err != nil {
			return tmpl, fmt.Errorf("failed to cast Static EntitlementTemplate: %w", err)
		}

		staticTemplate := productcatalog.StaticEntitlementTemplate{
			Metadata: lo.FromPtr(static.Metadata),
			Config:   static.Config,
		}

		tmpl = productcatalog.NewEntitlementTemplateFrom(staticTemplate)
	case string(apiv1models.RateCardBooleanEntitlementTypeBoolean):
		boolean, err := e.AsRateCardBooleanEntitlement()
		if err != nil {
			return tmpl, fmt.Errorf("failed to cast Boolean EntitlementTemplate: %w", err)
		}

		booleanTemplate := productcatalog.BooleanEntitlementTemplate{
			Metadata: lo.FromPtr(boolean.Metadata),
		}

		tmpl = productcatalog.NewEntitlementTemplateFrom(booleanTemplate)
	default:
		return nil, fmt.Errorf("invalid EntitlementTemplate type: %s", eType)
	}

	return tmpl, nil
}

func AsTaxConfig(c apiv1models.TaxConfig) productcatalog.TaxConfig {
	tc := productcatalog.TaxConfig{
		Behavior:  (*productcatalog.TaxBehavior)(c.Behavior),
		TaxCodeID: c.TaxCodeId,
	}

	if c.Stripe != nil {
		tc.Stripe = &productcatalog.StripeTaxConfig{
			Code: c.Stripe.Code,
		}
	}

	return tc
}

// TaxCodeResolutionError is a failure to resolve a v1 stripe.code to a Tax Code (e.g. a database
// error). It isn't caused by the request, so handlers must not report it as a validation error.
type TaxCodeResolutionError struct {
	Err error
}

func (e *TaxCodeResolutionError) Error() string {
	return e.Err.Error()
}

func (e *TaxCodeResolutionError) Unwrap() error {
	return e.Err
}

// FromAPITaxConfig turns v1 tax config input into a Tax Code reference. taxCodeId wins when set
// and is passed through unchecked: services validate Tax Code references. Otherwise a stripe.code
// is resolved to a Tax Code, creating one if the namespace doesn't have it yet.
func FromAPITaxConfig(ctx context.Context, resolver productcatalog.TaxCodeBackfillResolver, namespace string, in apiv1models.TaxConfig) (*productcatalog.TaxCodeConfig, error) {
	out, stripeCode, err := ParseAPITaxConfig(in)
	if err != nil {
		return nil, err
	}

	if stripeCode != "" {
		if resolver == nil {
			return nil, &TaxCodeResolutionError{Err: errors.New("tax code resolver is required to resolve stripe tax codes")}
		}

		tc, err := resolver.ResolveOrBackfillTaxCode(ctx, productcatalog.ResolveOrBackfillTaxCodeInput{
			Namespace:  namespace,
			StripeCode: stripeCode,
		})
		if err != nil {
			return nil, &TaxCodeResolutionError{Err: fmt.Errorf("failed to resolve tax code for stripe code %s: %w", stripeCode, err)}
		}

		out.TaxCodeID = tc.ID
	}

	if out.TaxCodeID == "" && out.Behavior == nil {
		return nil, nil
	}

	return lo.ToPtr(out.Clone()), nil
}

// ParseAPITaxConfig validates v1 tax config input without any lookups. It returns the Tax Code
// reference and the stripe.code still to resolve, which is empty when taxCodeId wins or no code
// was sent.
//
// stripe.code is format-checked even when taxCodeId wins, because the API always rejected
// malformed codes with a 400.
func ParseAPITaxConfig(in apiv1models.TaxConfig) (productcatalog.TaxCodeConfig, string, error) {
	stripeCode := lo.FromPtr(in.Stripe).Code

	if err := (&productcatalog.StripeTaxConfig{Code: stripeCode}).Validate(); err != nil {
		return productcatalog.TaxCodeConfig{}, "", models.ErrorWithFieldPrefix(
			models.NewFieldSelectorGroup(
				models.NewFieldSelector("taxConfig"),
				models.NewFieldSelector("stripe"),
				models.NewFieldSelector("code"),
			),
			err,
		)
	}

	// An explicit empty ID was always rejected; don't let it fall through to the Stripe code.
	if in.TaxCodeId != nil && *in.TaxCodeId == "" {
		return productcatalog.TaxCodeConfig{}, "", models.ErrorWithFieldPrefix(
			models.NewFieldSelectorGroup(
				models.NewFieldSelector("taxConfig"),
				models.NewFieldSelector("taxCodeId"),
			),
			models.NewGenericValidationError(errors.New("tax code id must not be empty")),
		)
	}

	out := productcatalog.TaxCodeConfig{
		Behavior:  (*productcatalog.TaxBehavior)(in.Behavior),
		TaxCodeID: lo.FromPtr(in.TaxCodeId),
	}

	if out.TaxCodeID != "" {
		return out, "", nil
	}

	return out, stripeCode, nil
}

func FromMetadata(metadata models.Metadata) *apiv1models.Metadata {
	if len(metadata) == 0 {
		return nil
	}

	result := make(apiv1models.Metadata)
	if len(metadata) > 0 {
		for k, v := range metadata {
			result[k] = v
		}
	}

	return &result
}

func FromValidationAttributes(attrs models.Attributes) *apiv1models.Annotations {
	if len(attrs) == 0 {
		return nil
	}

	out := apiv1models.Annotations(attrs.AsStringMap())

	if len(out) == 0 {
		return nil
	}

	return &out
}

func FromAnnotations(annotations models.Annotations) *apiv1models.Annotations {
	if len(annotations) == 0 {
		return nil
	}

	return lo.ToPtr(apiv1models.Annotations(annotations))
}

func FromValidationErrors(issues models.ValidationIssues) *[]apiv1models.ValidationError {
	if len(issues) == 0 {
		return nil
	}

	var result []apiv1models.ValidationError

	for _, issue := range issues {
		result = append(result, apiv1models.ValidationError{
			Message:    issue.Message(),
			Field:      issue.Field().JSONPath(),
			Code:       string(issue.Code()),
			Attributes: FromValidationAttributes(issue.Attributes()),
		})
	}

	return &result
}
