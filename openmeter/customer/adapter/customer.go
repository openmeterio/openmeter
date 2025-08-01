package adapter

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/customer"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	customerdb "github.com/openmeterio/openmeter/openmeter/ent/db/customer"
	customersubjectsdb "github.com/openmeterio/openmeter/openmeter/ent/db/customersubjects"
	plandb "github.com/openmeterio/openmeter/openmeter/ent/db/plan"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	subscriptiondb "github.com/openmeterio/openmeter/openmeter/ent/db/subscription"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

// ListCustomers lists customers
func (a *adapter) ListCustomers(ctx context.Context, input customer.ListCustomersInput) (pagination.PagedResponse[customer.Customer], error) {
	return entutils.TransactingRepo(
		ctx,
		a,
		func(ctx context.Context, repo *adapter) (pagination.PagedResponse[customer.Customer], error) {
			if err := input.Validate(); err != nil {
				return pagination.PagedResponse[customer.Customer]{}, models.NewGenericValidationError(err)
			}

			// Build the database query
			now := clock.Now().UTC()

			query := repo.db.Customer.Query().Where(customerdb.Namespace(input.Namespace))
			query = withSubjects(query)
			query = withSubscription(query, now)

			// Do not return deleted customers by default
			if !input.IncludeDeleted {
				query = query.Where(customerdb.DeletedAtIsNil())
			}

			// Filters
			if input.Key != nil {
				query = query.Where(customerdb.KeyEQ(*input.Key))
			}

			if input.Name != nil {
				query = query.Where(customerdb.NameContainsFold(*input.Name))
			}

			if input.PrimaryEmail != nil {
				query = query.Where(customerdb.PrimaryEmailContainsFold(*input.PrimaryEmail))
			}

			if input.Subject != nil {
				query = query.Where(customerdb.HasSubjectsWith(customersubjectsdb.SubjectKeyContainsFold(*input.Subject)))
			}

			if input.PlanKey != nil {
				applyActiveSubscriptionFilterWithPlanKey(query, now, *input.PlanKey)
			}

			if len(input.CustomerIDs) > 0 {
				query = query.Where(customerdb.IDIn(input.CustomerIDs...))
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
			response := pagination.PagedResponse[customer.Customer]{
				Page: input.Page,
			}

			paged, err := query.Paginate(ctx, input.Page)
			if err != nil {
				return response, err
			}

			result := make([]customer.Customer, 0, len(paged.Items))
			for _, item := range paged.Items {
				if item == nil {
					a.logger.Warn("invalid query result: nil customer received")
					continue
				}
				cust, err := CustomerFromDBEntity(*item)
				if err != nil {
					return response, fmt.Errorf("failed to convert customer: %w", err)
				}
				if cust == nil {
					return response, fmt.Errorf("invalid query result: nil customer received")
				}

				result = append(result, *cust)
			}

			response.TotalCount = paged.TotalCount
			response.Items = result

			return response, nil
		})
}

// CreateCustomer creates a new customer
func (a *adapter) CreateCustomer(ctx context.Context, input customer.CreateCustomerInput) (*customer.Customer, error) {
	return transaction.Run(ctx, a, func(ctx context.Context) (*customer.Customer, error) {
		return entutils.TransactingRepo(
			ctx,
			a,
			func(ctx context.Context, repo *adapter) (*customer.Customer, error) {
				if err := input.Validate(); err != nil {
					return nil, models.NewGenericValidationError(
						fmt.Errorf("error creating customer: %w", err),
					)
				}

				// Check if the key is not an ID of another customer
				if input.Key != nil {
					count, err := repo.db.Customer.Query().
						Where(customerdb.ID(*input.Key)).
						Where(customerdb.Namespace(input.Namespace)).
						Count(ctx)
					if err != nil {
						return nil, fmt.Errorf("failed to check if key overlaps with id: %w", err)
					}

					if count > 0 {
						return nil, models.NewGenericConflictError(
							fmt.Errorf("key %s overlaps with id of another customer", *input.Key),
						)
					}
				}

				// Check if the key is not a subject of another customer
				if input.Key != nil {
					conflictingCustomerIDs, err := repo.db.CustomerSubjects.Query().
						Select(customersubjectsdb.FieldCustomerID).
						Where(customersubjectsdb.Namespace(input.Namespace)).
						Where(customersubjectsdb.SubjectKey(*input.Key)).
						Where(customersubjectsdb.DeletedAtIsNil()).
						All(ctx)
					if err != nil {
						return nil, fmt.Errorf("failed to check if key overlaps with subject: %w", err)
					}

					if len(conflictingCustomerIDs) > 0 {
						return nil, models.NewGenericConflictError(
							fmt.Errorf("key %s overlaps with subject of another customer: %s", *input.Key, conflictingCustomerIDs[0].CustomerID),
						)
					}
				}

				// Create the customer in the database
				query := repo.db.Customer.Create().
					SetNamespace(input.Namespace).
					SetName(input.Name).
					SetNillableDescription(input.Description).
					SetNillablePrimaryEmail(input.PrimaryEmail).
					SetNillableCurrency(input.Currency)

				if input.Key != nil {
					query = query.SetKey(*input.Key)
				}

				if input.Metadata != nil {
					query = query.SetMetadata(input.Metadata.ToMap())
				}

				if input.Annotation != nil {
					query = query.SetAnnotations(*input.Annotation)
				}

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
					if entdb.IsConstraintError(err) {
						return nil, customer.NewKeyConflictError(
							input.Namespace,
							*lo.CoalesceOrEmpty(input.Key),
						)
					}

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
						return nil, customer.NewSubjectKeyConflictError(
							input.Namespace,
							input.UsageAttribution.SubjectKeys,
						)
					}

					return nil, fmt.Errorf("failed to create customer: failed to add subject keys: %w", err)
				}

				if customerEntity == nil {
					return nil, fmt.Errorf("invalid query result: nil customer received")
				}

				// When creating a customer it's not possible for it to have a subscription,
				// so we don't need to fetch it here.

				customerEntity.Edges.Subjects = customerSubjects
				cus, err := CustomerFromDBEntity(*customerEntity)
				if err != nil {
					return cus, fmt.Errorf("failed to convert customer: %w", err)
				}
				if cus == nil {
					return cus, fmt.Errorf("invalid query result: nil customer received")
				}

				return cus, nil
			},
		)
	})
}

// DeleteCustomer deletes a customer
func (a *adapter) DeleteCustomer(ctx context.Context, input customer.DeleteCustomerInput) error {
	return transaction.RunWithNoValue(ctx, a, func(ctx context.Context) error {
		_, err := entutils.TransactingRepo(
			ctx,
			a,
			func(ctx context.Context, repo *adapter) (any, error) {
				if err := input.Validate(); err != nil {
					return nil, models.NewGenericValidationError(
						fmt.Errorf("error deleting customer: %w", err),
					)
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
					return nil, models.NewGenericNotFoundError(
						fmt.Errorf("customer with id %s not found in %s namespace", input.ID, input.Namespace),
					)
				}

				// Soft delete the customer subjects
				err = repo.db.CustomerSubjects.
					Update().
					Where(customersubjectsdb.CustomerID(input.ID)).
					Where(customersubjectsdb.Namespace(input.Namespace)).
					Where(customersubjectsdb.DeletedAtIsNil()).
					SetDeletedAt(deletedAt).
					Exec(ctx)
				if err != nil {
					return nil, fmt.Errorf("failed to delete customer subjects: %w", err)
				}

				return nil, nil
			},
		)

		return err
	})
}

// GetCustomer gets a customer
func (a *adapter) GetCustomer(ctx context.Context, input customer.GetCustomerInput) (*customer.Customer, error) {
	return entutils.TransactingRepo(
		ctx,
		a,
		func(ctx context.Context, repo *adapter) (*customer.Customer, error) {
			if err := input.Validate(); err != nil {
				return nil, models.NewGenericValidationError(
					fmt.Errorf("error getting customer: %w", err),
				)
			}

			query := repo.db.Customer.Query()
			query = withSubjects(query)
			query = withActiveSubscription(query)

			if input.CustomerID != nil {
				query = query.Where(customerdb.Namespace(input.CustomerID.Namespace))
				query = query.Where(customerdb.ID(input.CustomerID.ID))
			} else if input.CustomerKey != nil {
				query = query.Where(customerdb.Namespace(input.CustomerKey.Namespace))
				query = query.Where(customerdb.Key(input.CustomerKey.Key))
				query = query.Where(customerdb.DeletedAtIsNil())
			} else if input.CustomerIDOrKey != nil {
				query = query.Where(customerdb.Namespace(input.CustomerIDOrKey.Namespace))
				query = query.Where(customerdb.Or(
					customerdb.ID(input.CustomerIDOrKey.IDOrKey),
					customerdb.Key(input.CustomerIDOrKey.IDOrKey),
				))
				query = query.Where(customerdb.DeletedAtIsNil())
				query = query.Order(customerdb.ByID(sql.OrderAsc()))
			} else {
				return nil, models.NewGenericValidationError(
					fmt.Errorf("customer id or key is required"),
				)
			}

			entity, err := query.First(ctx)
			if err != nil {
				if entdb.IsNotFound(err) {
					if input.CustomerID != nil {
						return nil, models.NewGenericNotFoundError(
							fmt.Errorf("customer with id %s not found in %s namespace", input.CustomerID.ID, input.CustomerID.Namespace),
						)
					} else if input.CustomerKey != nil {
						return nil, models.NewGenericNotFoundError(
							fmt.Errorf("customer with key %s not found in %s namespace", input.CustomerKey.Key, input.CustomerKey.Namespace),
						)
					} else if input.CustomerIDOrKey != nil {
						return nil, models.NewGenericNotFoundError(
							fmt.Errorf("customer with id or key %s not found in %s namespace", input.CustomerIDOrKey.IDOrKey, input.CustomerIDOrKey.Namespace),
						)
					}
				}

				return nil, fmt.Errorf("failed to fetch customer: %w", err)
			}

			if entity == nil {
				return nil, fmt.Errorf("invalid query result: nil customer received")
			}

			return CustomerFromDBEntity(*entity)
		},
	)
}

// GetCustomerByUsageAttribution gets a customer by usage attribution
func (a *adapter) GetCustomerByUsageAttribution(ctx context.Context, input customer.GetCustomerByUsageAttributionInput) (*customer.Customer, error) {
	if err := input.Validate(); err != nil {
		return nil, models.NewGenericValidationError(
			fmt.Errorf("error getting customer by usage attribution: %w", err),
		)
	}

	query := a.db.Customer.Query().
		Where(customerdb.Namespace(input.Namespace)).
		Where(customerdb.HasSubjectsWith(customersubjectsdb.SubjectKey(input.SubjectKey))).
		Where(customerdb.DeletedAtIsNil())
	query = withSubjects(query)
	query = withActiveSubscription(query)

	customerEntity, err := query.First(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, models.NewGenericNotFoundError(
				fmt.Errorf("customer with subject key %s not found in %s namespace", input.SubjectKey, input.Namespace),
			)
		}

		return nil, fmt.Errorf("failed to fetch customer: %w", err)
	}

	if customerEntity == nil {
		return nil, fmt.Errorf("invalid query result: nil customer received")
	}

	return CustomerFromDBEntity(*customerEntity)
}

// UpdateCustomer updates a customer
func (a *adapter) UpdateCustomer(ctx context.Context, input customer.UpdateCustomerInput) (*customer.Customer, error) {
	return transaction.Run(ctx, a, func(ctx context.Context) (*customer.Customer, error) {
		return entutils.TransactingRepo(
			ctx,
			a,
			func(ctx context.Context, repo *adapter) (*customer.Customer, error) {
				if err := input.Validate(); err != nil {
					return nil, models.NewGenericValidationError(
						fmt.Errorf("error updating customer: %w", err),
					)
				}

				// Check if the key is not an ID of another customer
				if input.Key != nil {
					count, err := repo.db.Customer.Query().
						Where(customerdb.ID(*input.Key)).
						Where(customerdb.Namespace(input.CustomerID.Namespace)).
						Count(ctx)
					if err != nil {
						return nil, fmt.Errorf("failed to check if key overlaps with id: %w", err)
					}

					if count > 0 {
						return nil, models.NewGenericConflictError(
							fmt.Errorf("key %s overlaps with id of another customer", *input.Key),
						)
					}
				}

				// Check if the key is not a subject of another customer
				if input.Key != nil {
					conflictingCustomerIDs, err := repo.db.CustomerSubjects.Query().
						Select(customersubjectsdb.FieldCustomerID).
						Where(customersubjectsdb.Namespace(input.CustomerID.Namespace)).
						Where(customersubjectsdb.CustomerIDNEQ(input.CustomerID.ID)).
						Where(customersubjectsdb.SubjectKey(*input.Key)).
						Where(customersubjectsdb.DeletedAtIsNil()).
						All(ctx)
					if err != nil {
						return nil, fmt.Errorf("failed to check if key overlaps with subject: %w", err)
					}

					if len(conflictingCustomerIDs) > 0 {
						return nil, models.NewGenericConflictError(
							fmt.Errorf("key %s overlaps with subject of another customer: %s", *input.Key, conflictingCustomerIDs[0].CustomerID),
						)
					}
				}

				// Get the customer to diff the subjects
				previousCustomer, err := repo.GetCustomer(ctx, customer.GetCustomerInput{
					CustomerID: &input.CustomerID,
				})
				if err != nil {
					return nil, err
				}

				query := repo.db.Customer.UpdateOneID(previousCustomer.ID).
					Where(customerdb.Namespace(previousCustomer.Namespace)).
					SetUpdatedAt(clock.Now().UTC()).
					SetName(input.Name).
					SetOrClearDescription(input.Description).
					SetNillablePrimaryEmail(input.PrimaryEmail).
					SetNillableCurrency(input.Currency).
					SetOrClearKey(input.Key)

				// Replace metadata
				if input.Metadata != nil {
					query = query.SetMetadata(input.Metadata.ToMap())
				} else {
					query = query.ClearMetadata()
				}

				if input.Annotation != nil {
					query = query.SetAnnotations(*input.Annotation)
				} else {
					query = query.ClearAnnotations()
				}

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
						return nil, models.NewGenericNotFoundError(
							fmt.Errorf("customer with id %s not found in %s namespace", input.CustomerID.ID, input.CustomerID.Namespace),
						)
					}

					if entdb.IsConstraintError(err) {
						return nil, customer.NewKeyConflictError(
							input.CustomerID.Namespace,
							*lo.CoalesceOrEmpty(input.Key),
						)
					}

					return nil, fmt.Errorf("failed to update customer: %w", err)
				}

				// Add new subjects
				var subjectsKeysToAdd []string

				for _, subjectKey := range input.UsageAttribution.SubjectKeys {
					found := false

					for _, existingSubjectKey := range previousCustomer.UsageAttribution.SubjectKeys {
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
						return nil, customer.NewSubjectKeyConflictError(
							input.CustomerID.Namespace,
							subjectsKeysToAdd,
						)
					}

					return nil, fmt.Errorf("failed to add customer subjects: %w", err)
				}

				// Remove subjects
				var subjectKeysToRemove []string

				for _, existingSubjectKey := range previousCustomer.UsageAttribution.SubjectKeys {
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
					Where(customersubjectsdb.DeletedAtIsNil()).
					SetDeletedAt(clock.Now().UTC()).
					Exec(ctx)
				if err != nil {
					if entdb.IsConstraintError(err) {
						return nil, customer.NewSubjectKeyConflictError(
							input.CustomerID.Namespace,
							subjectKeysToRemove,
						)
					}

					return nil, fmt.Errorf("failed to remove customer subjects: %w", err)
				}

				if entity == nil {
					return nil, fmt.Errorf("invalid query result: nil customer received")
				}

				// Let's fetch the Subscription if present
				subsQuery := repo.db.Subscription.Query()
				applyActiveSubscriptionFilter(subsQuery, clock.Now().UTC())
				subsEnt, err := subsQuery.
					WithPlan().
					Where(subscriptiondb.CustomerID(entity.ID)).
					Only(ctx)
				if err == nil && subsEnt != nil {
					entity.Edges.Subscription = []*entdb.Subscription{subsEnt}
				} else if !entdb.IsNotFound(err) {
					return nil, fmt.Errorf("failed to fetch customer subscription: %w", err)
				}

				// Final subject keys
				entity.Edges.Subjects = []*entdb.CustomerSubjects{}

				// Loop through the existing subjects and add the ones that are not removed
				for _, subjectKey := range previousCustomer.UsageAttribution.SubjectKeys {
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

				cus, err := CustomerFromDBEntity(*entity)
				if err != nil {
					return cus, fmt.Errorf("failed to convert customer: %w", err)
				}

				if cus == nil {
					return cus, fmt.Errorf("invalid query result: nil customer received")
				}

				return cus, nil
			},
		)
	})
}

// withSubjects returns a query with the subjects
func withSubjects(query *entdb.CustomerQuery) *entdb.CustomerQuery {
	return query.WithSubjects(func(query *entdb.CustomerSubjectsQuery) {
		query.Where(customersubjectsdb.DeletedAtIsNil())
	})
}

// withActiveSubscription returns a query with the active subscription
func withActiveSubscription(query *entdb.CustomerQuery) *entdb.CustomerQuery {
	now := clock.Now().UTC()

	return withSubscription(query, now)
}

// withSubscription returns a query with the subscription
func withSubscription(query *entdb.CustomerQuery, at time.Time) *entdb.CustomerQuery {
	return query.WithSubscription(func(query *entdb.SubscriptionQuery) {
		applyActiveSubscriptionFilter(query, at)
		query.WithPlan()
	})
}

func (a *adapter) CustomerExists(ctx context.Context, customerID customer.CustomerID) error {
	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, repo *adapter) error {
		count, err := repo.db.Customer.Query().
			Where(customerdb.Namespace(customerID.Namespace)).
			Where(customerdb.ID(customerID.ID)).
			Where(customerdb.DeletedAtIsNil()).
			Count(ctx)
		if err != nil {
			return err
		}

		if count == 0 {
			return models.NewGenericNotFoundError(
				fmt.Errorf("customer with id %s not found in %s namespace", customerID.ID, customerID.Namespace),
			)
		}

		return nil
	})
}

func applyActiveSubscriptionFilter(query *entdb.SubscriptionQuery, at time.Time) {
	query.Where(activeSubscriptionFilter(at)...)
}

func applyActiveSubscriptionFilterWithPlanKey(query *entdb.CustomerQuery, at time.Time, planKey string) {
	predicates := activeSubscriptionFilter(at)

	predicates = append(predicates, subscriptiondb.HasPlanWith(
		plandb.Key(planKey),
	))

	query.Where(
		customerdb.HasSubscriptionWith(predicates...),
	)
}

func activeSubscriptionFilter(at time.Time) []predicate.Subscription {
	return []predicate.Subscription{
		subscriptiondb.ActiveFromLTE(at),
		subscriptiondb.Or(
			subscriptiondb.ActiveToIsNil(),
			subscriptiondb.ActiveToGT(at),
		),
		subscriptiondb.Or(
			subscriptiondb.DeletedAtIsNil(),
			subscriptiondb.DeletedAtGT(at),
		),
		subscriptiondb.CreatedAtLTE(at),
	}
}
