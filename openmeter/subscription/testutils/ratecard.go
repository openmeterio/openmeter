package subscriptiontestutils

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/isodate"
	"github.com/openmeterio/openmeter/pkg/models"
)

var (
	ExampleRateCard1 productcatalog.FlatFeeRateCard = productcatalog.FlatFeeRateCard{
		RateCardMeta: productcatalog.RateCardMeta{
			Key:         ExampleFeatureKey,
			Name:        "Rate Card 1",
			Description: lo.ToPtr("Rate Card 1 Description"),
			Feature: &feature.Feature{
				Key: ExampleFeatureKey,
			},
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
				Amount: alpacadecimal.NewFromInt(int64(ExamplePriceAmount)),
			}),
		},
		BillingCadence: &ISOMonth,
	}
	ExampleRateCard2 productcatalog.FlatFeeRateCard = productcatalog.FlatFeeRateCard{
		RateCardMeta: productcatalog.RateCardMeta{
			Key:         "rate-card-2",
			Name:        "Rate Card 2",
			Description: lo.ToPtr("Rate Card 2 Description"),
			Feature:     nil,
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromInt(int64(0)),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
		},
		BillingCadence: &ISOMonth,
	}
	ExamplePriceAmount int = 100

	ExampleRateCardWithDiscounts productcatalog.FlatFeeRateCard = productcatalog.FlatFeeRateCard{
		RateCardMeta: productcatalog.RateCardMeta{
			Key:         ExampleFeatureKey,
			Name:        "Rate Card 1",
			Description: lo.ToPtr("Rate Card 1 Description"),
			Feature: &feature.Feature{
				Key: ExampleFeatureKey,
			},
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
				Amount: alpacadecimal.NewFromInt(int64(ExamplePriceAmount)),
			}),
			Discounts: productcatalog.Discounts{
				productcatalog.NewDiscountFrom(productcatalog.PercentageDiscount{
					Percentage: models.NewPercentage(10),
				}),
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
