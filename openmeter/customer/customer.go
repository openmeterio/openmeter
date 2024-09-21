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

package customer

import (
	"context"
	"errors"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// Customer represents a customer
type Customer struct {
	models.ManagedResource

	Name              string                    `json:"name"`
	UsageAttribution  CustomerUsageAttribution  `json:"usageAttribution"`
	PrimaryEmail      *string                   `json:"primaryEmail"`
	Currency          *models.CurrencyCode      `json:"currency"`
	BillingAddress    *models.Address           `json:"billingAddress"`
	TaxProvider       *models.TaxProvider       `json:"taxProvider"`
	InvoicingProvider *models.InvoicingProvider `json:"invoicingProvider"`
	PaymentProvider   *models.PaymentProvider   `json:"paymentProvider"`
	External          *CustomerExternalMapping  `json:"external"`
}

type CustomerID models.NamespacedID

func (i CustomerID) Validate() error {
	if i.Namespace == "" {
		return ValidationError{
			Err: errors.New("namespace is required"),
		}
	}

	if i.ID == "" {
		return ValidationError{
			Err: errors.New("customer id is required"),
		}
	}

	return nil
}

// CustomerUsageAttribution represents the usage attribution for a customer
type CustomerUsageAttribution struct {
	SubjectKeys []string
}

// CustomerExternalMapping represents the external mapping for a customer
type CustomerExternalMapping struct {
	StripeCustomerID *string `json:"stripeCustomerID"`
}

// ListCustomersInput represents the input for the ListCustomers method
type ListCustomersInput struct {
	Namespace string
	pagination.Page

	IncludeDisabled bool
}

// CreateCustomerInput represents the input for the CreateCustomer method
type CreateCustomerInput struct {
	Namespace string
	Customer
}

func (i CreateCustomerInput) Validate(_ context.Context, _ Service) error {
	if i.Namespace == "" {
		return ValidationError{
			Err: errors.New("namespace is required"),
		}
	}

	if i.Key == "" {
		return ValidationError{
			Err: errors.New("customer key is required"),
		}
	}

	if i.Name == "" {
		return ValidationError{
			Err: errors.New("customer name is required"),
		}
	}

	return nil
}

// UpdateCustomerInput represents the input for the UpdateCustomer method
type UpdateCustomerInput struct {
	Namespace string
	Customer
}

func (i UpdateCustomerInput) Validate(_ context.Context, _ Service) error {
	if i.Namespace == "" {
		return ValidationError{
			Err: errors.New("namespace is required"),
		}
	}

	if i.ID == "" {
		return ValidationError{
			Err: errors.New("customer id is required"),
		}
	}

	return nil
}

// DeleteCustomerInput represents the input for the DeleteCustomer method
type DeleteCustomerInput CustomerID

// GetCustomerInput represents the input for the GetCustomer method
type GetCustomerInput CustomerID
