package appsandbox

import (
	"github.com/openmeterio/openmeter/openmeter/app"
)

var (
	MarketplaceListing = app.MarketplaceListing{
		Type:        app.AppTypeSandbox,
		Name:        "Sandbox",
		Description: "Sandbox can be used to test OpenMeter without external connections.",
		Capabilities: []app.Capability{
			CollectPaymentCapability,
			CalculateTaxCapability,
			InvoiceCustomerCapability,
		},
	}

	CollectPaymentCapability = app.Capability{
		Type:        app.CapabilityTypeCollectPayments,
		Key:         "sandbox_collect_payment",
		Name:        "Payment",
		Description: "Process payments",
	}

	CalculateTaxCapability = app.Capability{
		Type:        app.CapabilityTypeCalculateTax,
		Key:         "sandbox_calculate_tax",
		Name:        "Calculate Tax",
		Description: "Calculate tax for a payment",
	}

	InvoiceCustomerCapability = app.Capability{
		Type:        app.CapabilityTypeInvoiceCustomers,
		Key:         "sandbox_invoice_customer",
		Name:        "Invoice Customer",
		Description: "Invoice a customer",
	}
)
