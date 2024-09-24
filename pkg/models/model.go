package models

import "time"

type ManagedUniqueResource struct {
	NamespacedModel
	ManagedModel

	// ID is the unique identifier for Resource.
	ID string `json:"id"`

	// Key is the unique key for Resource.
	Key string `json:"key"`
}

type ManagedResource struct {
	NamespacedModel
	ManagedModel

	// ID is the unique identifier for Resource.
	ID string `json:"id"`
}

type ManagedModel struct {
	CreatedAt time.Time `json:"createdAt"`
	// After creation the entity is considered updated.
	UpdatedAt time.Time `json:"updatedAt"`
	// Time of soft delete. If not null, the entity is considered deleted.
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}

type NamespacedModel struct {
	Namespace string `json:"-" yaml:"-"`
}

type Address struct {
	Country     *CountryCode `json:"country"`
	PostalCode  *string      `json:"postalCode"`
	State       *string      `json:"state"`
	City        *string      `json:"city"`
	Line1       *string      `json:"line1"`
	Line2       *string      `json:"line2"`
	PhoneNumber *string      `json:"phoneNumber"`
}

// Three-letter [ISO4217](https://www.iso.org/iso-4217-currency-codes.html) currency code.
type CurrencyCode string

// [ISO 3166-1](https://www.iso.org/iso-3166-country-codes.html) alpha-2 country code.
type CountryCode string

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
