package applistings

import "github.com/openmeterio/openmeter/openmeter/app"

var (
	StripeMarketplaceListing = app.MarketplaceListing{
		Key:         "stripe",
		Name:        "Stripe",
		Description: "Stripe is a payment processing platform.",
		IconURL:     "https://stripe.com/favicon.ico",
		Capabilities: []app.Capability{
			StripeCollectPaymentCapability,
			StripeCalculateTaxCapability,
			StripeInvoiceCustomerCapability,
		},
	}

	StripeCollectPaymentCapability = app.Capability{
		Key:         "stripe_collect_payment",
		Name:        "Payment",
		Description: "Process payments",
		Requirements: []app.Requirement{
			app.RequirementCustomerExternalStripeCustomerId,
		},
	}

	StripeCalculateTaxCapability = app.Capability{
		Key:         "stripe_calculate_tax",
		Name:        "Calculate Tax",
		Description: "Calculate tax for a payment",
		Requirements: []app.Requirement{
			app.RequirementCustomerExternalStripeCustomerId,
		},
	}

	StripeInvoiceCustomerCapability = app.Capability{
		Key:         "stripe_invoice_customer",
		Name:        "Invoice Customer",
		Description: "Invoice a customer",
		Requirements: []app.Requirement{
			app.RequirementCustomerExternalStripeCustomerId,
		},
	}
)
