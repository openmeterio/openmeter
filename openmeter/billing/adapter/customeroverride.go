package billingadapter

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingcustomeroverride"
	"github.com/openmeterio/openmeter/pkg/clock"
)

const (
	// defaultCustomerOverrideCacheSize is the default size of the customer override cache used for upsert operations
	defaultCustomerOverrideCacheSize = 10_000
)

var _ billing.CustomerOverrideAdapter = (*adapter)(nil)

func (r *adapter) CreateCustomerOverride(ctx context.Context, input billing.CreateCustomerOverrideInput) (*billingentity.CustomerOverride, error) {
	_, err := r.db.BillingCustomerOverride.Create().
		SetNamespace(input.Namespace).
		SetCustomerID(input.CustomerID).
		SetNillableBillingProfileID(lo.EmptyableToPtr(input.ProfileID)).
		SetNillableCollectionAlignment(input.Collection.Alignment).
		SetNillableLineCollectionPeriod(input.Collection.Interval.ISOStringPtrOrNil()).
		SetNillableInvoiceAutoAdvance(input.Invoicing.AutoAdvance).
		SetNillableInvoiceDraftPeriod(input.Invoicing.DraftPeriod.ISOStringPtrOrNil()).
		SetNillableInvoiceDueAfter(input.Invoicing.DueAfter.ISOStringPtrOrNil()).
		SetNillableInvoiceCollectionMethod(input.Payment.CollectionMethod).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	// Let's fetch the override with edges
	return r.GetCustomerOverride(ctx, billing.GetCustomerOverrideAdapterInput{
		Customer: customerentity.CustomerID{
			Namespace: input.Namespace,
			ID:        input.CustomerID,
		},
	})
}

func (r *adapter) UpdateCustomerOverride(ctx context.Context, input billing.UpdateCustomerOverrideAdapterInput) (*billingentity.CustomerOverride, error) {
	if input.ProfileID == "" {
		// Let's resolve the default profile
		defaultProfile, err := r.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{
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

	update := r.db.BillingCustomerOverride.Update().
		Where(billingcustomeroverride.CustomerID(input.CustomerID)).
		SetOrClearCollectionAlignment(input.Collection.Alignment).
		SetOrClearLineCollectionPeriod(input.Collection.Interval.ISOStringPtrOrNil()).
		SetOrClearInvoiceAutoAdvance(input.Invoicing.AutoAdvance).
		SetOrClearInvoiceDraftPeriod(input.Invoicing.DraftPeriod.ISOStringPtrOrNil()).
		SetOrClearInvoiceDueAfter(input.Invoicing.DueAfter.ISOStringPtrOrNil()).
		SetOrClearInvoiceCollectionMethod(input.Payment.CollectionMethod)

	if input.ResetDeletedAt {
		update = update.ClearDeletedAt()
	}

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

	return r.GetCustomerOverride(ctx, billing.GetCustomerOverrideAdapterInput{
		Customer: customerentity.CustomerID{
			Namespace: input.Namespace,
			ID:        input.CustomerID,
		},
	})
}

func (r *adapter) GetCustomerOverride(ctx context.Context, input billing.GetCustomerOverrideAdapterInput) (*billingentity.CustomerOverride, error) {
	query := r.db.BillingCustomerOverride.Query().
		Where(billingcustomeroverride.Namespace(input.Customer.Namespace)).
		Where(billingcustomeroverride.CustomerID(input.Customer.ID)).
		WithBillingProfile(func(bpq *db.BillingProfileQuery) {
			bpq.WithWorkflowConfig()
		}).
		WithCustomer()

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

	if dbCustomerOverride.Edges.Customer == nil {
		return nil, billing.NotFoundError{
			ID:     input.Customer.ID,
			Entity: billing.EntityCustomer,
			Err:    billing.ErrCustomerNotFound,
		}
	}

	return mapCustomerOverrideFromDB(dbCustomerOverride)
}

func (r *adapter) DeleteCustomerOverride(ctx context.Context, input billing.DeleteCustomerOverrideInput) error {
	rowsAffected, err := r.db.BillingCustomerOverride.Update().
		Where(billingcustomeroverride.CustomerID(input.CustomerID)).
		Where(billingcustomeroverride.Namespace(input.Namespace)).
		Where(billingcustomeroverride.DeletedAtIsNil()).
		SetDeletedAt(clock.Now()).
		Save(ctx)
	if err != nil {
		if db.IsNotFound(err) {
			return billing.NotFoundError{
				ID:     input.CustomerID,
				Entity: billing.EntityCustomerOverride,
				Err:    billing.ErrCustomerOverrideNotFound,
			}
		}

		return err
	}

	if rowsAffected == 0 {
		return billing.NotFoundError{
			ID:     input.CustomerID,
			Entity: billing.EntityCustomerOverride,
			Err:    billing.ErrCustomerOverrideNotFound,
		}
	}

	return nil
}

func (r *adapter) GetCustomerOverrideReferencingProfile(ctx context.Context, input billing.HasCustomerOverrideReferencingProfileAdapterInput) ([]customerentity.CustomerID, error) {
	dbCustomerOverrides, err := r.db.BillingCustomerOverride.Query().
		Where(billingcustomeroverride.Namespace(input.Namespace)).
		Where(billingcustomeroverride.BillingProfileID(input.ID)).
		Where(billingcustomeroverride.DeletedAtIsNil()).
		Select(billingcustomeroverride.FieldCustomerID).
		All(ctx)
	if err != nil {
		return nil, err
	}

	var customerIDs []customerentity.CustomerID
	for _, dbCustomerOverride := range dbCustomerOverrides {
		customerIDs = append(customerIDs, customerentity.CustomerID{
			Namespace: input.Namespace,
			ID:        dbCustomerOverride.CustomerID,
		})
	}

	return customerIDs, nil
}

func (r *adapter) UpsertCustomerOverrideIgnoringTrns(ctx context.Context, input billing.UpsertCustomerOverrideIgnoringTrnsAdapterInput) error {
	if val, ok := r.upsertCustomerOverrideCache.Get(input.ID); ok {
		if val.Namespace == input.Namespace {
			return nil
		}
	}

	ent, err := r.db.BillingCustomerOverride.Query().
		Where(billingcustomeroverride.CustomerID(input.ID)).
		Where(billingcustomeroverride.Namespace(input.Namespace)).
		First(ctx)
	if err != nil {
		if db.IsNotFound(err) {
			ent, err = r.db.BillingCustomerOverride.Create().
				SetNamespace(input.Namespace).
				SetCustomerID(input.ID).
				Save(ctx)
			if err != nil {
				// Most probably a conflict caused by concurrent upserts => let's try to fetch again
				ent, err = r.db.BillingCustomerOverride.Query().
					Where(billingcustomeroverride.CustomerID(input.ID)).
					Where(billingcustomeroverride.Namespace(input.Namespace)).
					First(ctx)
				if err != nil {
					return err
				}

				// Let's update the cache
				r.upsertCustomerOverrideCache.Add(input.ID, ent)
				return nil
			}

			// Let's update the cache
			r.upsertCustomerOverrideCache.Add(input.ID, ent)
			return nil
		}
		return err
	}
	r.upsertCustomerOverrideCache.Add(input.ID, ent)
	return nil
}

func (r *adapter) LockCustomerForUpdate(ctx context.Context, input billing.LockCustomerForUpdateAdapterInput) error {
	_, err := r.db.BillingCustomerOverride.Query().
		Where(billingcustomeroverride.CustomerID(input.ID)).
		Where(billingcustomeroverride.Namespace(input.Namespace)).
		ForUpdate().
		First(ctx)

	return err
}

func mapCustomerOverrideFromDB(dbOverride *db.BillingCustomerOverride) (*billingentity.CustomerOverride, error) {
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

	var profile *billingentity.Profile
	if baseProfile != nil {
		profile = &billingentity.Profile{
			BaseProfile: *baseProfile,
		}
	}

	return &billingentity.CustomerOverride{
		ID:        dbOverride.ID,
		Namespace: dbOverride.Namespace,

		CreatedAt: dbOverride.CreatedAt,
		UpdatedAt: dbOverride.UpdatedAt,

		CustomerID: dbOverride.CustomerID,
		Collection: billingentity.CollectionOverrideConfig{
			Alignment: dbOverride.CollectionAlignment,
			Interval:  collectionInterval,
		},

		Invoicing: billingentity.InvoicingOverrideConfig{
			AutoAdvance: dbOverride.InvoiceAutoAdvance,
			DraftPeriod: draftPeriod,
			DueAfter:    dueAfter,
		},

		Payment: billingentity.PaymentOverrideConfig{
			CollectionMethod: dbOverride.InvoiceCollectionMethod,
		},

		Profile: profile,
	}, nil
}
