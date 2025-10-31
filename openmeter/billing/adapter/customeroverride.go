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

// defaultBulkAssignCustomersToProfileBatchSize is the maximum number of customers that can be assigned to a profile in a single
// upsert operation. This is based on the maximum number of parameters PostgreSQL can handle in a single upsert operation (64k).
//
// This is a pessimistic approximation, as entgo might not try to insert the null columns, but still as of the writing we are still
// inserting in 4k batches, which is more than enough.
var defaultBulkAssignCustomersToProfileBatchSize int = (65535 / len(billingcustomeroverride.Columns)) - 1

var _ billing.CustomerOverrideAdapter = (*adapter)(nil)

func (a *adapter) CreateCustomerOverride(ctx context.Context, input billing.CreateCustomerOverrideAdapterInput) (*billing.CustomerOverride, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (*billing.CustomerOverride, error) {
		_, err := tx.db.BillingCustomerOverride.Create().
			SetNamespace(input.Namespace).
			SetCustomerID(input.CustomerID).
			SetNillableBillingProfileID(lo.EmptyableToPtr(input.ProfileID)).
			SetNillableCollectionAlignment(input.Collection.Alignment).
			SetAnchoredAlignmentDetail(input.Collection.AnchoredAlignmentDetail).
			SetNillableLineCollectionPeriod(input.Collection.Interval.ISOStringPtrOrNil()).
			SetNillableInvoiceAutoAdvance(input.Invoicing.AutoAdvance).
			SetNillableInvoiceDraftPeriod(input.Invoicing.DraftPeriod.ISOStringPtrOrNil()).
			SetNillableInvoiceDueAfter(input.Invoicing.DueAfter.ISOStringPtrOrNil()).
			SetNillableInvoiceCollectionMethod(input.Payment.CollectionMethod).
			SetNillableInvoiceProgressiveBilling(input.Invoicing.ProgressiveBilling).
			SetNillableInvoiceDefaultTaxConfig(input.Invoicing.DefaultTaxConfig).
			Save(ctx)
		if err != nil {
			return nil, err
		}

		// Let's fetch the override with edges
		return tx.GetCustomerOverride(ctx, billing.GetCustomerOverrideAdapterInput{
			Customer: customer.CustomerID{
				Namespace: input.Namespace,
				ID:        input.CustomerID,
			},
		})
	})
}

func (a *adapter) UpdateCustomerOverride(ctx context.Context, input billing.UpdateCustomerOverrideAdapterInput) (*billing.CustomerOverride, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (*billing.CustomerOverride, error) {
		if input.ProfileID == "" {
			// Let's resolve the default profile
			defaultProfile, err := tx.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{
				Namespace: input.Namespace,
			})
			if err != nil {
				return nil, billing.NotFoundError{
					Entity: billing.EntityDefaultProfile,
					Err:    billing.ErrDefaultProfileNotFound,
				}
			}

			input.ProfileID = defaultProfile.ID
		}

		update := tx.db.BillingCustomerOverride.Update().
			Where(billingcustomeroverride.CustomerID(input.CustomerID)).
			SetOrClearBillingProfileID(lo.EmptyableToPtr(input.ProfileID)).
			SetOrClearCollectionAlignment(input.Collection.Alignment).
			SetOrClearLineCollectionPeriod(input.Collection.Interval.ISOStringPtrOrNil()).
			SetOrClearInvoiceAutoAdvance(input.Invoicing.AutoAdvance).
			SetOrClearInvoiceDraftPeriod(input.Invoicing.DraftPeriod.ISOStringPtrOrNil()).
			SetOrClearInvoiceDueAfter(input.Invoicing.DueAfter.ISOStringPtrOrNil()).
			SetOrClearInvoiceCollectionMethod(input.Payment.CollectionMethod).
			SetOrClearInvoiceProgressiveBilling(input.Invoicing.ProgressiveBilling).
			SetOrClearInvoiceDefaultTaxConfig(input.Invoicing.DefaultTaxConfig).
			ClearDeletedAt()

		linesAffected, err := update.Save(ctx)
		if err != nil {
			return nil, err
		}

		if linesAffected == 0 {
			return nil, billing.NotFoundError{
				ID:     input.CustomerID,
				Entity: billing.EntityCustomerOverride,
				Err:    billing.ErrCustomerOverrideNotFound,
			}
		}

		return tx.GetCustomerOverride(ctx, billing.GetCustomerOverrideAdapterInput{
			Customer: customer.CustomerID{
				Namespace: input.Namespace,
				ID:        input.CustomerID,
			},
		})
	})
}

func (a *adapter) GetCustomerOverride(ctx context.Context, input billing.GetCustomerOverrideAdapterInput) (*billing.CustomerOverride, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (*billing.CustomerOverride, error) {
		query := tx.db.BillingCustomerOverride.Query().
			Where(billingcustomeroverride.Namespace(input.Customer.Namespace)).
			Where(billingcustomeroverride.CustomerID(input.Customer.ID)).
			WithBillingProfile(func(bpq *db.BillingProfileQuery) {
				bpq.WithWorkflowConfig()
			})

		if !input.IncludeDeleted {
			query = query.Where(billingcustomeroverride.DeletedAtIsNil())
		}

		dbCustomerOverride, err := query.First(ctx)
		if err != nil {
			if db.IsNotFound(err) {
				return nil, nil
			}

			return nil, err
		}

		if dbCustomerOverride.BillingProfileID == nil {
			// Let's fetch the default billing profile
			dbDefaultProfile, err := tx.db.BillingProfile.Query().
				Where(billingprofile.Namespace(input.Customer.Namespace)).
				Where(billingprofile.Default(true)).
				Where(billingprofile.DeletedAtIsNil()).
				WithWorkflowConfig().
				Only(ctx)
			if err != nil {
				if !db.IsNotFound(err) {
					return nil, err
				}
			}

			dbCustomerOverride.Edges.BillingProfile = dbDefaultProfile
		}

		return mapCustomerOverrideFromDB(dbCustomerOverride)
	})
}

func (a *adapter) ListCustomerOverrides(ctx context.Context, input billing.ListCustomerOverridesInput) (billing.ListCustomerOverridesAdapterResult, error) {
	// Warning: We need to use the customer db parts as for the UI (and for a good API) we need to
	// be able to filter based on customer fields too.
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (billing.ListCustomerOverridesAdapterResult, error) {
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
		case billing.CustomerOverrideOrderByCustomerID:
			query = query.Order(dbcustomer.ByID(order...))
		case billing.CustomerOverrideOrderByCustomerName:
			query = query.Order(dbcustomer.ByName(order...))
		case billing.CustomerOverrideOrderByCustomerKey:
			query = query.Order(dbcustomer.ByKey(order...))
		case billing.CustomerOverrideOrderByCustomerPrimaryEmail:
			query = query.Order(dbcustomer.ByPrimaryEmail(order...))
		case billing.CustomerOverrideOrderByCustomerCreatedAt:
			query = query.Order(dbcustomer.ByCreatedAt(order...))
		default:
			query = query.Order(dbcustomer.ByID(order...))
		}

		// Customer override filtering
		customerOverrideFilters := []predicate.BillingCustomerOverride{
			billingcustomeroverride.DeletedAtIsNil(),
			billingcustomeroverride.NamespaceEQ(input.Namespace),
		}

		if len(input.BillingProfiles) > 0 {
			customerOverrideFilters = append(customerOverrideFilters, billingcustomeroverride.BillingProfileIDIn(input.BillingProfiles...))
		}

		// If we are filtering by customers without pinned profiles, we need to include all customers
		if input.CustomersWithoutPinnedProfile {
			input.IncludeAllCustomers = true
		}

		if !input.IncludeAllCustomers {
			query = query.Where(dbcustomer.HasBillingCustomerOverrideWith(customerOverrideFilters...))
		} else if input.CustomersWithoutPinnedProfile {
			query = query.Where(dbcustomer.Not(dbcustomer.HasBillingCustomerOverrideWith(customerOverrideFilters...)))
		} else {
			// We need to understand if the default profile is being queried for or not

			shouldIncludeDefaultProfile := false
			if len(input.BillingProfiles) == 0 {
				shouldIncludeDefaultProfile = true
			} else {
				// Let's see if we are interested in the default profile
				defaultProfile, err := tx.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{
					Namespace: input.Namespace,
				})
				if err != nil {
					return billing.ListCustomerOverridesAdapterResult{}, err
				}

				shouldIncludeDefaultProfile = slices.Contains(input.BillingProfiles, defaultProfile.ID)
			}

			if shouldIncludeDefaultProfile {
				query = query.Where(
					dbcustomer.Or(
						dbcustomer.HasBillingCustomerOverrideWith(customerOverrideFilters...),
						dbcustomer.Not(dbcustomer.HasBillingCustomerOverride()),
					),
				)
			} else {
				query = query.Where(dbcustomer.HasBillingCustomerOverrideWith(customerOverrideFilters...))
			}
		}

		query = query.WithBillingCustomerOverride(func(overrideQuery *db.BillingCustomerOverrideQuery) {
			overrideQuery = overrideQuery.Where(billingcustomeroverride.NamespaceEQ(input.Namespace)).
				Where(billingcustomeroverride.DeletedAtIsNil())

			overrideQuery.WithBillingProfile(func(profileQuery *db.BillingProfileQuery) {
				profileQuery.WithWorkflowConfig()
			})
		})

		res, err := query.Paginate(ctx, input.Page)
		if err != nil {
			return billing.ListCustomerOverridesAdapterResult{}, err
		}

		return pagination.MapResultErr(res, func(dbCustomer *db.Customer) (billing.CustomerOverrideWithCustomerID, error) {
			if dbCustomer.Edges.BillingCustomerOverride == nil {
				return billing.CustomerOverrideWithCustomerID{
					CustomerID: customer.CustomerID{
						Namespace: dbCustomer.Namespace,
						ID:        dbCustomer.ID,
					},
				}, nil
			}

			override, err := mapCustomerOverrideFromDB(dbCustomer.Edges.BillingCustomerOverride)
			if err != nil {
				return billing.CustomerOverrideWithCustomerID{}, err
			}

			return billing.CustomerOverrideWithCustomerID{
				CustomerOverride: override,
				CustomerID: customer.CustomerID{
					Namespace: dbCustomer.Namespace,
					ID:        dbCustomer.ID,
				},
			}, nil
		})
	})
}

func (a *adapter) DeleteCustomerOverride(ctx context.Context, input billing.DeleteCustomerOverrideInput) error {
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

func (a *adapter) GetCustomerOverrideReferencingProfile(ctx context.Context, input billing.HasCustomerOverrideReferencingProfileAdapterInput) ([]customer.CustomerID, error) {
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
	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		creates := make([]*db.BillingCustomerOverrideCreate, len(input.CustomerIDs))
		for i, customerID := range input.CustomerIDs {
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
				Exec(ctx)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func mapCustomerOverrideFromDB(dbOverride *db.BillingCustomerOverride) (*billing.CustomerOverride, error) {
	collectionInterval, err := dbOverride.LineCollectionPeriod.ParsePtrOrNil()
	if err != nil {
		return nil, fmt.Errorf("cannot parse collection.interval: %w", err)
	}

	draftPeriod, err := dbOverride.InvoiceDraftPeriod.ParsePtrOrNil()
	if err != nil {
		return nil, fmt.Errorf("cannot parse invoicing.draftPeriod: %w", err)
	}

	dueAfter, err := dbOverride.InvoiceDueAfter.ParsePtrOrNil()
	if err != nil {
		return nil, fmt.Errorf("cannot parse invoicing.dueAfter: %w", err)
	}

	baseProfile, err := mapProfileFromDB(dbOverride.Edges.BillingProfile)
	if err != nil {
		return nil, fmt.Errorf("cannot map profile: %w", err)
	}

	var profile *billing.Profile
	if baseProfile != nil {
		profile = &billing.Profile{
			BaseProfile: baseProfile.BaseProfile,
		}
	}

	return &billing.CustomerOverride{
		ID:        dbOverride.ID,
		Namespace: dbOverride.Namespace,

		CreatedAt: dbOverride.CreatedAt,
		UpdatedAt: dbOverride.UpdatedAt,

		CustomerID: dbOverride.CustomerID,
		Collection: billing.CollectionOverrideConfig{
			Alignment:               dbOverride.CollectionAlignment,
			AnchoredAlignmentDetail: dbOverride.AnchoredAlignmentDetail,
			Interval:                collectionInterval,
		},

		Invoicing: billing.InvoicingOverrideConfig{
			AutoAdvance:        dbOverride.InvoiceAutoAdvance,
			DraftPeriod:        draftPeriod,
			DueAfter:           dueAfter,
			ProgressiveBilling: dbOverride.InvoiceProgressiveBilling,
			DefaultTaxConfig:   lo.EmptyableToPtr(dbOverride.InvoiceDefaultTaxConfig),
		},

		Payment: billing.PaymentOverrideConfig{
			CollectionMethod: dbOverride.InvoiceCollectionMethod,
		},

		Profile: profile,
	}, nil
}
