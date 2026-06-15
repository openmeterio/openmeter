package charges

import (
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/handlers/billingprofiles"
	"github.com/openmeterio/openmeter/api/v3/handlers/plans"
	"github.com/openmeterio/openmeter/api/v3/labels"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingcharges "github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// ConvertMetadataToLabels converts domain metadata to API labels.
var ConvertMetadataToLabels = labels.FromMetadata[models.Metadata]

// convertFlatFeeChargeToAPI maps a flatfee.Charge to the API representation.
func convertFlatFeeChargeToAPI(source flatfee.Charge) (api.BillingFlatFeeCharge, error) {
	var price api.BillingPrice
	if err := price.FromBillingPriceFlat(api.BillingPriceFlat{
		Amount: source.ChargeBase.Intent.AmountBeforeProration.String(),
		Type:   api.BillingPriceFlatTypeFlat,
	}); err != nil {
		return api.BillingFlatFeeCharge{}, fmt.Errorf("setting flat fee price union: %w", err)
	}

	return api.BillingFlatFeeCharge{
		AdvanceAfter:           source.State.AdvanceAfter,
		AmountAfterProration:   ConvertDecimalToCurrencyAmount(source.ChargeBase.State.AmountAfterProration),
		BillingPeriod:          ConvertClosedPeriodToAPI(source.ChargeBase.Intent.Intent.BillingPeriod),
		CreatedAt:              source.ChargeBase.ManagedResource.ManagedModel.CreatedAt,
		Currency:               ConvertCurrencyCodeToAPI(source.ChargeBase.Intent.Intent.Currency),
		Customer:               ConvertCustomerIDToReference(source.ChargeBase.Intent.Intent.CustomerID),
		DeletedAt:              source.ChargeBase.ManagedResource.ManagedModel.DeletedAt,
		Description:            source.ChargeBase.Intent.Intent.Description,
		Discounts:              convertFlatFeeDiscounts(source.Intent.PercentageDiscounts),
		FeatureKey:             lo.ToPtr(source.ChargeBase.Intent.FeatureKey),
		FullServicePeriod:      ConvertClosedPeriodToAPI(source.ChargeBase.Intent.Intent.FullServicePeriod),
		Id:                     source.ChargeBase.ManagedResource.ID,
		InvoiceAt:              source.ChargeBase.Intent.InvoiceAt,
		Labels:                 ConvertMetadataToLabels(source.ChargeBase.Intent.Intent.Metadata),
		ManagedBy:              ConvertManagedByToAPI(source.ChargeBase.Intent.Intent.ManagedBy),
		Name:                   source.ChargeBase.Intent.Intent.Name,
		PaymentTerm:            ConvertPaymentTermToAPI(source.ChargeBase.Intent.PaymentTerm),
		Price:                  price,
		ProrationConfiguration: ConvertProRatingConfigToAPI(source.ChargeBase.Intent.ProRating),
		ServicePeriod:          ConvertClosedPeriodToAPI(source.ChargeBase.Intent.Intent.ServicePeriod),
		SettlementMode:         ConvertSettlementModeToAPI(source.ChargeBase.Intent.SettlementMode),
		Status:                 ConvertChargeStatusToAPI(meta.ChargeStatus(source.Status)),
		Subscription:           subscriptionRefPtrToAPI(source.ChargeBase.Intent.Intent.Subscription),
		TaxConfig:              convertTaxCodeConfigToAPI(source.ChargeBase.Intent.Intent.TaxConfig),
		Type:                   api.BillingFlatFeeChargeTypeFlatFee,
		UniqueReferenceId:      source.ChargeBase.Intent.Intent.UniqueReferenceID,
		UpdatedAt:              source.ChargeBase.ManagedResource.ManagedModel.UpdatedAt,
	}, nil
}

// convertUsageBasedChargeToAPI maps a usagebased.Charge to the API representation.
func convertUsageBasedChargeToAPI(source usagebased.Charge) (api.BillingUsageBasedCharge, error) {
	status, err := ConvertUsageBasedStatusToAPI(source.ChargeBase.Status)
	if err != nil {
		return api.BillingUsageBasedCharge{}, fmt.Errorf("converting usage based charge status: %w", err)
	}

	price, err := toAPIBillingPrice(source.Intent.Price)
	if err != nil {
		return api.BillingUsageBasedCharge{}, fmt.Errorf("converting price: %w", err)
	}

	return api.BillingUsageBasedCharge{
		AdvanceAfter:      source.State.AdvanceAfter,
		BillingPeriod:     ConvertClosedPeriodToAPI(source.ChargeBase.Intent.Intent.BillingPeriod),
		CreatedAt:         source.ChargeBase.ManagedResource.ManagedModel.CreatedAt,
		Currency:          ConvertCurrencyCodeToAPI(source.ChargeBase.Intent.Intent.Currency),
		Customer:          ConvertCustomerIDToReference(source.ChargeBase.Intent.Intent.CustomerID),
		DeletedAt:         source.ChargeBase.ManagedResource.ManagedModel.DeletedAt,
		Description:       source.ChargeBase.Intent.Intent.Description,
		Discounts:         convertUsageBasedDiscounts(source.Intent.Discounts),
		FeatureKey:        source.ChargeBase.Intent.FeatureKey,
		FullServicePeriod: ConvertClosedPeriodToAPI(source.ChargeBase.Intent.Intent.FullServicePeriod),
		Id:                source.ChargeBase.ManagedResource.ID,
		InvoiceAt:         source.ChargeBase.Intent.InvoiceAt,
		Labels:            ConvertMetadataToLabels(source.ChargeBase.Intent.Intent.Metadata),
		ManagedBy:         ConvertManagedByToAPI(source.ChargeBase.Intent.Intent.ManagedBy),
		Name:              source.ChargeBase.Intent.Intent.Name,
		Price:             price,
		ServicePeriod:     ConvertClosedPeriodToAPI(source.ChargeBase.Intent.Intent.ServicePeriod),
		SettlementMode:    ConvertSettlementModeToAPI(source.ChargeBase.Intent.SettlementMode),
		Status:            lo.FromPtr(status),
		Subscription:      subscriptionRefPtrToAPI(source.ChargeBase.Intent.Intent.Subscription),
		TaxConfig:         convertTaxCodeConfigToAPI(source.ChargeBase.Intent.Intent.TaxConfig),
		Totals:            convertUsageBasedChargeTotals(source),
		Type:              api.BillingUsageBasedChargeTypeUsageBased,
		UniqueReferenceId: source.ChargeBase.Intent.Intent.UniqueReferenceID,
		UpdatedAt:         source.ChargeBase.ManagedResource.ManagedModel.UpdatedAt,
	}, nil
}

// subscriptionRefPtrToAPI converts a nullable SubscriptionReference pointer to the API type.
func subscriptionRefPtrToAPI(source *meta.SubscriptionReference) *api.BillingSubscriptionReference {
	if source == nil {
		return nil
	}
	ref := ConvertSubscriptionRefToAPI(*source)
	return &ref
}

// convertChargeToAPI dispatches on charge type and maps to the API union type.
func convertChargeToAPI(charge billingcharges.Charge) (api.BillingCharge, error) {
	var out api.BillingCharge

	switch charge.Type() {
	case meta.ChargeTypeFlatFee:
		ff, err := charge.AsFlatFeeCharge()
		if err != nil {
			return out, fmt.Errorf("converting flat fee charge: %w", err)
		}
		apiFF, err := convertFlatFeeChargeToAPI(ff)
		if err != nil {
			return out, err
		}
		if err := out.FromBillingFlatFeeCharge(apiFF); err != nil {
			return out, fmt.Errorf("setting flat fee charge union: %w", err)
		}

	case meta.ChargeTypeUsageBased:
		ub, err := charge.AsUsageBasedCharge()
		if err != nil {
			return out, fmt.Errorf("converting usage based charge: %w", err)
		}
		apiUB, err := convertUsageBasedChargeToAPI(ub)
		if err != nil {
			return out, err
		}
		if err := out.FromBillingUsageBasedCharge(apiUB); err != nil {
			return out, fmt.Errorf("setting usage based charge union: %w", err)
		}

	case meta.ChargeTypeCreditPurchase:
		// Credit purchases are excluded at the query level (ChargeTypes filter) and
		// should never reach this path. Return an error as a defensive measure.
		return out, fmt.Errorf("credit purchase charges are not supported in the charges API")

	default:
		return out, fmt.Errorf("unsupported charge type: %s", charge.Type())
	}

	return out, nil
}

// convertUsageBasedChargeTotals aggregates booked totals from persisted realization runs.
func convertUsageBasedChargeTotals(charge usagebased.Charge) api.BillingChargeTotals {
	out := api.BillingChargeTotals{
		Booked: toAPIBillingTotals(charge.Realizations.Sum()),
	}

	if charge.Expands.RealtimeUsage != nil {
		out.Realtime = lo.ToPtr(toAPIBillingTotals(*charge.Expands.RealtimeUsage))
	}

	return out
}

// toAPIBillingTotals maps a domain totals.Totals to the API BillingTotals type.
func toAPIBillingTotals(t totals.Totals) api.BillingTotals {
	return api.BillingTotals{
		Amount:              t.Amount.String(),
		ChargesTotal:        t.ChargesTotal.String(),
		CreditsTotal:        t.CreditsTotal.String(),
		DiscountsTotal:      t.DiscountsTotal.String(),
		TaxesExclusiveTotal: t.TaxesExclusiveTotal.String(),
		TaxesInclusiveTotal: t.TaxesInclusiveTotal.String(),
		TaxesTotal:          t.TaxesTotal.String(),
		Total:               t.Total.String(),
	}
}

// toAPIBillingPrice maps a domain productcatalog.Price to the API BillingPrice union type.
// DynamicPrice and PackagePrice have no API equivalent and return an error.
func toAPIBillingPrice(p productcatalog.Price) (api.BillingPrice, error) {
	var out api.BillingPrice

	switch p.Type() {
	case productcatalog.FlatPriceType:
		flat, err := p.AsFlat()
		if err != nil {
			return out, fmt.Errorf("reading flat price: %w", err)
		}
		if err := out.FromBillingPriceFlat(api.BillingPriceFlat{
			Amount: flat.Amount.String(),
			Type:   api.BillingPriceFlatTypeFlat,
		}); err != nil {
			return out, fmt.Errorf("setting flat price union: %w", err)
		}

	case productcatalog.UnitPriceType:
		unit, err := p.AsUnit()
		if err != nil {
			return out, fmt.Errorf("reading unit price: %w", err)
		}
		if err := out.FromBillingPriceUnit(api.BillingPriceUnit{
			Amount: unit.Amount.String(),
			Type:   api.BillingPriceUnitTypeUnit,
		}); err != nil {
			return out, fmt.Errorf("setting unit price union: %w", err)
		}

	case productcatalog.TieredPriceType:
		tiered, err := p.AsTiered()
		if err != nil {
			return out, fmt.Errorf("reading tiered price: %w", err)
		}
		tiers := lo.Map(tiered.Tiers, toAPIBillingPriceTier)
		switch tiered.Mode {
		case productcatalog.GraduatedTieredPrice:
			if err := out.FromBillingPriceGraduated(api.BillingPriceGraduated{
				Tiers: tiers,
				Type:  api.BillingPriceGraduatedTypeGraduated,
			}); err != nil {
				return out, fmt.Errorf("setting graduated price union: %w", err)
			}
		case productcatalog.VolumeTieredPrice:
			if err := out.FromBillingPriceVolume(api.BillingPriceVolume{
				Tiers: tiers,
				Type:  api.BillingPriceVolumeTypeVolume,
			}); err != nil {
				return out, fmt.Errorf("setting volume price union: %w", err)
			}
		default:
			return out, fmt.Errorf("unsupported tiered price mode: %s", tiered.Mode)
		}

	default:
		return out, fmt.Errorf("unsupported price type: %s", p.Type())
	}

	return out, nil
}

// toAPIBillingPriceTier maps a domain PriceTier to the API BillingPriceTier type.
func toAPIBillingPriceTier(t productcatalog.PriceTier, _ int) api.BillingPriceTier {
	tier := api.BillingPriceTier{}
	if t.UpToAmount != nil {
		s := t.UpToAmount.String()
		tier.UpToAmount = &s
	}
	if t.FlatPrice != nil {
		tier.FlatPrice = &api.BillingPriceFlat{
			Amount: t.FlatPrice.Amount.String(),
			Type:   api.BillingPriceFlatTypeFlat,
		}
	}
	if t.UnitPrice != nil {
		tier.UnitPrice = &api.BillingPriceUnit{
			Amount: t.UnitPrice.Amount.String(),
			Type:   api.BillingPriceUnitTypeUnit,
		}
	}
	return tier
}

// convertFlatFeeDiscounts maps the optional percentage discount to the anonymous API struct.
func convertFlatFeeDiscounts(pd *productcatalog.PercentageDiscount) *api.BillingFlatFeeDiscounts {
	if pd == nil {
		return nil
	}
	pct := float32(pd.Percentage.InexactFloat64())
	return &api.BillingFlatFeeDiscounts{Percentage: &pct}
}

// convertUsageBasedDiscounts maps usage-based discounts to the API type.
func convertUsageBasedDiscounts(d productcatalog.Discounts) *api.BillingRateCardDiscounts {
	if d.Percentage == nil && d.Usage == nil {
		return nil
	}
	result := &api.BillingRateCardDiscounts{}
	if d.Percentage != nil {
		pct := float32(d.Percentage.Percentage.InexactFloat64())
		result.Percentage = &pct
	}
	if d.Usage != nil {
		s := d.Usage.Quantity.String()
		result.Usage = &s
	}
	return result
}

// ConvertUsageBasedStatusToAPI maps usage-based substates to their top-level API status.
// For example, "active.final_realization.started" maps to "active".
func ConvertUsageBasedStatusToAPI(status usagebased.Status) (*api.BillingChargeStatus, error) {
	s, err := status.ToMetaChargeStatus()
	if err != nil {
		return nil, fmt.Errorf("converting usage-based status to charge status: %w", err)
	}
	return lo.ToPtr(ConvertChargeStatusToAPI(s)), nil
}

// ConvertClosedPeriodToAPI maps a domain ClosedPeriod to the API type.
func ConvertClosedPeriodToAPI(p timeutil.ClosedPeriod) api.ClosedPeriod {
	return api.ClosedPeriod{From: p.From, To: p.To}
}

// ConvertDecimalToCurrencyAmount wraps a decimal amount in a CurrencyAmount.
func ConvertDecimalToCurrencyAmount(d alpacadecimal.Decimal) api.CurrencyAmount {
	return api.CurrencyAmount{Amount: d.String()}
}

// ConvertCustomerIDToReference builds a BillingCustomerReference from a customer ID string.
func ConvertCustomerIDToReference(id string) api.BillingCustomerReference {
	return api.BillingCustomerReference{Id: id}
}

// ConvertProRatingConfigToAPI maps a ProRatingConfig to the API proration configuration.
func ConvertProRatingConfigToAPI(c productcatalog.ProRatingConfig) api.BillingRateCardProrationConfiguration {
	if !c.Enabled {
		return api.BillingRateCardProrationConfiguration{
			Mode: api.BillingRateCardProrationModeNoProration,
		}
	}
	return api.BillingRateCardProrationConfiguration{
		Mode: api.BillingRateCardProrationMode(c.Mode),
	}
}

// ConvertSubscriptionRefToAPI maps a SubscriptionReference to the API type.
func ConvertSubscriptionRefToAPI(ref meta.SubscriptionReference) api.BillingSubscriptionReference {
	var out api.BillingSubscriptionReference
	out.Id = ref.SubscriptionID
	out.Phase.Id = ref.PhaseID
	out.Phase.Item.Id = ref.ItemID

	return out
}

// ConvertChargeStatusToAPI casts a meta.ChargeStatus to api.BillingChargeStatus.
func ConvertChargeStatusToAPI(s meta.ChargeStatus) api.BillingChargeStatus {
	return api.BillingChargeStatus(s)
}

// ConvertSettlementModeToAPI casts a SettlementMode to its API equivalent.
func ConvertSettlementModeToAPI(s productcatalog.SettlementMode) api.BillingSettlementMode {
	return api.BillingSettlementMode(s)
}

// ConvertPaymentTermToAPI casts a PaymentTermType to its API equivalent.
func ConvertPaymentTermToAPI(pt productcatalog.PaymentTermType) api.BillingPricePaymentTerm {
	return api.BillingPricePaymentTerm(pt)
}

// ConvertManagedByToAPI casts an InvoiceLineManagedBy to its API equivalent.
func ConvertManagedByToAPI(mb billing.InvoiceLineManagedBy) api.ResourceManagedBy {
	return api.ResourceManagedBy(mb)
}

// ConvertCurrencyCodeToAPI casts a currencyx.Code to an API CurrencyCode.
func ConvertCurrencyCodeToAPI(c currencyx.Code) api.CurrencyCode {
	return api.CurrencyCode(c)
}

// convertTaxCodeConfigToAPI maps a TaxCodeConfig (Behavior + TaxCodeID) to the API type.
func convertTaxCodeConfigToAPI(cfg *productcatalog.TaxCodeConfig) *api.BillingTaxConfig {
	if cfg == nil {
		return nil
	}
	out := &api.BillingTaxConfig{}
	if cfg.Behavior != nil {
		out.Behavior = lo.ToPtr(api.BillingTaxBehavior(*cfg.Behavior))
	}
	if cfg.TaxCodeID != nil {
		out.TaxCode = &api.TaxCodeReference{Id: *cfg.TaxCodeID}
		out.TaxCodeId = cfg.TaxCodeID
	}
	if out.Behavior == nil && out.TaxCode == nil {
		return nil
	}
	return out
}

// convertAPIChargeStatus maps an API status string to its domain equivalent.
func convertAPIChargeStatus(s string) (meta.ChargeStatus, error) {
	switch api.BillingChargeStatus(s) {
	case api.BillingChargeStatusCreated:
		return meta.ChargeStatusCreated, nil
	case api.BillingChargeStatusActive:
		return meta.ChargeStatusActive, nil
	case api.BillingChargeStatusFinal:
		return meta.ChargeStatusFinal, nil
	case api.BillingChargeStatusDeleted:
		return meta.ChargeStatusDeleted, nil
	default:
		return "", fmt.Errorf("unsupported charge status: %q", s)
	}
}

func convertFlatFeeChargeAPIToIntent(customerID string, flatFee api.CreateFlatFeeChargeRequest) (billingcharges.ChargeIntent, error) {
	var zero billingcharges.ChargeIntent

	taxConfig, err := billingprofiles.FromAPIBillingTaxConfig(flatFee.TaxConfig)
	if err != nil {
		return zero, fmt.Errorf("invalid tax config: %w", err)
	}

	amountBeforeProration, err := alpacadecimal.NewFromString(flatFee.AmountBeforeProration.Amount)
	if err != nil {
		return zero, fmt.Errorf("invalid amount before proration: %w", err)
	}

	var metadata models.Metadata
	if flatFee.Labels != nil {
		metadata = models.Metadata(*flatFee.Labels)
	}

	var discount *productcatalog.PercentageDiscount
	if flatFee.Discounts != nil && flatFee.Discounts.Percentage != nil {
		discount = &productcatalog.PercentageDiscount{
			Percentage: models.NewPercentage(float64(lo.FromPtr(flatFee.Discounts.Percentage))),
		}
	}
	var proRating productcatalog.ProRatingConfig
	if flatFee.ProrationConfiguration.Mode == api.BillingRateCardProrationModeProratePrices {
		proRating = productcatalog.ProRatingConfig{
			Enabled: true,
			Mode:    productcatalog.ProRatingModeProratePrices,
		}
	} else {
		proRating = productcatalog.ProRatingConfig{
			Enabled: false,
		}
	}

	fullServicePeriod := timeutil.ClosedPeriod(flatFee.ServicePeriod)
	if flatFee.FullServicePeriod != nil {
		fullServicePeriod = timeutil.ClosedPeriod(*flatFee.FullServicePeriod)
	}

	billingPeriod := timeutil.ClosedPeriod(flatFee.ServicePeriod)
	if flatFee.BillingPeriod != nil {
		billingPeriod = timeutil.ClosedPeriod(*flatFee.BillingPeriod)
	}

	return billingcharges.NewChargeIntent(flatfee.Intent{
		Intent: meta.Intent{
			Name:              flatFee.Name,
			Description:       flatFee.Description,
			Metadata:          metadata,
			ManagedBy:         billing.ManuallyManagedLine,
			CustomerID:        customerID,
			Currency:          currencyx.Code(flatFee.Currency),
			ServicePeriod:     timeutil.ClosedPeriod(flatFee.ServicePeriod),
			FullServicePeriod: fullServicePeriod,
			BillingPeriod:     billingPeriod,
			TaxConfig:         productcatalog.TaxCodeConfigFrom(taxConfig),
			UniqueReferenceID: flatFee.UniqueReferenceId,
			Subscription:      nil,
		},
		InvoiceAt:             flatFee.InvoiceAt,
		SettlementMode:        productcatalog.SettlementMode(flatFee.SettlementMode),
		PaymentTerm:           productcatalog.PaymentTermType(flatFee.PaymentTerm),
		FeatureKey:            lo.FromPtr(flatFee.FeatureKey),
		PercentageDiscounts:   discount,
		ProRating:             proRating,
		AmountBeforeProration: amountBeforeProration,
	}), nil
}

func convertUsageBaseChargeAPIToIntent(customerID string, usageBasedFee api.CreateUsageBasedChargeRequest) (billingcharges.ChargeIntent, error) {
	var zero billingcharges.ChargeIntent

	taxConfig, err := billingprofiles.FromAPIBillingTaxConfig(usageBasedFee.TaxConfig)
	if err != nil {
		return zero, fmt.Errorf("invalid tax config: %w", err)
	}

	var metadata models.Metadata
	if usageBasedFee.Labels != nil {
		metadata = models.Metadata(*usageBasedFee.Labels)
	}

	var discounts productcatalog.Discounts
	if usageBasedFee.Discounts != nil {
		if usageBasedFee.Discounts.Percentage != nil {
			discounts.Percentage = &productcatalog.PercentageDiscount{
				Percentage: models.NewPercentage(float64(lo.FromPtr(usageBasedFee.Discounts.Percentage))),
			}
		}
		if usageBasedFee.Discounts.Usage != nil {
			quantity, err := alpacadecimal.NewFromString(lo.FromPtr(usageBasedFee.Discounts.Usage))
			if err != nil {
				return zero, fmt.Errorf("invalid usage discount quantity: %w", err)
			}
			discounts.Usage = &productcatalog.UsageDiscount{
				Quantity: quantity,
			}
		}
	}

	price, err := plans.FromAPIBillingPrice(usageBasedFee.Price, lo.ToPtr(api.BillingPricePaymentTermInArrears))
	if err != nil {
		return zero, fmt.Errorf("invalid price: %w", err)
	}

	fullServicePeriod := timeutil.ClosedPeriod(usageBasedFee.ServicePeriod)
	if usageBasedFee.FullServicePeriod != nil {
		fullServicePeriod = timeutil.ClosedPeriod(*usageBasedFee.FullServicePeriod)
	}

	billingPeriod := timeutil.ClosedPeriod(usageBasedFee.ServicePeriod)
	if usageBasedFee.BillingPeriod != nil {
		billingPeriod = timeutil.ClosedPeriod(*usageBasedFee.BillingPeriod)
	}

	return billingcharges.NewChargeIntent(usagebased.Intent{
		Intent: meta.Intent{
			Name:              usageBasedFee.Name,
			Description:       usageBasedFee.Description,
			Metadata:          metadata,
			ManagedBy:         billing.ManuallyManagedLine,
			CustomerID:        customerID,
			Currency:          currencyx.Code(usageBasedFee.Currency),
			ServicePeriod:     timeutil.ClosedPeriod(usageBasedFee.ServicePeriod),
			FullServicePeriod: fullServicePeriod,
			BillingPeriod:     billingPeriod,
			TaxConfig:         productcatalog.TaxCodeConfigFrom(taxConfig),
			UniqueReferenceID: usageBasedFee.UniqueReferenceId,
			Subscription:      nil,
		},
		InvoiceAt:      usageBasedFee.InvoiceAt,
		SettlementMode: productcatalog.SettlementMode(usageBasedFee.SettlementMode),
		FeatureKey:     usageBasedFee.FeatureKey,
		Price:          *price,
		Discounts:      discounts,
	}), nil
}
