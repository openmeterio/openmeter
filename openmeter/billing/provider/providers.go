package provider

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/provider/openmetersandbox"
	"github.com/openmeterio/openmeter/openmeter/billing/provider/stripe"
)

// Type specifies the provider used for billing
type Type string

const (
	// TypeOpenMeter specifies the OpenMeter billing provider, which is a dummy billing provider mostly useful for testing and
	// initial OpenMeter assessment
	TypeOpenMeter Type = "openmeter"
	// TypeStripe specifies the Stripe billing provider, which is a real billing provider that can be used in production
	TypeStripe Type = "stripe"
)

func (t Type) Values() []string {
	return []string{
		string(TypeOpenMeter),
		string(TypeStripe),
	}
}

type Meta struct {
	Type Type `json:"type"`
}

type Configuration struct {
	Meta

	OpenMeterSandbox openmetersandbox.Config `json:"openMeterSandbox"`
	Stripe           stripe.Config           `json:"stripe"`
}

func (c *Configuration) Validate() error {
	switch c.Type {
	case TypeOpenMeter:
		if err := c.OpenMeterSandbox.Validate(); err != nil {
			return fmt.Errorf("failed to validate openmeter configuration: %w", err)
		}

	case TypeStripe:
		if err := c.Stripe.Validate(); err != nil {
			return fmt.Errorf("failed to validate stripe configuration: %w", err)
		}

	default:
		return fmt.Errorf("unknown backend type: %s", c.Type)
	}

	return nil
}

type TaxProvider string

var (
	TaxProviderOpenMeterSandbox TaxProvider = "openmeter_sandbox"
	TaxProviderStripeTax        TaxProvider = "stripe_tax"
)

func (k TaxProvider) Values() []string {
	return []string{
		string(TaxProviderOpenMeterSandbox),
		string(TaxProviderStripeTax),
	}
}

type InvoicingProvider string

var (
	InvoicingProviderOpenMeterSandbox InvoicingProvider = "openmeter_sandbox"
	InvoicingProviderStripeInvoicing  InvoicingProvider = "stripe_invoicing"
)

func (k InvoicingProvider) Values() []string {
	return []string{
		string(InvoicingProviderOpenMeterSandbox),
		string(InvoicingProviderStripeInvoicing),
	}
}

type PaymentProvider string

var (
	PaymentProviderOpenMeterSandbox PaymentProvider = "openmeter_sandbox"
	PaymentProviderStripePayments   PaymentProvider = "stripe_payments"
)

func (k PaymentProvider) Values() []string {
	return []string{
		string(PaymentProviderOpenMeterSandbox),
		string(PaymentProviderStripePayments),
	}
}