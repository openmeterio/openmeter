package adapter

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	appobserver "github.com/openmeterio/openmeter/openmeter/app/observer"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	customerdb "github.com/openmeterio/openmeter/openmeter/ent/db/customer"
	customersubjectsdb "github.com/openmeterio/openmeter/openmeter/ent/db/customersubjects"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// Register registers a new observer
func (r *adapter) Register(observer appobserver.Observer[customerentity.Customer]) error {
	for _, o := range *r.observers {
		if o == observer {
			return fmt.Errorf("observer already registered")
		}
	}

	observers := append(*r.observers, observer)
	r.observers = &observers
	return nil
}

// Deregister deregisters an observer
func (r *adapter) Deregister(observer appobserver.Observer[customerentity.Customer]) error {
	for i, o := range *r.observers {
		if o == observer {
			observers := *r.observers
			observers = append(observers[:i], observers[i+1:]...)
			r.observers = &observers
			return nil
		}
	}

	return fmt.Errorf("observer not found")
}

// ListCustomers lists customers
func (r *adapter) ListCustomers(ctx context.Context, params customerentity.ListCustomersInput) (pagination.PagedResponse[customerentity.Customer], error) {
	client := r.DB().Client(ctx)

	query := client.Customer.
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

	response := pagination.PagedResponse[customerentity.Customer]{
		Page: params.Page,
	}

	paged, err := query.Paginate(ctx, params.Page)
	if err != nil {
		return response, err
	}

	result := make([]customerentity.Customer, 0, len(paged.Items))
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
func (r *adapter) CreateCustomer(ctx context.Context, params customerentity.CreateCustomerInput) (*customerentity.Customer, error) {
	client := r.DB().Client(ctx)

	// Create the customer in the database
	query := client.Customer.Create().
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
	customerSubjects, err := client.CustomerSubjects.
		CreateBulk(
			lo.Map(
				params.UsageAttribution.SubjectKeys,
				func(subjectKey string, _ int) *entdb.CustomerSubjectsCreate {
					return client.CustomerSubjects.Create().
						SetNamespace(customerEntity.Namespace).
						SetCustomerID(customerEntity.ID).
						SetSubjectKey(subjectKey)
				},
			)...,
		).
		Save(ctx)
	if err != nil {
		if entdb.IsConstraintError(err) {
			return nil, customerentity.SubjectKeyConflictError{
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
	customer := CustomerFromDBEntity(*customerEntity)

	// TODO: support mapping for apps
	customer.Apps = params.Apps

	// Post-create hook
	for _, observer := range *r.observers {
		if err := observer.PostCreate(ctx, customer); err != nil {
			r.logger.Error("failed to create customer: post-create hook failed", "error", err)
			return nil, fmt.Errorf("failed to create customer: post-create hook failed: %w", err)
		}
	}

	return customer, nil
}

// DeleteCustomer deletes a customer
func (r *adapter) DeleteCustomer(ctx context.Context, input customerentity.DeleteCustomerInput) error {
	client := r.DB().Client(ctx)

	// Soft delete the customer
	query := client.Customer.Update().
		Where(customerdb.ID(input.ID)).
		Where(customerdb.Namespace(input.Namespace)).
		Where(customerdb.DeletedAtIsNil()).
		SetDeletedAt(clock.Now().UTC())

	rows, err := query.Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete customer: %w", err)
	}

	if rows == 0 {
		return customerentity.NotFoundError{
			CustomerID: customerentity.CustomerID(input),
		}
	}

	// Deleted customer
	customer, err := r.GetCustomer(ctx, customerentity.GetCustomerInput(input))
	if err != nil {
		return fmt.Errorf("failed to get deleted customer: %w", err)
	}

	// Post-delete hook
	for _, observer := range *r.observers {
		if err := observer.PostDelete(ctx, customer); err != nil {
			return fmt.Errorf("failed to delete customer: post-delete hook failed: %w", err)
		}
	}

	return nil
}

// GetCustomer gets a customer
func (r *adapter) GetCustomer(ctx context.Context, input customerentity.GetCustomerInput) (*customerentity.Customer, error) {
	client := r.DB().Client(ctx)

	query := client.Customer.Query().
		WithSubjects().
		Where(customerdb.ID(input.ID)).
		Where(customerdb.Namespace(input.Namespace))

	entity, err := query.First(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, customerentity.NotFoundError{
				CustomerID: customerentity.CustomerID(input),
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
func (r *adapter) UpdateCustomer(ctx context.Context, input customerentity.UpdateCustomerInput) (*customerentity.Customer, error) {
	client := r.DB().Client(ctx)

	getCustomerInput := customerentity.GetCustomerInput{
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

	query := client.Customer.UpdateOneID(dbCustomer.ID).
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
			return nil, customerentity.NotFoundError{
				CustomerID: customerentity.CustomerID{
					Namespace: input.Namespace,
					ID:        input.ID,
				},
			}
		}

		if entdb.IsConstraintError(err) {
			return nil, customerentity.SubjectKeyConflictError{
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

	_, err = client.CustomerSubjects.
		CreateBulk(
			lo.Map(
				subjectsKeysToAdd,
				func(subjectKey string, _ int) *entdb.CustomerSubjectsCreate {
					return client.CustomerSubjects.Create().
						SetNamespace(input.Namespace).
						SetCustomerID(input.ID).
						SetSubjectKey(subjectKey)
				},
			)...,
		).
		Save(ctx)
	if err != nil {
		if entdb.IsConstraintError(err) {
			return nil, customerentity.SubjectKeyConflictError{
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

	_, err = client.CustomerSubjects.
		Delete().
		Where(customersubjectsdb.CustomerID(input.ID)).
		Where(customersubjectsdb.Namespace(input.Namespace)).
		Where(customersubjectsdb.SubjectKeyIn(subjectKeysToRemove...)).
		Exec(ctx)
	if err != nil {
		if entdb.IsConstraintError(err) {
			return nil, customerentity.SubjectKeyConflictError{
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

	customer := CustomerFromDBEntity(*entity)

	// Post-update hook
	for _, observer := range *r.observers {
		if err := observer.PostUpdate(ctx, customer); err != nil {
			return nil, fmt.Errorf("failed to update customer: post-update hook failed: %w", err)
		}
	}

	return customer, nil
}
