package provider

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/provider/openmetersandbox"
	"github.com/openmeterio/openmeter/openmeter/billing/provider/stripe"
)

// Type specifies the provider used for billing
type Type string

const (
	// TypeOpenMeterSandbox specifies the OpenMeter billing provider, which is a dummy billing provider mostly useful for testing and
	// initial OpenMeter assessment
	TypeOpenMeterSandbox Type = "openmeter_sandbox"
	// TypeStripe specifies the Stripe billing provider, which is a real billing provider that can be used in production
	TypeStripe Type = "stripe"
)

type Meta struct {
	Type Type `json:"type"`
}

func (t Type) Values() []string {
	return []string{
		string(TypeOpenMeterSandbox),
		string(TypeStripe),
	}
}

type TaxProvider string

var (
	TaxProviderOpenMeterSandbox TaxProvider = TaxProvider(TypeOpenMeterSandbox)
	TaxProviderStripeTax        TaxProvider = TaxProvider(TypeStripe)
)

func (k TaxProvider) Values() []string {
	return []string{
		string(TaxProviderOpenMeterSandbox),
		string(TaxProviderStripeTax),
	}
}

type TaxConfiguration struct {
	Meta

	OpenMeter openmetersandbox.TaxConfiguration
	Stripe    stripe.TaxConfiguration
}

func (c *TaxConfiguration) Validate() error {
	switch c.Type {
	case TypeOpenMeterSandbox:
		return c.OpenMeter.Validate()
	case TypeStripe:
		return c.Stripe.Validate()
	default:
		return fmt.Errorf("unknown tax provider: %s", c.Type)
	}
}

type InvoicingProvider string

var (
	InvoicingProviderOpenMeterSandbox InvoicingProvider = InvoicingProvider(TypeOpenMeterSandbox)
	InvoicingProviderStripeInvoicing  InvoicingProvider = InvoicingProvider(TypeStripe)
)

func (k InvoicingProvider) Values() []string {
	return []string{
		string(InvoicingProviderOpenMeterSandbox),
		string(InvoicingProviderStripeInvoicing),
	}
}

type InvoicingConfiguration struct {
	Meta

	OpenMeter openmetersandbox.InvoicingConfiguration
	Stripe    stripe.InvoicingConfiguration
}

func (c *InvoicingConfiguration) Validate() error {
	switch c.Type {
	case TypeOpenMeterSandbox:
		return c.OpenMeter.Validate()
	case TypeStripe:
		return c.Stripe.Validate()
	default:
		return fmt.Errorf("unknown invoicing provider: %s", c.Type)
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

type PaymentConfiguration struct {
	Meta

	OpenMeter openmetersandbox.PaymentConfiguration
	Stripe    stripe.PaymentConfiguration
}

func (c *PaymentConfiguration) Validate() error {
	switch c.Type {
	case TypeOpenMeterSandbox:
		return c.OpenMeter.Validate()
	case TypeStripe:
		return c.Stripe.Validate()
	default:
		return fmt.Errorf("unknown payment provider: %s", c.Type)
	}
}
