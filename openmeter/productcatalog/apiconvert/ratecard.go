package apiconvert

import (
	"fmt"

	"github.com/samber/lo"

	apiv1models "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/convert"
)

func ToAPIRateCardUsageBasedPrice(price productcatalog.Price) (apiv1models.RateCardUsageBasedPrice, error) {
	var resp apiv1models.RateCardUsageBasedPrice

	switch price.Type() {
	case productcatalog.FlatPriceType:
		flatPrice, err := price.AsFlat()
		if err != nil {
			return resp, fmt.Errorf("failed to cast FlatPrice: %w", err)
		}

		err = resp.FromFlatPriceWithPaymentTerm(apiv1models.FlatPriceWithPaymentTerm{
			Amount:      flatPrice.Amount.String(),
			PaymentTerm: lo.ToPtr(ToAPIPricePaymentTerm(flatPrice.PaymentTerm)),
			Type:        apiv1models.FlatPriceWithPaymentTermTypeFlat,
		})
		if err != nil {
			return resp, fmt.Errorf("failed to cast FlatPrice: %w", err)
		}
	case productcatalog.UnitPriceType:
		unitPrice, err := price.AsUnit()
		if err != nil {
			return resp, fmt.Errorf("failed to cast UnitPrice: %w", err)
		}

		err = resp.FromUnitPriceWithCommitments(apiv1models.UnitPriceWithCommitments{
			Amount:        unitPrice.Amount.String(),
			MinimumAmount: convert.StringerPtrToStringPtr(unitPrice.MinimumAmount),
			MaximumAmount: convert.StringerPtrToStringPtr(unitPrice.MaximumAmount),
			Type:          apiv1models.UnitPriceWithCommitmentsTypeUnit,
		})
		if err != nil {
			return resp, fmt.Errorf("failed to cast UnitPrice: %w", err)
		}
	case productcatalog.TieredPriceType:
		tieredPrice, err := price.AsTiered()
		if err != nil {
			return resp, fmt.Errorf("failed to cast TieredPrice: %w", err)
		}

		err = resp.FromTieredPriceWithCommitments(apiv1models.TieredPriceWithCommitments{
			Type:          apiv1models.TieredPriceWithCommitmentsTypeTiered,
			Mode:          apiv1models.TieredPriceMode(tieredPrice.Mode),
			MinimumAmount: convert.StringerPtrToStringPtr(tieredPrice.MinimumAmount),
			MaximumAmount: convert.StringerPtrToStringPtr(tieredPrice.MaximumAmount),
			Tiers: lo.Map(tieredPrice.Tiers, func(t productcatalog.PriceTier, _ int) apiv1models.PriceTier {
				var upToAmount *apiv1models.Numeric
				if t.UpToAmount != nil {
					upToAmount = lo.ToPtr(t.UpToAmount.String())
				}

				var unitPrice *apiv1models.UnitPrice
				if t.UnitPrice != nil {
					unitPrice = &apiv1models.UnitPrice{
						Type:   apiv1models.UnitPriceTypeUnit,
						Amount: t.UnitPrice.Amount.String(),
					}
				}

				var flatPrice *apiv1models.FlatPrice
				if t.FlatPrice != nil {
					flatPrice = &apiv1models.FlatPrice{
						Type:   apiv1models.FlatPriceTypeFlat,
						Amount: t.FlatPrice.Amount.String(),
					}
				}

				return apiv1models.PriceTier{
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

		err = resp.FromDynamicPriceWithCommitments(apiv1models.DynamicPriceWithCommitments{
			Type:          apiv1models.DynamicPriceWithCommitmentsTypeDynamic,
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

		err = resp.FromPackagePriceWithCommitments(apiv1models.PackagePriceWithCommitments{
			Type:               apiv1models.PackagePriceWithCommitmentsTypePackage,
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

func ToAPIPricePaymentTerm(t productcatalog.PaymentTermType) apiv1models.PricePaymentTerm {
	switch t {
	case productcatalog.InArrearsPaymentTerm:
		return apiv1models.PricePaymentTermInArrears
	case productcatalog.InAdvancePaymentTerm:
		fallthrough
	default:
		return apiv1models.PricePaymentTermInAdvance
	}
}

func ToAPITaxConfig(c productcatalog.TaxConfig) apiv1models.TaxConfig {
	var stripe *apiv1models.StripeTaxConfig

	if c.Stripe != nil {
		stripe = &apiv1models.StripeTaxConfig{
			Code: c.Stripe.Code,
		}
	}

	return apiv1models.TaxConfig{
		Stripe:    stripe,
		Behavior:  (*apiv1models.TaxBehavior)(c.Behavior),
		TaxCodeId: c.TaxCodeID,
	}
}
