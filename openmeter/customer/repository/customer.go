package repository

import (
	"context"
	"fmt"

	customer_model "github.com/openmeterio/openmeter/openmeter/customer"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	customerdb "github.com/openmeterio/openmeter/openmeter/ent/db/customer"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func (r repository) ListCustomers(ctx context.Context, params customer_model.ListCustomersInput) (pagination.PagedResponse[customer_model.Customer], error) {
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

	response := pagination.PagedResponse[customer_model.Customer]{
		Page: params.Page,
	}

	paged, err := query.Paginate(ctx, params.Page)
	if err != nil {
		return response, err
	}

	result := make([]customer_model.Customer, 0, len(paged.Items))
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

func (r repository) CreateCustomer(ctx context.Context, params customer_model.CreateCustomerInput) (*customer_model.Customer, error) {
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

	if params.Address != nil {
		query = query.
			SetNillableAddressCity(params.Address.City).
			SetNillableAddressCountry(params.Address.Country).
			SetNillableAddressLine1(params.Address.Line1).
			SetNillableAddressLine2(params.Address.Line2).
			SetNillableAddressPhoneNumber(params.Address.PhoneNumber).
			SetNillableAddressPostalCode(params.Address.PostalCode).
			SetNillableAddressState(params.Address.State)
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

func (r repository) DeleteCustomer(ctx context.Context, params customer_model.DeleteCustomerInput) error {
	db := r.client()

	query := db.Customer.UpdateOneID(params.ID).
		SetDeletedAt(clock.Now().UTC())

	_, err := query.Save(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return customer_model.NotFoundError{
				NamespacedID: models.NamespacedID{
					Namespace: params.Namespace,
					ID:        params.ID,
				},
			}
		}

		return fmt.Errorf("failed to delete customer customer: %w", err)
	}

	return nil
}

func (r repository) GetCustomer(ctx context.Context, params customer_model.GetCustomerInput) (*customer_model.Customer, error) {
	db := r.client()

	query := db.Customer.Query().
		Where(customerdb.ID(params.ID)).
		Where(customerdb.Namespace(params.Namespace))

	entity, err := query.First(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, customer_model.NotFoundError{
				NamespacedID: models.NamespacedID{
					Namespace: params.Namespace,
					ID:        params.ID,
				},
			}
		}

		return nil, fmt.Errorf("failed to fetch customer customer: %w", err)
	}

	if entity == nil {
		return nil, fmt.Errorf("invalid query result: nil customer customer received")
	}

	return CustomerFromDBEntity(*entity), nil
}

func (r repository) UpdateCustomer(ctx context.Context, params customer_model.UpdateCustomerInput) (*customer_model.Customer, error) {
	db := r.client()

	customer, err := r.GetCustomer(ctx, customer_model.GetCustomerInput{
		NamespacedModel: params.NamespacedModel,
		ID:              params.ID,
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

	if params.Address != nil {
		query = query.
			SetNillableAddressCity(params.Address.City).
			SetNillableAddressCountry(params.Address.Country).
			SetNillableAddressLine1(params.Address.Line1).
			SetNillableAddressLine2(params.Address.Line2).
			SetNillableAddressPhoneNumber(params.Address.PhoneNumber).
			SetNillableAddressPostalCode(params.Address.PostalCode).
			SetNillableAddressState(params.Address.State)
	} else {
		query = query.
			ClearAddressCity().
			ClearAddressCountry().
			ClearAddressLine1().
			ClearAddressLine2().
			ClearAddressPhoneNumber().
			ClearAddressPostalCode().
			ClearAddressState()
	}

	// Add new subjects
	for _, subjectKey := range params.UsageAttribution.SubjectKeys {
		found := false

		for _, existingSubjectKey := range customer.UsageAttribution.SubjectKeys {
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
	for _, existingSubjectKey := range customer.UsageAttribution.SubjectKeys {
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
			return nil, customer_model.NotFoundError{
				NamespacedID: models.NamespacedID{
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
