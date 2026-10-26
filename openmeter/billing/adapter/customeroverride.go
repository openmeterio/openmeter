package billingadapter

import (
	"context"
	"fmt"
	"slices"

	"entgo.io/ent/dialect/sql"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingcustomeroverride"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingprofile"
	dbcustomer "github.com/openmeterio/openmeter/openmeter/ent/db/customer"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

// Stay below PostgreSQL's parameter limit even while the legacy columns remain.
var defaultBulkAssignCustomersToProfileBatchSize = (65535 / len(billingcustomeroverride.Columns)) - 1

var _ billing.CustomerBillingProfileAdapter = (*adapter)(nil)

func (a *adapter) SetCustomerBillingProfile(ctx context.Context, input billing.SetCustomerBillingProfileInput) (billing.CustomerBillingProfileAdapterResult, error) {
	if err := input.Validate(); err != nil {
		return billing.CustomerBillingProfileAdapterResult{}, billing.ValidationError{Err: err}
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (billing.CustomerBillingProfileAdapterResult, error) {
		err := tx.db.BillingCustomerOverride.Create().
			SetNamespace(input.Customer.Namespace).
			SetCustomerID(input.Customer.ID).
			SetBillingProfileID(input.ProfileID).
			OnConflict(sql.ConflictColumns(billingcustomeroverride.FieldNamespace, billingcustomeroverride.FieldCustomerID)).
			UpdateBillingProfileID().
			UpdateUpdatedAt().
			ClearDeletedAt().
			Exec(ctx)
		if err != nil {
			return billing.CustomerBillingProfileAdapterResult{}, err
		}

		return tx.GetCustomerBillingProfile(ctx, billing.GetCustomerBillingProfileInput{Customer: input.Customer})
	})
}

func (a *adapter) GetCustomerBillingProfile(ctx context.Context, input billing.GetCustomerBillingProfileInput) (billing.CustomerBillingProfileAdapterResult, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (billing.CustomerBillingProfileAdapterResult, error) {
		row, err := tx.db.BillingCustomerOverride.Query().
			Where(billingcustomeroverride.Namespace(input.Customer.Namespace)).
			Where(billingcustomeroverride.CustomerID(input.Customer.ID)).
			Where(billingcustomeroverride.DeletedAtIsNil()).
			Where(billingcustomeroverride.BillingProfileIDNotNil()).
			WithBillingProfile(func(query *db.BillingProfileQuery) {
				query.Where(billingprofile.Namespace(input.Customer.Namespace))
				query.WithWorkflowConfig(workflowConfigWithTaxCode(input.Customer.Namespace))
			}).Only(ctx)
		if err != nil && !db.IsNotFound(err) {
			return billing.CustomerBillingProfileAdapterResult{}, err
		}

		selection, err := tx.mapCustomerBillingProfileFromDB(ctx, row)
		if err != nil {
			return billing.CustomerBillingProfileAdapterResult{}, err
		}

		selection.CustomerID = input.Customer

		return selection, nil
	})
}

func (a *adapter) ListCustomerBillingProfiles(ctx context.Context, input billing.ListCustomerBillingProfilesInput) (billing.ListCustomerBillingProfilesAdapterResult, error) {
	// Warning: We need to use the customer db parts as for the UI (and for a good API) we need to
	// be able to filter based on customer fields too.
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (billing.ListCustomerBillingProfilesAdapterResult, error) {
		query := tx.db.Customer.Query().
			Where(dbcustomer.NamespaceEQ(input.Namespace)).
			Where(dbcustomer.DeletedAtIsNil())

		// Customer field filters
		if len(input.CustomerIDs) > 0 {
			query = query.Where(dbcustomer.IDIn(input.CustomerIDs...))
		}

		if input.CustomerName != "" {
			query = query.Where(dbcustomer.NameContainsFold(input.CustomerName))
		}

		if input.CustomerKey != "" {
			query = query.Where(dbcustomer.KeyEQ(input.CustomerKey))
		}

		if input.CustomerPrimaryEmail != "" {
			query = query.Where(dbcustomer.PrimaryEmailContainsFold(input.CustomerPrimaryEmail))
		}

		order := entutils.GetOrdering(sortx.OrderDefault)
		if !input.Order.IsDefaultValue() {
			order = entutils.GetOrdering(input.Order)
		}

		switch input.OrderBy {
		case billing.CustomerBillingProfileOrderByCustomerID:
			query = query.Order(dbcustomer.ByID(order...))
		case billing.CustomerBillingProfileOrderByCustomerName:
			query = query.Order(dbcustomer.ByName(order...))
		case billing.CustomerBillingProfileOrderByCustomerKey:
			query = query.Order(dbcustomer.ByKey(order...))
		case billing.CustomerBillingProfileOrderByCustomerPrimaryEmail:
			query = query.Order(dbcustomer.ByPrimaryEmail(order...))
		case billing.CustomerBillingProfileOrderByCustomerCreatedAt:
			query = query.Order(dbcustomer.ByCreatedAt(order...))
		default:
			query = query.Order(dbcustomer.ByID(order...))
		}

		// A historical empty or deleted override row is not an active profile pin.
		activePin := []predicate.BillingCustomerOverride{
			billingcustomeroverride.NamespaceEQ(input.Namespace),
			billingcustomeroverride.DeletedAtIsNil(),
			billingcustomeroverride.BillingProfileIDNotNil(),
		}
		matchingPin := append([]predicate.BillingCustomerOverride{}, activePin...)
		if len(input.BillingProfiles) > 0 {
			matchingPin = append(matchingPin, billingcustomeroverride.BillingProfileIDIn(input.BillingProfiles...))
		}

		switch {
		case input.CustomersWithoutPinnedProfile:
			query = query.Where(dbcustomer.Not(dbcustomer.HasBillingCustomerOverrideWith(activePin...)))
		case !input.IncludeAllCustomers:
			query = query.Where(dbcustomer.HasBillingCustomerOverrideWith(matchingPin...))
		case len(input.BillingProfiles) > 0:
			defaultProfile, err := tx.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{Namespace: input.Namespace})
			if err != nil {
				return billing.ListCustomerBillingProfilesAdapterResult{}, err
			}

			if defaultProfile != nil && slices.Contains(input.BillingProfiles, defaultProfile.ID) {
				query = query.Where(dbcustomer.Or(
					dbcustomer.HasBillingCustomerOverrideWith(matchingPin...),
					dbcustomer.Not(dbcustomer.HasBillingCustomerOverrideWith(activePin...)),
				))
			} else {
				query = query.Where(dbcustomer.HasBillingCustomerOverrideWith(matchingPin...))
			}
		}

		query = query.WithBillingCustomerOverride(func(overrideQuery *db.BillingCustomerOverrideQuery) {
			overrideQuery = overrideQuery.Where(billingcustomeroverride.NamespaceEQ(input.Namespace)).
				Where(billingcustomeroverride.DeletedAtIsNil()).
				Where(billingcustomeroverride.BillingProfileIDNotNil())

			overrideQuery.WithBillingProfile(func(profileQuery *db.BillingProfileQuery) {
				profileQuery.Where(billingprofile.Namespace(input.Namespace))
				profileQuery.WithWorkflowConfig(workflowConfigWithTaxCode(input.Namespace))
			})
		})

		res, err := query.Paginate(ctx, input.Page)
		if err != nil {
			return billing.ListCustomerBillingProfilesAdapterResult{}, err
		}

		return pagination.MapResultErr(res, func(dbCustomer *db.Customer) (billing.CustomerBillingProfileAdapterResult, error) {
			selection, err := tx.mapCustomerBillingProfileFromDB(ctx, dbCustomer.Edges.BillingCustomerOverride)
			if err != nil {
				return billing.CustomerBillingProfileAdapterResult{}, err
			}

			selection.CustomerID = customer.CustomerID{Namespace: dbCustomer.Namespace, ID: dbCustomer.ID}

			return selection, nil
		})
	})
}

func (a *adapter) ClearCustomerBillingProfile(ctx context.Context, input billing.ClearCustomerBillingProfileInput) error {
	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		rowsAffected, err := tx.db.BillingCustomerOverride.Update().
			Where(billingcustomeroverride.CustomerID(input.Customer.ID)).
			Where(billingcustomeroverride.Namespace(input.Customer.Namespace)).
			Where(billingcustomeroverride.DeletedAtIsNil()).
			SetDeletedAt(clock.Now()).
			Save(ctx)
		if err != nil {
			if db.IsNotFound(err) {
				return billing.NotFoundError{
					ID:     input.Customer.ID,
					Entity: billing.EntityCustomerOverride,
					Err:    billing.ErrCustomerOverrideNotFound,
				}
			}

			return err
		}

		if rowsAffected == 0 {
			return billing.NotFoundError{
				ID:     input.Customer.ID,
				Entity: billing.EntityCustomerOverride,
				Err:    billing.ErrCustomerOverrideNotFound,
			}
		}

		return nil
	})
}

func (a *adapter) GetCustomersReferencingProfile(ctx context.Context, input billing.ProfileID) ([]customer.CustomerID, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]customer.CustomerID, error) {
		dbCustomerOverrides, err := tx.db.BillingCustomerOverride.Query().
			Where(billingcustomeroverride.Namespace(input.Namespace)).
			Where(billingcustomeroverride.BillingProfileID(input.ID)).
			Where(billingcustomeroverride.DeletedAtIsNil()).
			Select(billingcustomeroverride.FieldCustomerID).
			All(ctx)
		if err != nil {
			return nil, err
		}

		var customerIDs []customer.CustomerID
		for _, dbCustomerOverride := range dbCustomerOverrides {
			customerIDs = append(customerIDs, customer.CustomerID{
				Namespace: input.Namespace,
				ID:        dbCustomerOverride.CustomerID,
			})
		}

		return customerIDs, nil
	})
}

func (a *adapter) BulkAssignCustomersToProfile(ctx context.Context, input billing.BulkAssignCustomersToProfileInput) error {
	if err := input.Validate(); err != nil {
		return billing.ValidationError{Err: err}
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		profileExists, err := tx.db.BillingProfile.Query().
			Where(billingprofile.Namespace(input.ProfileID.Namespace)).
			Where(billingprofile.ID(input.ProfileID.ID)).
			Where(billingprofile.DeletedAtIsNil()).
			Exist(ctx)
		if err != nil {
			return err
		}

		if !profileExists {
			return billing.NotFoundError{Err: billing.ErrProfileNotFound}
		}

		customers := lo.Uniq(input.CustomerIDs)
		customerIDs := lo.Map(customers, func(customerID customer.CustomerID, _ int) string {
			return customerID.ID
		})
		customerCount, err := tx.db.Customer.Query().
			Where(dbcustomer.Namespace(input.ProfileID.Namespace)).
			Where(dbcustomer.IDIn(customerIDs...)).
			Where(dbcustomer.DeletedAtIsNil()).
			Count(ctx)
		if err != nil {
			return err
		}

		if customerCount != len(customerIDs) {
			return billing.NotFoundError{Err: billing.ErrCustomerNotFound}
		}

		creates := make([]*db.BillingCustomerOverrideCreate, len(customers))
		for i, customerID := range customers {
			creates[i] = tx.db.BillingCustomerOverride.Create().
				SetNamespace(input.ProfileID.Namespace).
				SetCustomerID(customerID.ID).
				SetBillingProfileID(input.ProfileID.ID)
		}

		for _, createChunk := range lo.Chunk(creates, defaultBulkAssignCustomersToProfileBatchSize) {
			err := tx.db.BillingCustomerOverride.
				CreateBulk(createChunk...).
				OnConflict(
					sql.ConflictColumns(billingcustomeroverride.FieldNamespace, billingcustomeroverride.FieldCustomerID),
				).
				UpdateBillingProfileID().
				UpdateUpdatedAt().
				ClearDeletedAt().
				Exec(ctx)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// mapCustomerBillingProfileFromDB must never substitute a default for the persisted pin.
func (a *adapter) mapCustomerBillingProfileFromDB(ctx context.Context, row *db.BillingCustomerOverride) (billing.CustomerBillingProfileAdapterResult, error) {
	if row == nil || row.DeletedAt != nil || row.BillingProfileID == nil {
		return billing.CustomerBillingProfileAdapterResult{}, nil
	}

	if row.Edges.BillingProfile == nil || row.Edges.BillingProfile.Namespace != row.Namespace {
		a.logger.ErrorContext(ctx, "billing customer profile pin namespace isolation violation",
			"customer_override_id", row.ID,
			"expected_namespace", row.Namespace,
			"billing_profile_id", *row.BillingProfileID,
		)

		return billing.CustomerBillingProfileAdapterResult{}, billing.NotFoundError{Err: fmt.Errorf("%w [id=%s]", billing.ErrProfileNotFound, *row.BillingProfileID)}
	}

	profile, err := a.mapProfileFromDB(ctx, row.Edges.BillingProfile)
	if err != nil {
		return billing.CustomerBillingProfileAdapterResult{}, fmt.Errorf("mapping pinned profile: %w", err)
	}

	return billing.CustomerBillingProfileAdapterResult{
		CustomerID: customer.CustomerID{Namespace: row.Namespace, ID: row.CustomerID},
		AssignedProfile: &billing.CustomerBillingProfileAssignment{
			ID:        row.ID,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
			ProfileID: *row.BillingProfileID,
		},
		Profile: &billing.Profile{BaseProfile: profile.BaseProfile},
	}, nil
}
