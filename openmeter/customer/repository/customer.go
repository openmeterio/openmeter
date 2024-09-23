package repository

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/customer"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	customerdb "github.com/openmeterio/openmeter/openmeter/ent/db/customer"
	customersubjectsdb "github.com/openmeterio/openmeter/openmeter/ent/db/customersubjects"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// ListCustomers lists customers
func (r repository) ListCustomers(ctx context.Context, params customer.ListCustomersInput) (pagination.PagedResponse[customer.Customer], error) {
	db := r.client()

	query := db.Customer.
		Query().
		WithSubjects().
		Where(customerdb.Namespace(params.Namespace))

	// Do not return deleted customers by default
	if !params.IncludeDeleted {
		query = query.Where(customerdb.DeletedAtIsNil())
	}

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
			r.logger.Warn("invalid query result: nil customer received")
			continue
		}

		result = append(result, *CustomerFromDBEntity(*item))
	}

	response.TotalCount = paged.TotalCount
	response.Items = result

	return response, nil
}

// CreateCustomer creates a new customer
func (r repository) CreateCustomer(ctx context.Context, params customer.CreateCustomerInput) (*customer.Customer, error) {
	// Create the customer in the database
	query := r.tx.Customer.Create().
		SetNamespace(params.Namespace).
		SetName(params.Name).
		SetNillablePrimaryEmail(params.PrimaryEmail).
		SetNillableCurrency(params.Currency).
		SetNillableTimezone(params.Timezone)

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

	customerEntity, err := query.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create customer: %w", err)
	}

	// Create customer subjects
	// TODO: customer.AddSubjects produces an invalid database query so we create it separately in a transaction.
	// The number and shape of the queries executed is the same, it's a devex thing only.
	customerSubjects, err := r.tx.CustomerSubjects.
		CreateBulk(
			lo.Map(
				params.UsageAttribution.SubjectKeys,
				func(subjectKey string, _ int) *entdb.CustomerSubjectsCreate {
					return r.tx.CustomerSubjects.Create().
						SetNamespace(customerEntity.Namespace).
						SetCustomerID(customerEntity.ID).
						SetSubjectKey(subjectKey)
				},
			)...,
		).
		Save(ctx)
	if err != nil {
		if entdb.IsConstraintError(err) {
			return nil, customer.SubjectKeyConflictError{
				Namespace:   params.Namespace,
				SubjectKeys: params.UsageAttribution.SubjectKeys,
			}
		}

		return nil, fmt.Errorf("failed to create customer: failed to add subject keys: %w", err)
	}

	if customerEntity == nil {
		return nil, fmt.Errorf("invalid query result: nil customer received")
	}

	customerEntity.Edges.Subjects = customerSubjects

	return CustomerFromDBEntity(*customerEntity), nil
}

// DeleteCustomer deletes a customer
func (r repository) DeleteCustomer(ctx context.Context, input customer.DeleteCustomerInput) error {
	db := r.client()

	// Get the customer to resolve the ID if it's a key
	getCustomerInput := customer.GetCustomerInput(input)

	dbCustomer, err := r.GetCustomer(ctx, getCustomerInput)
	if err != nil {
		return err
	}

	// Soft delete the customer
	query := db.Customer.UpdateOneID(dbCustomer.ID).
		Where(customerdb.Namespace(input.Namespace)).
		SetDeletedAt(clock.Now().UTC())

	_, err = query.Save(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			// Construct the customer ID from the database customer
			// to ensure we return the error with ID and not key
			customerId := customer.CustomerID{
				Namespace: dbCustomer.Namespace,
				ID:        dbCustomer.ID,
			}

			if vErr := customerId.Validate(); err != nil {
				return fmt.Errorf("invalid customer ID: %w", vErr)
			}

			return customer.NotFoundError{
				CustomerID: customerId,
			}
		}

		return fmt.Errorf("failed to delete customer: %w", err)
	}

	return nil
}

// GetCustomer gets a customer
func (r repository) GetCustomer(ctx context.Context, input customer.GetCustomerInput) (*customer.Customer, error) {
	db := r.client()

	query := db.Customer.Query().
		WithSubjects().
		Where(customerdb.ID(input.ID)).
		Where(customerdb.Namespace(input.Namespace))

	entity, err := query.First(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, customer.NotFoundError{
				CustomerID: customer.CustomerID(input),
			}
		}

		return nil, fmt.Errorf("failed to fetch customer: %w", err)
	}

	if entity == nil {
		return nil, fmt.Errorf("invalid query result: nil customer received")
	}

	return CustomerFromDBEntity(*entity), nil
}

// UpdateCustomer updates a customer
func (r repository) UpdateCustomer(ctx context.Context, input customer.UpdateCustomerInput) (*customer.Customer, error) {
	getCustomerInput := customer.GetCustomerInput{
		Namespace: input.Namespace,
		ID:        input.ID,
	}

	if err := getCustomerInput.Validate(); err != nil {
		return nil, fmt.Errorf("invalid customer ID: %w", err)
	}

	// Get the customer to diff the subjects
	dbCustomer, err := r.GetCustomer(ctx, getCustomerInput)
	if err != nil {
		return nil, err
	}

	query := r.tx.Customer.UpdateOneID(dbCustomer.ID).
		Where(customerdb.Namespace(input.Namespace)).
		SetUpdatedAt(clock.Now().UTC()).
		SetName(input.Name).
		SetNillablePrimaryEmail(input.PrimaryEmail).
		SetNillableTimezone(input.Timezone).
		SetNillableCurrency(input.Currency)

	if input.BillingAddress != nil {
		query = query.
			SetNillableBillingAddressCity(input.BillingAddress.City).
			SetNillableBillingAddressCountry(input.BillingAddress.Country).
			SetNillableBillingAddressLine1(input.BillingAddress.Line1).
			SetNillableBillingAddressLine2(input.BillingAddress.Line2).
			SetNillableBillingAddressPhoneNumber(input.BillingAddress.PhoneNumber).
			SetNillableBillingAddressPostalCode(input.BillingAddress.PostalCode).
			SetNillableBillingAddressState(input.BillingAddress.State)
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

	// Save the updated customer
	entity, err := query.Save(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, customer.NotFoundError{
				CustomerID: customer.CustomerID{
					Namespace: input.Namespace,
					ID:        input.ID,
				},
			}
		}

		if entdb.IsConstraintError(err) {
			return nil, customer.SubjectKeyConflictError{
				Namespace:   input.Namespace,
				SubjectKeys: input.UsageAttribution.SubjectKeys,
			}
		}

		return nil, fmt.Errorf("failed to update customer: %w", err)
	}

	// Add new subjects
	var subjectsKeysToAdd []string

	for _, subjectKey := range input.UsageAttribution.SubjectKeys {
		found := false

		for _, existingSubjectKey := range dbCustomer.UsageAttribution.SubjectKeys {
			if subjectKey == existingSubjectKey {
				found = true
				continue
			}
		}

		if !found {
			subjectsKeysToAdd = append(subjectsKeysToAdd, subjectKey)
		}
	}

	_, err = r.tx.CustomerSubjects.
		CreateBulk(
			lo.Map(
				subjectsKeysToAdd,
				func(subjectKey string, _ int) *entdb.CustomerSubjectsCreate {
					return r.tx.CustomerSubjects.Create().
						SetNamespace(input.Namespace).
						SetCustomerID(input.ID).
						SetSubjectKey(subjectKey)
				},
			)...,
		).
		Save(ctx)
	if err != nil {
		if entdb.IsConstraintError(err) {
			return nil, customer.SubjectKeyConflictError{
				Namespace:   input.Namespace,
				SubjectKeys: subjectsKeysToAdd,
			}
		}

		return nil, fmt.Errorf("failed to add customer subjects: %w", err)
	}

	// Remove subjects
	var subjectKeysToRemove []string

	for _, existingSubjectKey := range dbCustomer.UsageAttribution.SubjectKeys {
		found := false

		for _, subjectKey := range input.UsageAttribution.SubjectKeys {
			if subjectKey == existingSubjectKey {
				found = true
				continue
			}
		}

		if !found {
			subjectKeysToRemove = append(subjectKeysToRemove, existingSubjectKey)
		}
	}

	_, err = r.tx.CustomerSubjects.
		Delete().
		Where(customersubjectsdb.CustomerID(input.ID)).
		Where(customersubjectsdb.Namespace(input.Namespace)).
		Where(customersubjectsdb.SubjectKeyIn(subjectKeysToRemove...)).
		Exec(ctx)
	if err != nil {
		if entdb.IsConstraintError(err) {
			return nil, customer.SubjectKeyConflictError{
				Namespace:   input.Namespace,
				SubjectKeys: subjectKeysToRemove,
			}
		}

		return nil, fmt.Errorf("failed to remove customer subjects: %w", err)
	}

	if entity == nil {
		return nil, fmt.Errorf("invalid query result: nil customer received")
	}

	// Final subject keys
	entity.Edges.Subjects = []*entdb.CustomerSubjects{}

	// Loop through the existing subjects and add the ones that are not removed
	for _, subjectKey := range dbCustomer.UsageAttribution.SubjectKeys {
		if lo.Contains(subjectKeysToRemove, subjectKey) {
			continue
		}

		entity.Edges.Subjects = append(entity.Edges.Subjects, &entdb.CustomerSubjects{
			Namespace:  input.Namespace,
			CustomerID: input.ID,
			SubjectKey: subjectKey,
		})
	}

	// Add the new subjects
	for _, subjectKey := range subjectsKeysToAdd {
		entity.Edges.Subjects = append(entity.Edges.Subjects, &entdb.CustomerSubjects{
			Namespace:  input.Namespace,
			CustomerID: input.ID,
			SubjectKey: subjectKey,
		})
	}

	return CustomerFromDBEntity(*entity), nil
}
