package subscriptiontestutils

import (
	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
)

var (
	ExampleRateCard1   subscription.RateCard
	ExampleRateCard2   subscription.RateCard
	ExamplePriceAmount int = 100
)

func init() {
	p1 := productcatalog.Price{}

	p1.FromUnit(productcatalog.UnitPrice{
		Amount: alpacadecimal.NewFromInt(int64(ExamplePriceAmount)),
	})

	e1 := productcatalog.EntitlementTemplate{}

	e1.FromMetered(productcatalog.MeteredEntitlementTemplate{
		IssueAfterReset: lo.ToPtr(100.0),
		UsagePeriod:     ISOMonth,
	})

	ExampleRateCard1 = subscription.RateCard{
		Name:                "Rate Card 1",
		Description:         lo.ToPtr("Rate Card 1 Description"),
		EntitlementTemplate: &e1,
		FeatureKey:          &ExampleFeatureKey,
		Price:               &p1,
		BillingCadence:      &ISOMonth,
		TaxConfig: &productcatalog.TaxConfig{
			Stripe: &productcatalog.StripeTaxConfig{
				Code: "txcd_10000000",
			},
		},
	}
	ExampleRateCard2 = subscription.RateCard{
		Name:        "Rate Card 2",
		Description: lo.ToPtr("Rate Card 2 Description"),
		Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
			Amount:      alpacadecimal.NewFromInt(int64(0)),
			PaymentTerm: productcatalog.InAdvancePaymentTerm,
		}),
		BillingCadence: &ISOMonth,
		TaxConfig: &productcatalog.TaxConfig{
			Stripe: &productcatalog.StripeTaxConfig{
				Code: "txcd_10000000",
			},
		},
	}
}
