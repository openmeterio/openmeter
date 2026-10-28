package subscriptiontestutils

import (
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
)

// GetExampleTaxCodeInput is passed to TaxCodeEnv.CreateTaxCode for the example rate cards' tax code.
// The Stripe mapping is the code those rate cards carried before they referenced tax codes by ID.
func GetExampleTaxCodeInput() taxcode.CreateTaxCodeInput {
	return taxcode.CreateTaxCodeInput{
		AppMappings: taxcode.TaxCodeAppMappings{
			{AppType: app.AppTypeStripe, TaxCode: "txcd_10000000"},
		},
	}
}

// ExampleTaxConfig points rate cards at tc with exclusive tax behavior.
func ExampleTaxConfig(tc taxcode.TaxCode) productcatalog.TaxConfig {
	return productcatalog.TaxConfig{
		TaxCodeID: lo.ToPtr(tc.ID),
		Behavior:  lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
	}
}
