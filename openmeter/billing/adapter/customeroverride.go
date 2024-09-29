package billingadapter

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingcustomeroverride"
	"github.com/openmeterio/openmeter/pkg/clock"
)

var _ billing.CustomerOverrideAdapter = (*adapter)(nil)

func (r adapter) CreateCustomerOverride(ctx context.Context, input billing.CreateCustomerOverrideInput) (*billing.CustomerOverride, error) {
	if r.tx == nil {
		return nil, fmt.Errorf("create customer override: %w", ErrTransactionRequired)
	}

	_, err := r.client().BillingCustomerOverride.Create().
		SetNamespace(input.Namespace).
		SetCustomerID(input.CustomerID).
		SetNillableBillingProfileID(lo.EmptyableToPtr(input.ProfileID)).
		SetNillableCollectionAlignment(input.Collection.Alignment).
		SetNillableItemCollectionPeriodSeconds(durationPtrToSecondsPtr(input.Collection.ItemCollectionPeriod)).
		SetNillableInvoiceAutoAdvance(input.Invoicing.AutoAdvance).
		SetNillableInvoiceDraftPeriodSeconds(durationPtrToSecondsPtr(input.Invoicing.DraftPeriod)).
		SetNillableInvoiceDueAfterSeconds(durationPtrToSecondsPtr(input.Invoicing.DueAfter)).
		SetNillableInvoiceItemResolution(input.Invoicing.ItemResolution).
		SetNillableInvoiceItemPerSubject(input.Invoicing.ItemPerSubject).
		SetNillableInvoiceCollectionMethod(input.Payment.CollectionMethod).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	// Let's fetch the override with edges
	return r.GetCustomerOverride(ctx, billing.GetCustomerOverrideAdapterInput{
		Namespace:  input.Namespace,
		CustomerID: input.CustomerID,
	})
}

func (r adapter) UpdateCustomerOverride(ctx context.Context, input billing.UpdateCustomerOverrideAdapterInput) (*billing.CustomerOverride, error) {
	if r.tx == nil {
		return nil, fmt.Errorf("update customer override: %w", ErrTransactionRequired)
	}

	update := r.client().BillingCustomerOverride.Update().
		Where(billingcustomeroverride.CustomerID(input.CustomerID)).
		SetOrClearBillingProfileID(lo.EmptyableToPtr(input.ProfileID)).
		SetOrClearCollectionAlignment(input.Collection.Alignment).
		SetOrClearItemCollectionPeriodSeconds(durationPtrToSecondsPtr(input.Collection.ItemCollectionPeriod)).
		SetOrClearInvoiceAutoAdvance(input.Invoicing.AutoAdvance).
		SetOrClearInvoiceDraftPeriodSeconds(durationPtrToSecondsPtr(input.Invoicing.DraftPeriod)).
		SetOrClearInvoiceDueAfterSeconds(durationPtrToSecondsPtr(input.Invoicing.DueAfter)).
		SetOrClearInvoiceItemResolution(input.Invoicing.ItemResolution).
		SetOrClearInvoiceItemPerSubject(input.Invoicing.ItemPerSubject).
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
		Namespace:  input.Namespace,
		CustomerID: input.CustomerID,
	})
}

func (r adapter) GetCustomerOverride(ctx context.Context, input billing.GetCustomerOverrideAdapterInput) (*billing.CustomerOverride, error) {
	query := r.client().BillingCustomerOverride.Query().
		Where(billingcustomeroverride.Namespace(input.Namespace)).
		Where(billingcustomeroverride.CustomerID(input.CustomerID)).
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
			ID:     input.CustomerID,
			Entity: billing.EntityCustomer,
			Err:    billing.ErrCustomerNotFound,
		}
	}

	return mapCustomerOverrideFromDB(dbCustomerOverride), nil
}

func (r adapter) DeleteCustomerOverride(ctx context.Context, input billing.DeleteCustomerOverrideInput) error {
	rowsAffected, err := r.client().BillingCustomerOverride.Update().
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

func (r adapter) GetCustomerOverrideReferencingProfile(ctx context.Context, input billing.HasCustomerOverrideReferencingProfileAdapterInput) ([]customerentity.CustomerID, error) {
	dbCustomerOverrides, err := r.client().BillingCustomerOverride.Query().
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

func mapCustomerOverrideFromDB(dbOverride *db.BillingCustomerOverride) *billing.CustomerOverride {
	return &billing.CustomerOverride{
		ID:        dbOverride.ID,
		Namespace: dbOverride.Namespace,

		CreatedAt: dbOverride.CreatedAt,
		UpdatedAt: dbOverride.UpdatedAt,

		CustomerID: dbOverride.CustomerID,
		Profile:    mapProfileFromDB(dbOverride.Edges.BillingProfile),
		Collection: billing.CollectionOverrideConfig{
			Alignment:            dbOverride.CollectionAlignment,
			ItemCollectionPeriod: secondsPtrToDurationPtr(dbOverride.ItemCollectionPeriodSeconds),
		},

		Invoicing: billing.InvoicingOverrideConfig{
			AutoAdvance:    dbOverride.InvoiceAutoAdvance,
			DraftPeriod:    secondsPtrToDurationPtr(dbOverride.InvoiceDraftPeriodSeconds),
			DueAfter:       secondsPtrToDurationPtr(dbOverride.InvoiceDueAfterSeconds),
			ItemResolution: dbOverride.InvoiceItemResolution,
			ItemPerSubject: dbOverride.InvoiceItemPerSubject,
		},

		Payment: billing.PaymentOverrideConfig{
			CollectionMethod: dbOverride.InvoiceCollectionMethod,
		},
	}
}

func durationPtrToSecondsPtr(d *time.Duration) *int64 {
	if d == nil {
		return nil
	}

	v := int64(*d / time.Second)
	return &v
}

func secondsPtrToDurationPtr(s *int64) *time.Duration {
	if s == nil {
		return nil
	}

	v := time.Duration(*s) * time.Second
	return &v
}
