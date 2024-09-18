package models

import "time"

type ManagedResource struct {
	NamespacedModel
	ManagedModel

	// ID is the unique identifier for Resource.
	ID string `json:"id"`
	// Key is the unique key for Resource.
	Key string `json:"key"`
	// Name is the name of the Resource.
	Name string `json:"name"`
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
	Country     *string `json:"country"`
	PostalCode  *string `json:"postalCode"`
	State       *string `json:"state"`
	City        *string `json:"city"`
	Line1       *string `json:"line1"`
	Line2       *string `json:"line2"`
	PhoneNumber *string `json:"phoneNumber"`
}

type CurrencyCode string

type TaxProvider string

var (
	TaxProviderOpenMeterTest TaxProvider = "openmeter_test"
	TaxProviderStripeTax     TaxProvider = "stripe_tax"
)

func (k TaxProvider) Values() []string {
	return []string{
		string(TaxProviderOpenMeterTest),
		string(TaxProviderStripeTax),
	}
}

type InvoicingProvider string

var (
	InvoicingProviderOpenMeterTest   InvoicingProvider = "openmeter_test"
	InvoicingProviderStripeInvoicing InvoicingProvider = "stripe_invoicing"
)

func (k InvoicingProvider) Values() []string {
	return []string{
		string(InvoicingProviderOpenMeterTest),
		string(InvoicingProviderStripeInvoicing),
	}
}

type PaymentProvider string

var (
	PaymentProviderOpenMeterTest  PaymentProvider = "openmeter_test"
	PaymentProviderStripePayments PaymentProvider = "stripe_payments"
)

func (k PaymentProvider) Values() []string {
	return []string{
		string(PaymentProviderOpenMeterTest),
		string(PaymentProviderStripePayments),
	}
}
