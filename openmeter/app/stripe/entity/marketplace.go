package appstripeentity

import (
	"github.com/openmeterio/openmeter/openmeter/app"
)

var (
	StripeMarketplaceListing = app.MarketplaceListing{
		Type:        app.AppTypeStripe,
		Name:        "Stripe",
		Description: "Send invoices, calculate tax and collect payments.",
		Capabilities: []app.Capability{
			StripeCollectPaymentCapability,
			StripeCalculateTaxCapability,
			StripeInvoiceCustomerCapability,
		},
		InstallMethods: []app.InstallMethod{
			app.InstallMethodAPIKey,
		},
	}

	StripeCollectPaymentCapability = app.Capability{
		Type:        app.CapabilityTypeCollectPayments,
		Key:         "stripe_collect_payment",
		Name:        "Payment",
		Description: "Process payments",
	}

	StripeCalculateTaxCapability = app.Capability{
		Type:        app.CapabilityTypeCalculateTax,
		Key:         "stripe_calculate_tax",
		Name:        "Calculate Tax",
		Description: "Calculate tax for a payment",
	}

	StripeInvoiceCustomerCapability = app.Capability{
		Type:        app.CapabilityTypeInvoiceCustomers,
		Key:         "stripe_invoice_customer",
		Name:        "Invoice Customer",
		Description: "Invoice a customer",
	}
)
