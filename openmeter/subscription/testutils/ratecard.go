package subscriptiontestutils

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/isodate"
	"github.com/openmeterio/openmeter/pkg/models"
)

var (
	ExamplePriceAmount int64                             = 100
	ExampleRateCard1   productcatalog.UsageBasedRateCard = productcatalog.UsageBasedRateCard{
		RateCardMeta: productcatalog.RateCardMeta{
			Key:         ExampleFeatureKey,
			Name:        "Rate Card 1",
			Description: lo.ToPtr("Rate Card 1 Description"),
			FeatureKey:  lo.ToPtr(ExampleFeatureKey),
			FeatureID:   nil,
			EntitlementTemplate: productcatalog.NewEntitlementTemplateFrom(productcatalog.MeteredEntitlementTemplate{
				IssueAfterReset: lo.ToPtr(100.0),
				UsagePeriod:     ISOMonth,
			}),
			TaxConfig: &productcatalog.TaxConfig{
				Stripe: &productcatalog.StripeTaxConfig{
					Code: "txcd_10000000",
				},
			},
			Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromInt(ExamplePriceAmount),
			}),
		},
		BillingCadence: ISOMonth,
	}
	ExampleRateCard2 productcatalog.FlatFeeRateCard = productcatalog.FlatFeeRateCard{
		RateCardMeta: productcatalog.RateCardMeta{
			Key:         "rate-card-2",
			Name:        "Rate Card 2",
			Description: lo.ToPtr("Rate Card 2 Description"),
			FeatureKey:  nil,
			FeatureID:   nil,
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromInt(int64(0)),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
		},
		BillingCadence: &ISOMonth,
	}
	ExampleRateCard3ForAddons productcatalog.FlatFeeRateCard = productcatalog.FlatFeeRateCard{
		RateCardMeta: productcatalog.RateCardMeta{
			Key:         ExampleFeatureKey2,
			Name:        "Rate Card 3 for Addons",
			Description: lo.ToPtr("Rate Card 3 Description"),
			FeatureKey:  lo.ToPtr(ExampleFeatureKey2),
			FeatureID:   nil,
			EntitlementTemplate: productcatalog.NewEntitlementTemplateFrom(productcatalog.MeteredEntitlementTemplate{
				IssueAfterReset: lo.ToPtr(100.0),
				UsagePeriod:     ISOMonth,
			}),
			TaxConfig: &productcatalog.TaxConfig{
				Stripe: &productcatalog.StripeTaxConfig{
					Code: "txcd_10000000",
				},
			},
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromInt(ExamplePriceAmount),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
		},
		BillingCadence: &ISOMonth,
	}
	// Has a boolean entitlement
	ExampleRateCard4ForAddons productcatalog.FlatFeeRateCard = productcatalog.FlatFeeRateCard{
		RateCardMeta: productcatalog.RateCardMeta{
			Key:                 ExampleFeatureKey,
			Name:                "Rate Card 4 for Addons",
			Description:         lo.ToPtr("Rate Card 4 Description"),
			FeatureKey:          lo.ToPtr(ExampleFeatureKey),
			FeatureID:           nil,
			EntitlementTemplate: productcatalog.NewEntitlementTemplateFrom(productcatalog.BooleanEntitlementTemplate{}),
		},
		BillingCadence: &ISOMonth,
	}
	// DOES NOT have a boolean entitlement
	ExampleRateCard5ForAddons productcatalog.FlatFeeRateCard = productcatalog.FlatFeeRateCard{
		RateCardMeta: productcatalog.RateCardMeta{
			Key:                 ExampleFeatureKey,
			Name:                "Rate Card 5 for Addons",
			Description:         lo.ToPtr("Rate Card 5 Description"),
			FeatureKey:          lo.ToPtr(ExampleFeatureKey),
			FeatureID:           nil,
			EntitlementTemplate: nil,
		},
		BillingCadence: &ISOMonth,
	}
	ExampleRateCardWithDiscounts productcatalog.FlatFeeRateCard = productcatalog.FlatFeeRateCard{
		RateCardMeta: productcatalog.RateCardMeta{
			Key:         ExampleFeatureKey,
			Name:        "Rate Card 1",
			Description: lo.ToPtr("Rate Card 1 Description"),
			FeatureKey:  lo.ToPtr(ExampleFeatureKey),
			FeatureID:   nil,
			EntitlementTemplate: productcatalog.NewEntitlementTemplateFrom(productcatalog.MeteredEntitlementTemplate{
				IssueAfterReset: lo.ToPtr(100.0),
				UsagePeriod:     ISOMonth,
			}),
			TaxConfig: &productcatalog.TaxConfig{
				Stripe: &productcatalog.StripeTaxConfig{
					Code: "txcd_10000000",
				},
			},
			Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromInt(ExamplePriceAmount),
			}),
			Discounts: productcatalog.Discounts{
				Percentage: &productcatalog.PercentageDiscount{
					Percentage: models.NewPercentage(10),
				},
			},
		},
		BillingCadence: &ISOMonth,
	}
)

func GetEntitlementTemplateUsagePeriod(t *testing.T, et productcatalog.EntitlementTemplate) *isodate.Period {
	t.Helper()

	switch et.Type() {
	case entitlement.EntitlementTypeMetered:
		if e, err := et.AsMetered(); err == nil {
			return &e.UsagePeriod
		}
	case entitlement.EntitlementTypeStatic:
		return nil
	case entitlement.EntitlementTypeBoolean:
		return nil
	}

	return nil
}
