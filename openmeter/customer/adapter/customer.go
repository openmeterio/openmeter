package adapter

import (
	"context"
	"fmt"
	"slices"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/customer"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	customerdb "github.com/openmeterio/openmeter/openmeter/ent/db/customer"
	customersubjectsdb "github.com/openmeterio/openmeter/openmeter/ent/db/customersubjects"
	plandb "github.com/openmeterio/openmeter/openmeter/ent/db/plan"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	subscriptiondb "github.com/openmeterio/openmeter/openmeter/ent/db/subscription"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

// ListCustomers lists customers
func (a *adapter) ListCustomers(ctx context.Context, input customer.ListCustomersInput) (pagination.Result[customer.Customer], error) {
	if err := input.Validate(); err != nil {
		return pagination.Result[customer.Customer]{}, models.NewGenericValidationError(err)
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, repo *adapter) (pagination.Result[customer.Customer], error) {
		// Build the database query
		now := clock.Now().UTC()
		expandSubscriptions := slices.Contains(input.Expands, customer.ExpandSubscriptions)

		query := repo.db.Customer.Query().Where(customerdb.Namespace(input.Namespace))
		query = WithSubjects(query, now)

		// Expands
		if expandSubscriptions {
			query = WithActiveSubscriptions(query, now)
		}

		// Do not return deleted customers by default
		if !input.IncludeDeleted {
			query = query.Where(customerdb.Or(
				customerdb.DeletedAtIsNil(),
				customerdb.DeletedAtGTE(now),
			))
		}

		// Filters
		if input.Key != nil {
			query = query.Where(customerdb.KeyContainsFold(*input.Key))
		}

		if input.Name != nil {
			query = query.Where(customerdb.NameContainsFold(*input.Name))
		}

		if input.PrimaryEmail != nil {
			query = query.Where(customerdb.PrimaryEmailContainsFold(*input.PrimaryEmail))
		}

		if input.Subject != nil {
			query = query.Where(customerdb.HasSubjectsWith(
				customersubjectsdb.SubjectKeyContainsFold(*input.Subject),
				customersubjectsdb.Or(
					customersubjectsdb.DeletedAtIsNil(),
					customersubjectsdb.DeletedAtGTE(now),
				),
			))
		}

		if len(input.CustomerIDs) > 0 {
			query = query.Where(customerdb.IDIn(input.CustomerIDs...))
		}

		// Subscription filters
		if input.PlanID != nil || input.PlanKey != nil {
			subscriptionPredicates := activeSubscriptionFilterPredicates(now)

			// Plan ID filter
			if input.PlanID != nil {
				subscriptionPredicates = append(subscriptionPredicates, subscriptiondb.HasPlanWith(
					plandb.ID(*input.PlanID),
				))
			}

			// Plan key filter
			if input.PlanKey != nil {
				subscriptionPredicates = append(subscriptionPredicates, subscriptiondb.HasPlanWith(
					plandb.Key(*input.PlanKey),
				))
			}

			query = query.Where(
				customerdb.HasSubscriptionWith(subscriptionPredicates...),
			)
		}

		// Order
		order := entutils.GetOrdering(sortx.OrderDefault)
		if !input.Order.IsDefaultValue() {
			order = entutils.GetOrdering(input.Order)
		}

		switch input.OrderBy {
		case "id":
			query = query.Order(customerdb.ByID(order...))
		case "created_at":
			query = query.Order(customerdb.ByCreatedAt(order...))
		case "name":
			fallthrough
		default:
			query = query.Order(customerdb.ByName(order...))
		}

		// Response
		response := pagination.Result[customer.Customer]{
			Page: input.Page,
		}

		paged, err := query.Paginate(ctx, input.Page)
		if err != nil {
			return response, err
		}

		result := make([]customer.Customer, 0, len(paged.Items))
		for _, item := range paged.Items {
			if item == nil {
				a.logger.WarnContext(ctx, "invalid query result: nil customer received")
				continue
			}
			cust, err := CustomerFromDBEntity(*item, input.Expands)
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

// ListCustomerUsageAttributions lists customers usage attributions
func (a *adapter) ListCustomerUsageAttributions(ctx context.Context, input customer.ListCustomerUsageAttributionsInput) (pagination.Result[streaming.CustomerUsageAttribution], error) {
	if err := input.Validate(); err != nil {
		return pagination.Result[streaming.CustomerUsageAttribution]{}, models.NewGenericValidationError(err)
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, repo *adapter) (pagination.Result[streaming.CustomerUsageAttribution], error) {
		// Build the database query
		now := clock.Now().UTC()

		query := repo.db.Customer.Query().
			// We only need to select the fields we need for the usage attribution to optimize the query
			Select(
				customerdb.FieldID,
				customerdb.FieldKey,
			).
			Where(customerdb.Namespace(input.Namespace)).
			Order(customerdb.ByID(sql.OrderAsc()))
		query = WithSubjects(query, now)

		// Filters
		if len(input.CustomerIDs) > 0 {
			query = query.Where(customerdb.IDIn(input.CustomerIDs...))
		}

		// Do not return deleted customers by default
		if !input.IncludeDeleted {
			query = query.Where(customerdb.Or(
				customerdb.DeletedAtIsNil(),
				customerdb.DeletedAtGTE(now),
			))
		}

		// Response
		response := pagination.Result[streaming.CustomerUsageAttribution]{
			Page: input.Page,
		}

		paged, err := query.Paginate(ctx, input.Page)
		if err != nil {
			return response, err
		}

		result := make([]streaming.CustomerUsageAttribution, 0, len(paged.Items))
		for _, item := range paged.Items {
			if item == nil {
				a.logger.WarnContext(ctx, "invalid query result: nil customer received")
				continue
			}

			subjectKeys, err := subjectKeysFromDBEntity(*item)
			if err != nil {
				return response, err
			}

			var usageAttribution streaming.CustomerUsageAttribution

			if item.Key == "" {
				usageAttribution = streaming.NewCustomerUsageAttribution(item.ID, nil, subjectKeys)
			} else {
				usageAttribution = streaming.NewCustomerUsageAttribution(item.ID, &item.Key, subjectKeys)
			}

			result = append(result, usageAttribution)
		}

		response.TotalCount = paged.TotalCount
		response.Items = result

		return response, nil
	})
}

// CreateCustomer creates a new customer
func (a *adapter) CreateCustomer(ctx context.Context, input customer.CreateCustomerInput) (*customer.Customer, error) {
	if err := input.Validate(); err != nil {
		return nil, models.NewGenericValidationError(
			fmt.Errorf("error creating customer: %w", err),
		)
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, repo *adapter) (*customer.Customer, error) {
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

		if customerEntity == nil {
			return nil, fmt.Errorf("invalid query result: nil customer received")
		}

		// Create customer subjects
		// TODO: customer.AddSubjects produces an invalid database query so we create it separately in a transaction.
		// The number and shape of the queries executed is the same, it's a devex thing only.
		if input.UsageAttribution != nil && len(input.UsageAttribution.SubjectKeys) > 0 {
			_, err = repo.db.CustomerSubjects.
				CreateBulk(
					lo.Map(
						input.UsageAttribution.SubjectKeys,
						func(subjectKey string, _ int) *entdb.CustomerSubjectsCreate {
							return repo.db.CustomerSubjects.Create().
								SetNamespace(customerEntity.Namespace).
								SetCustomerID(customerEntity.ID).
								SetSubjectKey(subjectKey).
								SetCreatedAt(customerEntity.CreatedAt)
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
		}

		return repo.GetCustomer(ctx, customer.GetCustomerInput{
			CustomerID: &customer.CustomerID{
				ID:        customerEntity.ID,
				Namespace: customerEntity.Namespace,
			},
		})
	})
}

// DeleteCustomer deletes a customer
func (a *adapter) DeleteCustomer(ctx context.Context, input customer.DeleteCustomerInput) error {
	if err := input.Validate(); err != nil {
		return models.NewGenericValidationError(
			fmt.Errorf("error deleting customer: %w", err),
		)
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, repo *adapter) error {
		deletedAt := clock.Now().UTC()

		// Soft delete the customer
		rows, err := repo.db.Customer.Update().
			Where(customerdb.ID(input.ID)).
			Where(customerdb.Namespace(input.Namespace)).
			Where(customerdb.DeletedAtIsNil()).
			SetDeletedAt(deletedAt).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("failed to delete customer: %w", err)
		}

		if rows == 0 {
			return models.NewGenericNotFoundError(
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
			return fmt.Errorf("failed to delete customer subjects: %w", err)
		}

		return nil
	})
}

// GetCustomer gets a customer
func (a *adapter) GetCustomer(ctx context.Context, input customer.GetCustomerInput) (*customer.Customer, error) {
	if err := input.Validate(); err != nil {
		return nil, models.NewGenericValidationError(
			fmt.Errorf("error getting customer: %w", err),
		)
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, repo *adapter) (*customer.Customer, error) {
		now := clock.Now().UTC()

		query := repo.db.Customer.Query()
		query = WithSubjects(query, now)
		if slices.Contains(input.Expands, customer.ExpandSubscriptions) {
			query = WithActiveSubscriptions(query, now)
		}

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
				customerdb.And(
					customerdb.Key(input.CustomerIDOrKey.IDOrKey),
					customerdb.DeletedAtIsNil(),
				),
			))
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

		return CustomerFromDBEntity(*entity, input.Expands)
	})
}

// GetCustomerByUsageAttribution gets a customer by usage attribution
func (a *adapter) GetCustomerByUsageAttribution(ctx context.Context, input customer.GetCustomerByUsageAttributionInput) (*customer.Customer, error) {
	if err := input.Validate(); err != nil {
		return nil, models.NewGenericValidationError(
			fmt.Errorf("error getting customer by usage attribution: %w", err),
		)
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, repo *adapter) (*customer.Customer, error) {
		now := clock.Now().UTC()

		query := repo.db.Customer.Query().
			Where(customerdb.Namespace(input.Namespace)).
			Where(
				customerdb.Or(
					// We lookup the customer by subject key in the subjects table
					customerdb.HasSubjectsWith(
						customersubjectsdb.SubjectKey(input.Key),
						customersubjectsdb.Or(
							customersubjectsdb.DeletedAtIsNil(),
							customersubjectsdb.DeletedAtGT(now),
						),
					),
					// Or else we lookup the customer by key in the customers table
					customerdb.Key(input.Key),
				),
			).
			Where(customerdb.DeletedAtIsNil())
		query = WithSubjects(query, now)
		if slices.Contains(input.Expands, customer.ExpandSubscriptions) {
			query = WithActiveSubscriptions(query, now)
		}

		customerEntity, err := query.First(ctx)
		if err != nil {
			if entdb.IsNotFound(err) {
				return nil, models.NewGenericNotFoundError(
					fmt.Errorf("customer with subject key %s not found in %s namespace", input.Key, input.Namespace),
				)
			}

			return nil, fmt.Errorf("failed to fetch customer: %w", err)
		}

		if customerEntity == nil {
			return nil, fmt.Errorf("invalid query result: nil customer received")
		}

		return CustomerFromDBEntity(*customerEntity, input.Expands)
	})
}

// UpdateCustomer updates a customer
func (a *adapter) UpdateCustomer(ctx context.Context, input customer.UpdateCustomerInput) (*customer.Customer, error) {
	if err := input.Validate(); err != nil {
		return nil, models.NewGenericValidationError(
			fmt.Errorf("error updating customer: %w", err),
		)
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, repo *adapter) (*customer.Customer, error) {
		// Get the customer to diff the subjects
		previousCustomer, err := repo.GetCustomer(ctx, customer.GetCustomerInput{
			CustomerID: &input.CustomerID,
		})
		if err != nil {
			return nil, err
		}

		if previousCustomer != nil && previousCustomer.DeletedAt != nil {
			return nil, models.NewGenericValidationError(
				fmt.Errorf("cannot updated already deleted customer [namespace=%s customer.id=%s]", input.CustomerID.Namespace, input.CustomerID.ID),
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

		if entity == nil {
			return nil, fmt.Errorf("invalid query result: nil customer received")
		}

		var previousSubjectKeys, newSubjectKeys []string
		if previousCustomer.UsageAttribution != nil {
			previousSubjectKeys = previousCustomer.UsageAttribution.SubjectKeys
		}
		if input.UsageAttribution != nil {
			newSubjectKeys = input.UsageAttribution.SubjectKeys
		}

		subKeysToRemove, subKeysToAdd := lo.Difference(
			lo.Uniq(previousSubjectKeys),
			lo.Uniq(newSubjectKeys),
		)

		now := clock.Now().UTC()

		// Add subjects
		if len(subKeysToAdd) > 0 {
			_, err = repo.db.CustomerSubjects.
				CreateBulk(
					lo.Map(
						subKeysToAdd,
						func(subjectKey string, _ int) *entdb.CustomerSubjectsCreate {
							return repo.db.CustomerSubjects.Create().
								SetNamespace(input.CustomerID.Namespace).
								SetCustomerID(input.CustomerID.ID).
								SetSubjectKey(subjectKey).
								SetCreatedAt(now)
						},
					)...,
				).
				Save(ctx)
			if err != nil {
				if entdb.IsConstraintError(err) {
					return nil, customer.NewSubjectKeyConflictError(
						input.CustomerID.Namespace,
						subKeysToAdd,
					)
				}

				return nil, fmt.Errorf("failed to add customer subjects: %w", err)
			}
		}

		// Remove subjects
		if len(subKeysToRemove) > 0 {
			err = repo.db.CustomerSubjects.
				Update().
				Where(customersubjectsdb.CustomerID(input.CustomerID.ID)).
				Where(customersubjectsdb.Namespace(input.CustomerID.Namespace)).
				Where(customersubjectsdb.SubjectKeyIn(subKeysToRemove...)).
				Where(customersubjectsdb.DeletedAtIsNil()).
				SetDeletedAt(now).
				Exec(ctx)
			if err != nil {
				if entdb.IsConstraintError(err) {
					return nil, customer.NewSubjectKeyConflictError(
						input.CustomerID.Namespace,
						subKeysToRemove,
					)
				}

				return nil, fmt.Errorf("failed to remove customer subjects: %w", err)
			}
		}

		return repo.GetCustomer(ctx, customer.GetCustomerInput{
			CustomerID: &input.CustomerID,
		})
	})
}

// WithSubjects returns a query with the subjects
func WithSubjects(q *entdb.CustomerQuery, at time.Time) *entdb.CustomerQuery {
	return q.WithSubjects(func(query *entdb.CustomerSubjectsQuery) {
		query.Where(func(s *sql.Selector) {
			ct := sql.Table(customerdb.Table)

			s.Join(ct).On(ct.C(customerdb.FieldID), s.C(customersubjectsdb.FieldCustomerID))

			s.Where(
				sql.Or(
					sql.And(
						sql.NotNull(ct.C(customerdb.FieldDeletedAt)),
						sql.ColumnsEQ(s.C(customersubjectsdb.FieldDeletedAt), ct.C(customerdb.FieldDeletedAt)),
					),
					sql.And(
						sql.IsNull(ct.C(customerdb.FieldDeletedAt)),
						sql.Or(
							sql.IsNull(s.C(customersubjectsdb.FieldDeletedAt)),
							sql.GTE(s.C(customersubjectsdb.FieldDeletedAt), at),
						),
					),
				),
			)
		})
	})
}

// WithActiveSubscriptions returns a query with the subscription
func WithActiveSubscriptions(query *entdb.CustomerQuery, at time.Time) *entdb.CustomerQuery {
	return query.WithSubscription(func(query *entdb.SubscriptionQuery) {
		query.Where(activeSubscriptionFilterPredicates(at)...)
		query.WithPlan()
	})
}

// activeSubscriptionFilterPredicates returns the active subscription predicates
func activeSubscriptionFilterPredicates(at time.Time) []predicate.Subscription {
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
