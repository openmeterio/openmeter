package apiconvert

import (
	"fmt"

	"github.com/samber/lo"

	apimodels "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/convert"
)

func ToAPIRateCardUsageBasedPrice(price productcatalog.Price) (apimodels.RateCardUsageBasedPrice, error) {
	var resp apimodels.RateCardUsageBasedPrice

	switch price.Type() {
	case productcatalog.FlatPriceType:
		flatPrice, err := price.AsFlat()
		if err != nil {
			return resp, fmt.Errorf("failed to cast FlatPrice: %w", err)
		}

		err = resp.FromFlatPriceWithPaymentTerm(apimodels.FlatPriceWithPaymentTerm{
			Amount:      flatPrice.Amount.String(),
			PaymentTerm: lo.ToPtr(ToAPIPricePaymentTerm(flatPrice.PaymentTerm)),
			Type:        apimodels.FlatPriceWithPaymentTermTypeFlat,
		})
		if err != nil {
			return resp, fmt.Errorf("failed to cast FlatPrice: %w", err)
		}
	case productcatalog.UnitPriceType:
		unitPrice, err := price.AsUnit()
		if err != nil {
			return resp, fmt.Errorf("failed to cast UnitPrice: %w", err)
		}

		err = resp.FromUnitPriceWithCommitments(apimodels.UnitPriceWithCommitments{
			Amount:        unitPrice.Amount.String(),
			MinimumAmount: convert.StringerPtrToStringPtr(unitPrice.MinimumAmount),
			MaximumAmount: convert.StringerPtrToStringPtr(unitPrice.MaximumAmount),
			Type:          apimodels.UnitPriceWithCommitmentsTypeUnit,
		})
		if err != nil {
			return resp, fmt.Errorf("failed to cast UnitPrice: %w", err)
		}
	case productcatalog.TieredPriceType:
		tieredPrice, err := price.AsTiered()
		if err != nil {
			return resp, fmt.Errorf("failed to cast TieredPrice: %w", err)
		}

		err = resp.FromTieredPriceWithCommitments(apimodels.TieredPriceWithCommitments{
			Type:          apimodels.TieredPriceWithCommitmentsTypeTiered,
			Mode:          apimodels.TieredPriceMode(tieredPrice.Mode),
			MinimumAmount: convert.StringerPtrToStringPtr(tieredPrice.MinimumAmount),
			MaximumAmount: convert.StringerPtrToStringPtr(tieredPrice.MaximumAmount),
			Tiers: lo.Map(tieredPrice.Tiers, func(t productcatalog.PriceTier, _ int) apimodels.PriceTier {
				var upToAmount *apimodels.Numeric
				if t.UpToAmount != nil {
					upToAmount = lo.ToPtr(t.UpToAmount.String())
				}

				var unitPrice *apimodels.UnitPrice
				if t.UnitPrice != nil {
					unitPrice = &apimodels.UnitPrice{
						Type:   apimodels.UnitPriceTypeUnit,
						Amount: t.UnitPrice.Amount.String(),
					}
				}

				var flatPrice *apimodels.FlatPrice
				if t.FlatPrice != nil {
					flatPrice = &apimodels.FlatPrice{
						Type:   apimodels.FlatPriceTypeFlat,
						Amount: t.FlatPrice.Amount.String(),
					}
				}

				return apimodels.PriceTier{
					UpToAmount: upToAmount,
					UnitPrice:  unitPrice,
					FlatPrice:  flatPrice,
				}
			}),
		})
		if err != nil {
			return resp, fmt.Errorf("failed to cast TieredPrice: %w", err)
		}
	case productcatalog.DynamicPriceType:
		dynamicPrice, err := price.AsDynamic()
		if err != nil {
			return resp, fmt.Errorf("failed to cast DynamicPrice: %w", err)
		}

		var multiplier *string
		if !dynamicPrice.Multiplier.Equal(productcatalog.DynamicPriceDefaultMultiplier) {
			multiplier = lo.ToPtr(dynamicPrice.Multiplier.String())
		}

		err = resp.FromDynamicPriceWithCommitments(apimodels.DynamicPriceWithCommitments{
			Type:          apimodels.DynamicPriceWithCommitmentsTypeDynamic,
			MinimumAmount: convert.StringerPtrToStringPtr(dynamicPrice.MinimumAmount),
			MaximumAmount: convert.StringerPtrToStringPtr(dynamicPrice.MaximumAmount),
			Multiplier:    multiplier,
		})
		if err != nil {
			return resp, fmt.Errorf("failed to cast DynamicPrice: %w", err)
		}
	case productcatalog.PackagePriceType:
		packagePrice, err := price.AsPackage()
		if err != nil {
			return resp, fmt.Errorf("failed to cast PackagePrice: %w", err)
		}

		err = resp.FromPackagePriceWithCommitments(apimodels.PackagePriceWithCommitments{
			Type:               apimodels.PackagePriceWithCommitmentsTypePackage,
			Amount:             packagePrice.Amount.String(),
			QuantityPerPackage: packagePrice.QuantityPerPackage.String(),
			MinimumAmount:      convert.StringerPtrToStringPtr(packagePrice.MinimumAmount),
			MaximumAmount:      convert.StringerPtrToStringPtr(packagePrice.MaximumAmount),
		})
		if err != nil {
			return resp, fmt.Errorf("failed to cast PackagePrice: %w", err)
		}
	default:
		return resp, fmt.Errorf("invalid Price type: %s", price.Type())
	}

	return resp, nil
}

func ToAPIPricePaymentTerm(t productcatalog.PaymentTermType) apimodels.PricePaymentTerm {
	switch t {
	case productcatalog.InArrearsPaymentTerm:
		return apimodels.PricePaymentTermInArrears
	case productcatalog.InAdvancePaymentTerm:
		fallthrough
	default:
		return apimodels.PricePaymentTermInAdvance
	}
}

func ToAPITaxConfig(c productcatalog.TaxConfig) apimodels.TaxConfig {
	var stripe *apimodels.StripeTaxConfig

	if c.Stripe != nil {
		stripe = &apimodels.StripeTaxConfig{
			Code: c.Stripe.Code,
		}
	}

	return apimodels.TaxConfig{
		Stripe:    stripe,
		Behavior:  (*apimodels.TaxBehavior)(c.Behavior),
		TaxCodeId: c.TaxCodeID,
	}
}
