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

package repository

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/customer"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	customerdb "github.com/openmeterio/openmeter/openmeter/ent/db/customer"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func (r repository) ListCustomers(ctx context.Context, params customer.ListCustomersInput) (pagination.PagedResponse[customer.Customer], error) {
	db := r.client()

	query := db.Customer.Query().
		Where(customerdb.DeletedAtIsNil()) // Do not return deleted customers

	// order := entutils.GetOrdering(sortx.OrderDefault)
	// if !params.Order.IsDefaultValue() {
	// 	order = entutils.GetOrdering(params.Order)
	// }

	// switch params.OrderBy {
	// case customer.CustomerOrderByCreatedAt:
	// 	query = query.Order(customerdb.ByCreatedAt(order...))
	// case customer.CustomerOrderByUpdatedAt:
	// 	query = query.Order(customerdb.ByUpdatedAt(order...))
	// case customer.CustomerOrderByID:
	// 	fallthrough
	// default:
	// 	query = query.Order(customerdb.ByID(order...))
	// }

	response := pagination.PagedResponse[customer.Customer]{
		Page: params.Page,
	}

	paged, err := query.Paginate(ctx, params.Page)
	if err != nil {
		return response, err
	}

	result := make([]customer.Customer, 0, len(paged.Items))
	for _, item := range paged.Items {
		if item == nil {
			r.logger.Warn("invalid query result: nil customer customer received")
			continue
		}

		result = append(result, *CustomerFromDBEntity(*item))
	}

	response.TotalCount = paged.TotalCount
	response.Items = result

	return response, nil
}

func (r repository) CreateCustomer(ctx context.Context, params customer.CreateCustomerInput) (*customer.Customer, error) {
	db := r.client()

	query := db.Customer.Create().
		SetNamespace(params.Namespace).
		SetName(params.Name).
		SetKey(params.Key).
		SetNillablePrimaryEmail(params.PrimaryEmail).
		SetNillableCurrency(params.Currency).
		SetNillableTaxProvider(params.TaxProvider).
		SetNillableInvoicingProvider(params.InvoicingProvider).
		SetNillablePaymentProvider(params.PaymentProvider)

	if params.BillingAddress != nil {
		query = query.
			SetNillableBillingAddressCity(params.BillingAddress.City).
			SetNillableBillingAddressCountry(params.BillingAddress.Country).
			SetNillableBillingAddressLine1(params.BillingAddress.Line1).
			SetNillableBillingAddressLine2(params.BillingAddress.Line2).
			SetNillableBillingAddressPhoneNumber(params.BillingAddress.PhoneNumber).
			SetNillableBillingAddressPostalCode(params.BillingAddress.PostalCode).
			SetNillableBillingAddressState(params.BillingAddress.State)
	}

	for _, subjectKey := range params.UsageAttribution.SubjectKeys {
		query = query.AddSubjects(&entdb.CustomerSubjects{
			CustomerID: params.Key,
			SubjectKey: subjectKey,
		})
	}

	entity, err := query.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create customer customer: %w", err)
	}

	if entity == nil {
		return nil, fmt.Errorf("invalid query result: nil customer customer received")
	}

	return CustomerFromDBEntity(*entity), nil
}

func (r repository) DeleteCustomer(ctx context.Context, params customer.DeleteCustomerInput) error {
	db := r.client()

	query := db.Customer.UpdateOneID(params.ID).
		Where(customerdb.Namespace(params.Namespace)).
		SetDeletedAt(clock.Now().UTC())

	_, err := query.Save(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return customer.NotFoundError{
				CustomerID: customer.CustomerID(params),
			}
		}

		return fmt.Errorf("failed to delete customer customer: %w", err)
	}

	return nil
}

func (r repository) GetCustomer(ctx context.Context, params customer.GetCustomerInput) (*customer.Customer, error) {
	db := r.client()

	query := db.Customer.Query().
		Where(customerdb.ID(params.ID)).
		Where(customerdb.Namespace(params.Namespace))

	entity, err := query.First(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, customer.NotFoundError{
				CustomerID: customer.CustomerID(params),
			}
		}

		return nil, fmt.Errorf("failed to fetch customer customer: %w", err)
	}

	if entity == nil {
		return nil, fmt.Errorf("invalid query result: nil customer customer received")
	}

	return CustomerFromDBEntity(*entity), nil
}

func (r repository) UpdateCustomer(ctx context.Context, params customer.UpdateCustomerInput) (*customer.Customer, error) {
	db := r.client()

	dbCustomer, err := r.GetCustomer(ctx, customer.GetCustomerInput{
		Namespace: params.Namespace,
		ID:        params.ID,
	})
	if err != nil {
		return nil, err
	}

	query := db.Customer.UpdateOneID(params.ID).
		SetUpdatedAt(clock.Now().UTC()).
		SetName(params.Name).
		SetNillablePrimaryEmail(params.PrimaryEmail).
		SetNillableCurrency(params.Currency).
		SetNillableTaxProvider(params.TaxProvider).
		SetNillableInvoicingProvider(params.InvoicingProvider).
		SetNillablePaymentProvider(params.PaymentProvider)

	if params.BillingAddress != nil {
		query = query.
			SetNillableBillingAddressCity(params.BillingAddress.City).
			SetNillableBillingAddressCountry(params.BillingAddress.Country).
			SetNillableBillingAddressLine1(params.BillingAddress.Line1).
			SetNillableBillingAddressLine2(params.BillingAddress.Line2).
			SetNillableBillingAddressPhoneNumber(params.BillingAddress.PhoneNumber).
			SetNillableBillingAddressPostalCode(params.BillingAddress.PostalCode).
			SetNillableBillingAddressState(params.BillingAddress.State)
	} else {
		query = query.
			ClearBillingAddressCity().
			ClearBillingAddressCountry().
			ClearBillingAddressLine1().
			ClearBillingAddressLine2().
			ClearBillingAddressPhoneNumber().
			ClearBillingAddressPostalCode().
			ClearBillingAddressState()
	}

	// Add new subjects
	for _, subjectKey := range params.UsageAttribution.SubjectKeys {
		found := false

		for _, existingSubjectKey := range dbCustomer.UsageAttribution.SubjectKeys {
			if subjectKey == existingSubjectKey {
				found = true
				continue
			}
		}

		if !found {
			query = query.AddSubjects(&entdb.CustomerSubjects{
				CustomerID: params.Key,
				SubjectKey: subjectKey,
			})
		}
	}

	// Remove subjects
	for _, existingSubjectKey := range dbCustomer.UsageAttribution.SubjectKeys {
		found := false

		for _, subjectKey := range params.UsageAttribution.SubjectKeys {
			if subjectKey == existingSubjectKey {
				found = true
				continue
			}
		}

		if !found {
			query = query.RemoveSubjects(&entdb.CustomerSubjects{
				CustomerID: params.Key,
				SubjectKey: existingSubjectKey,
			})
		}
	}

	// Save the updated customer
	entity, err := query.Save(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, customer.NotFoundError{
				CustomerID: customer.CustomerID{
					Namespace: params.Namespace,
					ID:        params.ID,
				},
			}
		}

		return nil, fmt.Errorf("failed to update customer customer: %w", err)
	}

	if entity == nil {
		return nil, fmt.Errorf("invalid query result: nil customer customer received")
	}

	return CustomerFromDBEntity(*entity), nil
}
