package appadapter

import "github.com/openmeterio/openmeter/openmeter/app"

var StripeCollectPaymentCapability = app.Capability{
	Key:         "stripe_collect_payment",
	Name:        "Payment",
	Description: "Process payments",
	Requirements: []app.Requirement{
		app.RequirementCustomerExternalStripeCustomerId,
	},
}

var StripeCalculateTaxCapability = app.Capability{
	Key:         "stripe_calculate_tax",
	Name:        "Calculate Tax",
	Description: "Calculate tax for a payment",
	Requirements: []app.Requirement{
		app.RequirementCustomerExternalStripeCustomerId,
	},
}

var StripeInvoiceCustomerCapability = app.Capability{
	Key:         "stripe_invoice_customer",
	Name:        "Invoice Customer",
	Description: "Invoice a customer",
	Requirements: []app.Requirement{
		app.RequirementCustomerExternalStripeCustomerId,
	},
}

var MarketplaceListings = map[string]app.MarketplaceListing{
	"stripe": {
		Key:         "stripe",
		Name:        "Stripe",
		Description: "Stripe is a payment processing platform.",
		IconURL:     "https://stripe.com/favicon.ico",
		Capabilities: []app.Capability{
			StripeCollectPaymentCapability,
			StripeCalculateTaxCapability,
			StripeInvoiceCustomerCapability,
		},
	},
}
