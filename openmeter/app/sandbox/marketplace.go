package appsandbox

import (
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
)

var (
	MarketplaceListing = appentitybase.MarketplaceListing{
		Type:        appentitybase.AppTypeSandbox,
		Name:        "Sandbox",
		Description: "Sandbox can be used to test OpenMeter without external connections.",
		IconURL:     "https://openmeter.cloud/favicon.ico",
		Capabilities: []appentitybase.Capability{
			CollectPaymentCapability,
			CalculateTaxCapability,
			InvoiceCustomerCapability,
		},
	}

	CollectPaymentCapability = appentitybase.Capability{
		Type:        appentitybase.CapabilityTypeCollectPayments,
		Key:         "sandbox_collect_payment",
		Name:        "Payment",
		Description: "Process payments",
	}

	CalculateTaxCapability = appentitybase.Capability{
		Type:        appentitybase.CapabilityTypeCalculateTax,
		Key:         "sandbox_calculate_tax",
		Name:        "Calculate Tax",
		Description: "Calculate tax for a payment",
	}

	InvoiceCustomerCapability = appentitybase.Capability{
		Type:        appentitybase.CapabilityTypeInvoiceCustomers,
		Key:         "sandbox_invoice_customer",
		Name:        "Invoice Customer",
		Description: "Invoice a customer",
	}
)
