// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package models

import "time"

type ManagedResource struct {
	NamespacedModel
	ManagedModel

	// ID is the unique identifier for Resource.
	ID string `json:"id"`
	// Key is the unique key for Resource.
	Key string `json:"key"`
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
