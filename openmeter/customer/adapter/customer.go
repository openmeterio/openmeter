package adapter

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	customerdb "github.com/openmeterio/openmeter/openmeter/ent/db/customer"
	customersubjectsdb "github.com/openmeterio/openmeter/openmeter/ent/db/customersubjects"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

// ListCustomers lists customers
func (a *adapter) ListCustomers(ctx context.Context, input customerentity.ListCustomersInput) (pagination.PagedResponse[customerentity.Customer], error) {
	return entutils.TransactingRepo(
		ctx,
		a,
		func(ctx context.Context, repo *adapter) (pagination.PagedResponse[customerentity.Customer], error) {
			if err := input.Validate(); err != nil {
				return pagination.PagedResponse[customerentity.Customer]{}, customerentity.ValidationError{
					Err: err,
				}
			}

			// Build the database query
			query := repo.db.Customer.
				Query().
				WithSubjects(func(query *entdb.CustomerSubjectsQuery) {
					query.Where(customersubjectsdb.IsDeletedEQ(false))
				}).
				Where(customerdb.Namespace(input.Namespace))

			// Do not return deleted customers by default
			if !input.IncludeDeleted {
				query = query.Where(customerdb.DeletedAtIsNil())
			}

			// Filters
			if input.Name != nil {
				query = query.Where(customerdb.NameContainsFold(*input.Name))
			}

			if input.PrimaryEmail != nil {
				query = query.Where(customerdb.PrimaryEmailContainsFold(*input.PrimaryEmail))
			}

			if input.Subject != nil {
				query = query.Where(customerdb.HasSubjectsWith(customersubjectsdb.SubjectKeyContainsFold(*input.Subject)))
			}

			// Order
			order := entutils.GetOrdering(sortx.OrderDefault)
			if !input.Order.IsDefaultValue() {
				order = entutils.GetOrdering(input.Order)
			}

			switch input.OrderBy {
			case api.CustomerOrderById:
				query = query.Order(customerdb.ByID(order...))
			case api.CustomerOrderByCreatedAt:
				query = query.Order(customerdb.ByCreatedAt(order...))
			case api.CustomerOrderByName:
				fallthrough
			default:
				query = query.Order(customerdb.ByName(order...))
			}

			// Response
			response := pagination.PagedResponse[customerentity.Customer]{
				Page: input.Page,
			}

			paged, err := query.Paginate(ctx, input.Page)
			if err != nil {
				return response, err
			}

			result := make([]customerentity.Customer, 0, len(paged.Items))
			for _, item := range paged.Items {
				if item == nil {
					a.logger.Warn("invalid query result: nil customer received")
					continue
				}

				result = append(result, *CustomerFromDBEntity(*item))
			}

			response.TotalCount = paged.TotalCount
			response.Items = result

			return response, nil
		})
}

// CreateCustomer creates a new customer
func (a *adapter) CreateCustomer(ctx context.Context, input customerentity.CreateCustomerInput) (*customerentity.Customer, error) {
	return entutils.TransactingRepo(
		ctx,
		a,
		func(ctx context.Context, repo *adapter) (*customerentity.Customer, error) {
			if err := input.Validate(); err != nil {
				return nil, customerentity.ValidationError{
					Err: fmt.Errorf("error creating customer: %w", err),
				}
			}

			// Create the customer in the database
			query := repo.db.Customer.Create().
				SetNamespace(input.Namespace).
				SetName(input.Name).
				SetNillableDescription(input.Description).
				SetNillablePrimaryEmail(input.PrimaryEmail).
				SetNillableCurrency(input.Currency).
				SetNillableTimezone(input.Timezone)

			if input.BillingAddress != nil {
				query = query.
					SetNillableBillingAddressCity(input.BillingAddress.City).
					SetNillableBillingAddressCountry(input.BillingAddress.Country).
					SetNillableBillingAddressLine1(input.BillingAddress.Line1).
					SetNillableBillingAddressLine2(input.BillingAddress.Line2).
					SetNillableBillingAddressPhoneNumber(input.BillingAddress.PhoneNumber).
					SetNillableBillingAddressPostalCode(input.BillingAddress.PostalCode).
					SetNillableBillingAddressState(input.BillingAddress.State)
			}

			customerEntity, err := query.Save(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to create customer: %w", err)
			}

			// Create customer subjects
			// TODO: customer.AddSubjects produces an invalid database query so we create it separately in a transaction.
			// The number and shape of the queries executed is the same, it's a devex thing only.
			customerSubjects, err := repo.db.CustomerSubjects.
				CreateBulk(
					lo.Map(
						input.UsageAttribution.SubjectKeys,
						func(subjectKey string, _ int) *entdb.CustomerSubjectsCreate {
							return repo.db.CustomerSubjects.Create().
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
						Namespace:   input.Namespace,
						SubjectKeys: input.UsageAttribution.SubjectKeys,
					}
				}

				return nil, fmt.Errorf("failed to create customer: failed to add subject keys: %w", err)
			}

			if customerEntity == nil {
				return nil, fmt.Errorf("invalid query result: nil customer received")
			}

			customerEntity.Edges.Subjects = customerSubjects
			customer := CustomerFromDBEntity(*customerEntity)

			return customer, nil
		},
	)
}

// DeleteCustomer deletes a customer
func (a *adapter) DeleteCustomer(ctx context.Context, input customerentity.DeleteCustomerInput) error {
	_, err := entutils.TransactingRepo(
		ctx,
		a,
		func(ctx context.Context, repo *adapter) (any, error) {
			if err := input.Validate(); err != nil {
				return nil, customerentity.ValidationError{
					Err: fmt.Errorf("error deleting customer: %w", err),
				}
			}

			deletedAt := clock.Now().UTC()

			// Soft delete the customer
			rows, err := repo.db.Customer.Update().
				Where(customerdb.ID(input.ID)).
				Where(customerdb.Namespace(input.Namespace)).
				Where(customerdb.DeletedAtIsNil()).
				SetDeletedAt(deletedAt).
				Save(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to delete customer: %w", err)
			}

			if rows == 0 {
				return nil, customerentity.NotFoundError{
					CustomerID: customerentity.CustomerID(input),
				}
			}

			// Soft delete the customer subjects
			err = repo.db.CustomerSubjects.
				Update().
				Where(customersubjectsdb.CustomerID(input.ID)).
				Where(customersubjectsdb.Namespace(input.Namespace)).
				Where(customersubjectsdb.IsDeletedEQ(false)).
				SetIsDeleted(true).
				SetDeletedAt(deletedAt).
				Exec(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to delete customer subjects: %w", err)
			}

			return nil, nil
		},
	)

	return err
}

// GetCustomer gets a customer
func (a *adapter) GetCustomer(ctx context.Context, input customerentity.GetCustomerInput) (*customerentity.Customer, error) {
	return entutils.TransactingRepo(
		ctx,
		a,
		func(ctx context.Context, repo *adapter) (*customerentity.Customer, error) {
			if err := input.Validate(); err != nil {
				return nil, customerentity.ValidationError{
					Err: fmt.Errorf("error getting customer: %w", err),
				}
			}

			query := repo.db.Customer.Query().
				WithSubjects(func(query *entdb.CustomerSubjectsQuery) {
					query.Where(customersubjectsdb.IsDeletedEQ(false))
				}).
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

			customer := CustomerFromDBEntity(*entity)

			return customer, nil
		},
	)
}

// UpdateCustomer updates a customer
func (a *adapter) UpdateCustomer(ctx context.Context, input customerentity.UpdateCustomerInput) (*customerentity.Customer, error) {
	return entutils.TransactingRepo(
		ctx,
		a,
		func(ctx context.Context, repo *adapter) (*customerentity.Customer, error) {
			if err := input.Validate(); err != nil {
				return nil, customerentity.ValidationError{
					Err: fmt.Errorf("error updating customer: %w", err),
				}
			}

			// Get the customer to diff the subjects
			dbCustomer, err := repo.GetCustomer(ctx, customerentity.GetCustomerInput(input.CustomerID))
			if err != nil {
				return nil, err
			}

			query := repo.db.Customer.UpdateOneID(dbCustomer.ID).
				Where(customerdb.Namespace(dbCustomer.Namespace)).
				SetUpdatedAt(clock.Now().UTC()).
				SetName(input.Name).
				SetOrClearDescription(input.Description).
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
						CustomerID: input.CustomerID,
					}
				}

				if entdb.IsConstraintError(err) {
					return nil, customerentity.SubjectKeyConflictError{
						Namespace:   input.CustomerID.Namespace,
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

			_, err = repo.db.CustomerSubjects.
				CreateBulk(
					lo.Map(
						subjectsKeysToAdd,
						func(subjectKey string, _ int) *entdb.CustomerSubjectsCreate {
							return repo.db.CustomerSubjects.Create().
								SetNamespace(input.CustomerID.Namespace).
								SetCustomerID(input.CustomerID.ID).
								SetSubjectKey(subjectKey)
						},
					)...,
				).
				Save(ctx)
			if err != nil {
				if entdb.IsConstraintError(err) {
					return nil, customerentity.SubjectKeyConflictError{
						Namespace:   input.CustomerID.Namespace,
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

			err = repo.db.CustomerSubjects.
				Update().
				Where(customersubjectsdb.CustomerID(input.CustomerID.ID)).
				Where(customersubjectsdb.Namespace(input.CustomerID.Namespace)).
				Where(customersubjectsdb.SubjectKeyIn(subjectKeysToRemove...)).
				Where(customersubjectsdb.IsDeletedEQ(false)).
				SetIsDeleted(true).
				SetDeletedAt(clock.Now().UTC()).
				Exec(ctx)
			if err != nil {
				if entdb.IsConstraintError(err) {
					return nil, customerentity.SubjectKeyConflictError{
						Namespace:   input.CustomerID.Namespace,
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
					Namespace:  input.CustomerID.Namespace,
					CustomerID: input.CustomerID.ID,
					SubjectKey: subjectKey,
				})
			}

			// Add the new subjects
			for _, subjectKey := range subjectsKeysToAdd {
				entity.Edges.Subjects = append(entity.Edges.Subjects, &entdb.CustomerSubjects{
					Namespace:  input.CustomerID.Namespace,
					CustomerID: input.CustomerID.ID,
					SubjectKey: subjectKey,
				})
			}

			customer := CustomerFromDBEntity(*entity)

			return customer, nil
		},
	)
}
