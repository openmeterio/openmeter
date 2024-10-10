package appstripeentity

import (
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
)

var (
	StripeMarketplaceListing = appentitybase.MarketplaceListing{
		Type:        appentitybase.AppTypeStripe,
		Name:        "Stripe",
		Description: "Stripe is a payment processing platform.",
		IconURL:     "https://stripe.com/favicon.ico",
		Capabilities: []appentitybase.Capability{
			StripeCollectPaymentCapability,
			StripeCalculateTaxCapability,
			StripeInvoiceCustomerCapability,
		},
	}

	StripeCollectPaymentCapability = appentitybase.Capability{
		Type:        appentitybase.CapabilityTypeCollectPayments,
		Key:         "stripe_collect_payment",
		Name:        "Payment",
		Description: "Process payments",
	}

	StripeCalculateTaxCapability = appentitybase.Capability{
		Type:        appentitybase.CapabilityTypeCalculateTax,
		Key:         "stripe_calculate_tax",
		Name:        "Calculate Tax",
		Description: "Calculate tax for a payment",
	}

	StripeInvoiceCustomerCapability = appentitybase.Capability{
		Type:        appentitybase.CapabilityTypeInvoiceCustomers,
		Key:         "stripe_invoice_customer",
		Name:        "Invoice Customer",
		Description: "Invoice a customer",
	}
)
