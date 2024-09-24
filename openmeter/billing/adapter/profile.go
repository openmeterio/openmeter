package billingadapter

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/provider"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingprofile"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingworkflowconfig"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ billing.ProfileAdapter = (*adapter)(nil)

func (a adapter) CreateProfile(ctx context.Context, input billing.CreateProfileInput) (*billing.Profile, error) {
	if a.tx == nil {
		return nil, fmt.Errorf("cannot create profile: %w", ErrTransactionRequired)
	}

	c := a.client()

	dbWorkflowConfig, err := c.BillingWorkflowConfig.Create().
		SetNamespace(input.Namespace).
		SetNillableTimezone(input.WorkflowConfig.Timezone).
		SetCollectionAlignment(input.WorkflowConfig.Collection.Alignment).
		SetItemCollectionPeriodSeconds(int64(input.WorkflowConfig.Collection.ItemCollectionPeriod / time.Second)).
		SetInvoiceAutoAdvance(input.WorkflowConfig.Invoicing.AutoAdvance).
		SetInvoiceDraftPeriodSeconds(int64(input.WorkflowConfig.Invoicing.DraftPeriod / time.Second)).
		SetInvoiceDueAfterSeconds(int64(input.WorkflowConfig.Invoicing.DueAfter / time.Second)).
		SetInvoiceItemResolution(input.WorkflowConfig.Invoicing.ItemResolution).
		SetInvoiceItemPerSubject(input.WorkflowConfig.Invoicing.ItemPerSubject).
		SetInvoiceCollectionMethod(input.WorkflowConfig.Payment.CollectionMethod).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	dbProfile, err := c.BillingProfile.Create().
		SetNamespace(input.Namespace).
		SetKey(input.Key).
		SetDefault(input.Default).
		SetTaxProvider(input.TaxConfiguration.Type).
		SetInvoicingProvider(input.InvoicingConfiguration.Type).
		SetPaymentProvider(input.PaymentConfiguration.Type).
		SetSupplierName(input.Supplier.Name).
		SetSupplierAddressCountry(*input.Supplier.Address.Country). // Validation is done at service level
		SetNillableSupplierAddressState(input.Supplier.Address.State).
		SetNillableSupplierAddressCity(input.Supplier.Address.City).
		SetNillableSupplierAddressPostalCode(input.Supplier.Address.PostalCode).
		SetNillableSupplierAddressLine1(input.Supplier.Address.Line1).
		SetNillableSupplierAddressLine2(input.Supplier.Address.Line2).
		SetNillableSupplierAddressPhoneNumber(input.Supplier.Address.PhoneNumber).
		SetWorkflowConfig(dbWorkflowConfig).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	// Hack: we need to add the edges back
	dbProfile.Edges.WorkflowConfig = dbWorkflowConfig

	return mapProfileFromDB(dbProfile), nil
}

func (a adapter) GetProfileByKeyOrID(ctx context.Context, input billing.GetProfileByKeyOrIDInput) (*billing.Profile, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	dbProfiles, err := a.client().BillingProfile.Query().
		Where(billingprofile.Namespace(input.Namespace)).
		Where(billingprofile.Or(
			billingprofile.And(billingprofile.Key(input.IDOrKey), billingprofile.DeletedAtIsNil()),
			billingprofile.ID(input.IDOrKey),
		)).
		WithWorkflowConfig().All(ctx)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, nil
		}

		return nil, err
	}

	for _, dbProfile := range dbProfiles {
		if dbProfile.Key == input.IDOrKey {
			return mapProfileFromDB(dbProfile), nil
		}
	}

	for _, dbProfile := range dbProfiles {
		if dbProfile.ID == input.IDOrKey {
			return mapProfileFromDB(dbProfile), nil
		}
	}

	return nil, nil
}

func (a adapter) GetProfileByID(ctx context.Context, input billing.GetProfileByIDAdapterInput) (*billing.Profile, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	dbProfile, err := a.client().BillingProfile.Query().
		Where(billingprofile.Namespace(input.Namespace)).
		Where(billingprofile.ID(input.ID)).
		WithWorkflowConfig().
		Only(ctx)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, nil
		}

		return nil, err
	}

	return mapProfileFromDB(dbProfile), nil
}

func (a adapter) GetProfileByKey(ctx context.Context, input billing.GetProfileByKeyAdapterInput) (*billing.Profile, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	dbProfile, err := a.client().BillingProfile.Query().
		Where(billingprofile.Namespace(input.Namespace)).
		Where(billingprofile.Key(input.Key)).
		Where(billingprofile.DeletedAtIsNil()).
		WithWorkflowConfig().
		Only(ctx)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, nil
		}

		return nil, err
	}

	return mapProfileFromDB(dbProfile), nil
}

func (a adapter) GetDefaultProfile(ctx context.Context, input billing.GetDefaultProfileInput) (*billing.Profile, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	dbProfile, err := a.client().BillingProfile.Query().
		Where(billingprofile.Namespace(input.Namespace)).
		Where(billingprofile.Default(true)).
		Where(billingprofile.DeletedAtIsNil()).
		WithWorkflowConfig().
		Only(ctx)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, nil
		}

		return nil, err
	}

	return mapProfileFromDB(dbProfile), nil
}

func (a adapter) DeleteProfile(ctx context.Context, input billing.DeleteProfileAdapterInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	if a.tx == nil {
		return fmt.Errorf("cannot delete profile: %w", ErrTransactionRequired)
	}

	profile, err := a.GetProfileByID(ctx, billing.GetProfileByIDAdapterInput(input))
	if err != nil {
		return err
	}

	c := a.client()

	_, err = c.BillingWorkflowConfig.UpdateOneID(profile.WorkflowConfig.ID).
		SetDeletedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return err
	}

	_, err = c.BillingProfile.UpdateOneID(input.ID).
		SetDeletedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (a adapter) UpdateProfile(ctx context.Context, input billing.UpdateProfileAdapterInput) (*billing.Profile, error) {
	if a.tx == nil {
		return nil, fmt.Errorf("cannot update profile: %w", ErrTransactionRequired)
	}

	if err := input.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	targetState := input.TargetState

	updatedProfile, err := a.client().BillingProfile.UpdateOneID(targetState.ID).
		Where(billingprofile.Namespace(targetState.Namespace)).
		SetTaxProvider(targetState.TaxConfiguration.Type).
		SetInvoicingProvider(targetState.InvoicingConfiguration.Type).
		SetPaymentProvider(targetState.PaymentConfiguration.Type).
		SetSupplierName(targetState.Supplier.Name).
		SetSupplierAddressCountry(*targetState.Supplier.Address.Country).
		SetNillableSupplierAddressState(targetState.Supplier.Address.State).
		SetNillableSupplierAddressCity(targetState.Supplier.Address.City).
		SetNillableSupplierAddressPostalCode(targetState.Supplier.Address.PostalCode).
		SetNillableSupplierAddressLine1(targetState.Supplier.Address.Line1).
		SetNillableSupplierAddressLine2(targetState.Supplier.Address.Line2).
		SetNillableSupplierAddressPhoneNumber(targetState.Supplier.Address.PhoneNumber).
		SetDefault(targetState.Default).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	updatedWorkflowConfig, err := a.client().BillingWorkflowConfig.UpdateOneID(input.WorkflowConfigID).
		Where(billingworkflowconfig.Namespace(targetState.Namespace)).
		SetNillableTimezone(targetState.WorkflowConfig.Timezone).
		SetCollectionAlignment(targetState.WorkflowConfig.Collection.Alignment).
		SetItemCollectionPeriodSeconds(int64(targetState.WorkflowConfig.Collection.ItemCollectionPeriod / time.Second)).
		SetInvoiceAutoAdvance(targetState.WorkflowConfig.Invoicing.AutoAdvance).
		SetInvoiceDraftPeriodSeconds(int64(targetState.WorkflowConfig.Invoicing.DraftPeriod / time.Second)).
		SetInvoiceDueAfterSeconds(int64(targetState.WorkflowConfig.Invoicing.DueAfter / time.Second)).
		SetInvoiceItemResolution(targetState.WorkflowConfig.Invoicing.ItemResolution).
		SetInvoiceItemPerSubject(targetState.WorkflowConfig.Invoicing.ItemPerSubject).
		SetInvoiceCollectionMethod(targetState.WorkflowConfig.Payment.CollectionMethod).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	updatedProfile.Edges.WorkflowConfig = updatedWorkflowConfig
	return mapProfileFromDB(updatedProfile), nil
}

func mapProfileFromDB(dbProfile *db.BillingProfile) *billing.Profile {
	return &billing.Profile{
		Namespace: dbProfile.Namespace,
		ID:        dbProfile.ID,
		Key:       dbProfile.Key,
		Default:   dbProfile.Default,

		CreatedAt: dbProfile.CreatedAt,
		UpdatedAt: dbProfile.UpdatedAt,
		DeletedAt: dbProfile.DeletedAt,

		Supplier: billing.SupplierContact{
			Name: dbProfile.SupplierName,
			Address: models.Address{
				Country:     dbProfile.SupplierAddressCountry,
				PostalCode:  dbProfile.SupplierAddressPostalCode,
				City:        dbProfile.SupplierAddressCity,
				State:       dbProfile.SupplierAddressState,
				Line1:       dbProfile.SupplierAddressLine1,
				Line2:       dbProfile.SupplierAddressLine2,
				PhoneNumber: dbProfile.SupplierAddressPhoneNumber,
			},
		},

		TaxConfiguration: provider.TaxConfiguration{
			Type: dbProfile.TaxProvider,
		},
		InvoicingConfiguration: provider.InvoicingConfiguration{
			Type: dbProfile.InvoicingProvider,
		},
		PaymentConfiguration: provider.PaymentConfiguration{
			Type: dbProfile.PaymentProvider,
		},

		WorkflowConfig: mapWorkflowConfigFromDB(dbProfile.Edges.WorkflowConfig),
	}
}

func mapWorkflowConfigFromDB(dbWC *db.BillingWorkflowConfig) billing.WorkflowConfig {
	return billing.WorkflowConfig{
		ID: dbWC.ID,

		CreatedAt: dbWC.CreatedAt,
		UpdatedAt: dbWC.UpdatedAt,
		DeletedAt: dbWC.DeletedAt,

		Timezone: dbWC.Timezone,

		Collection: billing.CollectionConfig{
			Alignment:            dbWC.CollectionAlignment,
			ItemCollectionPeriod: time.Duration(dbWC.ItemCollectionPeriodSeconds) * time.Second,
		},

		Invoicing: billing.InvoicingConfig{
			AutoAdvance: dbWC.InvoiceAutoAdvance,
			DraftPeriod: time.Duration(dbWC.InvoiceDraftPeriodSeconds) * time.Second,
			DueAfter:    time.Duration(dbWC.InvoiceDueAfterSeconds) * time.Second,

			ItemResolution: dbWC.InvoiceItemResolution,
			ItemPerSubject: dbWC.InvoiceItemPerSubject,
		},

		Payment: billing.PaymentConfig{
			CollectionMethod: dbWC.InvoiceCollectionMethod,
		},
	}
}
