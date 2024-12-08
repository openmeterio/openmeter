package billingadapter

import (
	"context"
	"database/sql"
	"fmt"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingcustomeroverride"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

var _ billing.CustomerOverrideAdapter = (*adapter)(nil)

func (a *adapter) CreateCustomerOverride(ctx context.Context, input billing.CreateCustomerOverrideInput) (*billing.CustomerOverride, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (*billing.CustomerOverride, error) {
		_, err := tx.db.BillingCustomerOverride.Create().
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
		return tx.GetCustomerOverride(ctx, billing.GetCustomerOverrideAdapterInput{
			Customer: customerentity.CustomerID{
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

		return tx.GetCustomerOverride(ctx, billing.GetCustomerOverrideAdapterInput{
			Customer: customerentity.CustomerID{
				Namespace: input.Namespace,
				ID:        input.CustomerID,
			},
		})
	})
}

func (a *adapter) GetCustomerOverride(ctx context.Context, input billing.GetCustomerOverrideAdapterInput) (*billing.CustomerOverride, error) {
	query := a.db.BillingCustomerOverride.Query().
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

func (a *adapter) DeleteCustomerOverride(ctx context.Context, input billing.DeleteCustomerOverrideInput) error {
	rowsAffected, err := a.db.BillingCustomerOverride.Update().
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

func (a *adapter) GetCustomerOverrideReferencingProfile(ctx context.Context, input billing.HasCustomerOverrideReferencingProfileAdapterInput) ([]customerentity.CustomerID, error) {
	dbCustomerOverrides, err := a.db.BillingCustomerOverride.Query().
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

func (a *adapter) UpsertCustomerOverride(ctx context.Context, input billing.UpsertCustomerOverrideAdapterInput) error {
	err := a.db.BillingCustomerOverride.Create().
		SetNamespace(input.Namespace).
		SetCustomerID(input.ID).
		OnConflict(
			entsql.DoNothing(),
		).
		Exec(ctx)
	if err != nil {
		// The do nothing returns no lines, so we have the record ready
		if err == sql.ErrNoRows {
			return nil
		}
	}
	return nil
}

func (a *adapter) LockCustomerForUpdate(ctx context.Context, input billing.LockCustomerForUpdateAdapterInput) error {
	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		if err := tx.UpsertCustomerOverride(ctx, input); err != nil {
			return err
		}

		_, err := tx.db.BillingCustomerOverride.Query().
			Where(billingcustomeroverride.CustomerID(input.ID)).
			Where(billingcustomeroverride.Namespace(input.Namespace)).
			ForUpdate().
			First(ctx)

		return err
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
			BaseProfile: *baseProfile,
		}
	}

	return &billing.CustomerOverride{
		ID:        dbOverride.ID,
		Namespace: dbOverride.Namespace,

		CreatedAt: dbOverride.CreatedAt,
		UpdatedAt: dbOverride.UpdatedAt,

		CustomerID: dbOverride.CustomerID,
		Collection: billing.CollectionOverrideConfig{
			Alignment: dbOverride.CollectionAlignment,
			Interval:  collectionInterval,
		},

		Invoicing: billing.InvoicingOverrideConfig{
			AutoAdvance: dbOverride.InvoiceAutoAdvance,
			DraftPeriod: draftPeriod,
			DueAfter:    dueAfter,
		},

		Payment: billing.PaymentOverrideConfig{
			CollectionMethod: dbOverride.InvoiceCollectionMethod,
		},

		Profile: profile,
	}, nil
}
