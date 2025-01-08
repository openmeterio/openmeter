package billingadapter

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingprofile"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingworkflowconfig"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

var _ billing.ProfileAdapter = (*adapter)(nil)

func (a *adapter) CreateProfile(ctx context.Context, input billing.CreateProfileInput) (*billing.BaseProfile, error) {
	if err := input.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (*billing.BaseProfile, error) {
		// Create the new workflow config
		dbWorkflowConfig, err := tx.createWorkflowConfig(ctx, input.Namespace, input.WorkflowConfig)
		if err != nil {
			return nil, err
		}

		// Create the new profile
		dbProfile, err := tx.db.BillingProfile.Create().
			SetNamespace(input.Namespace).
			SetDefault(input.Default).
			SetName(input.Name).
			SetNillableDescription(input.Description).
			SetSupplierName(input.Supplier.Name).
			SetNillableSupplierTaxCode(input.Supplier.TaxCode).
			SetSupplierAddressCountry(*input.Supplier.Address.Country). // Validation is done at service level
			SetNillableSupplierAddressState(input.Supplier.Address.State).
			SetNillableSupplierAddressCity(input.Supplier.Address.City).
			SetNillableSupplierAddressPostalCode(input.Supplier.Address.PostalCode).
			SetNillableSupplierAddressLine1(input.Supplier.Address.Line1).
			SetNillableSupplierAddressLine2(input.Supplier.Address.Line2).
			SetNillableSupplierAddressPhoneNumber(input.Supplier.Address.PhoneNumber).
			SetWorkflowConfig(dbWorkflowConfig).
			SetInvoicingAppID(input.Apps.Invoicing.ID).
			SetPaymentAppID(input.Apps.Payment.ID).
			SetTaxAppID(input.Apps.Tax.ID).
			SetMetadata(input.Metadata).
			Save(ctx)
		if err != nil {
			return nil, err
		}

		// Hack: we need to add the edges back
		dbProfile.Edges.WorkflowConfig = dbWorkflowConfig

		return mapProfileFromDB(dbProfile)
	})
}

func (a *adapter) createWorkflowConfig(ctx context.Context, ns string, input billing.WorkflowConfig) (*db.BillingWorkflowConfig, error) {
	return a.db.BillingWorkflowConfig.Create().
		SetNamespace(ns).
		SetCollectionAlignment(input.Collection.Alignment).
		SetLineCollectionPeriod(input.Collection.Interval.ISOString()).
		SetInvoiceAutoAdvance(input.Invoicing.AutoAdvance).
		SetInvoiceDraftPeriod(input.Invoicing.DraftPeriod.ISOString()).
		SetInvoiceDueAfter(input.Invoicing.DueAfter.ISOString()).
		SetInvoiceCollectionMethod(input.Payment.CollectionMethod).
		SetInvoiceProgressiveBilling(input.Invoicing.ProgressiveBilling).
		Save(ctx)
}

func (a *adapter) GetProfile(ctx context.Context, input billing.GetProfileInput) (*billing.BaseProfile, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	dbProfile, err := a.db.BillingProfile.Query().
		Where(billingprofile.Namespace(input.Profile.Namespace)).
		Where(billingprofile.ID(input.Profile.ID)).
		WithWorkflowConfig().First(ctx)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, nil
		}

		return nil, err
	}

	return mapProfileFromDB(dbProfile)
}

func (a *adapter) ListProfiles(ctx context.Context, input billing.ListProfilesInput) (pagination.PagedResponse[billing.BaseProfile], error) {
	query := a.db.BillingProfile.Query().
		Where(billingprofile.Namespace(input.Namespace)).
		WithWorkflowConfig()

	if !input.IncludeArchived {
		query = query.Where(billingprofile.DeletedAtIsNil())
	}

	order := entutils.GetOrdering(sortx.OrderDefault)
	if !input.Order.IsDefaultValue() {
		order = entutils.GetOrdering(input.Order)
	}

	switch input.OrderBy {
	case api.BillingProfileOrderByCreatedAt:
		query = query.Order(billingprofile.ByCreatedAt(order...))
	case api.BillingProfileOrderByUpdatedAt:
		query = query.Order(billingprofile.ByUpdatedAt(order...))
	case api.BillingProfileOrderByName:
		query = query.Order(billingprofile.ByName(order...))
	case api.BillingProfileOrderByDefault:
		query = query.Order(billingprofile.ByDefault(order...))
	default:
		query = query.Order(billingprofile.ByCreatedAt(order...))
	}

	response := pagination.PagedResponse[billing.BaseProfile]{
		Page: input.Page,
	}

	paged, err := query.Paginate(ctx, input.Page)
	if err != nil {
		return response, err
	}

	result := make([]billing.BaseProfile, 0, len(paged.Items))
	for _, item := range paged.Items {
		if item == nil {
			a.logger.WarnContext(ctx, "invalid query result: nil billing profile received")
			continue
		}

		profile, err := mapProfileFromDB(item)
		if err != nil {
			return response, fmt.Errorf("cannot map profile: %w", err)
		}

		result = append(result, *profile)
	}

	response.TotalCount = paged.TotalCount
	response.Items = result

	return response, nil
}

func (a *adapter) GetDefaultProfile(ctx context.Context, input billing.GetDefaultProfileInput) (*billing.BaseProfile, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	dbProfile, err := a.db.BillingProfile.Query().
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

	return mapProfileFromDB(dbProfile)
}

func (a *adapter) DeleteProfile(ctx context.Context, input billing.DeleteProfileInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		profile, err := tx.GetProfile(ctx, billing.GetProfileInput{
			Profile: input,
		})
		if err != nil {
			return err
		}

		_, err = tx.db.BillingWorkflowConfig.UpdateOneID(profile.WorkflowConfig.ID).
			Where(billingworkflowconfig.Namespace(profile.Namespace)).
			SetDeletedAt(clock.Now()).
			Save(ctx)
		if err != nil {
			return err
		}

		_, err = tx.db.BillingProfile.UpdateOneID(input.ID).
			Where(billingprofile.Namespace(input.Namespace)).
			SetDeletedAt(clock.Now()).
			Save(ctx)
		if err != nil {
			return err
		}

		return nil
	})
}

func (a *adapter) UpdateProfile(ctx context.Context, input billing.UpdateProfileAdapterInput) (*billing.BaseProfile, error) {
	if err := input.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (*billing.BaseProfile, error) {
		targetState := input.TargetState

		update := tx.db.BillingProfile.UpdateOneID(targetState.ID).
			Where(billingprofile.Namespace(targetState.Namespace)).
			SetName(targetState.Name).
			SetNillableDescription(targetState.Description).
			SetSupplierName(targetState.Supplier.Name).
			SetSupplierAddressCountry(*targetState.Supplier.Address.Country).
			SetDefault(targetState.Default).
			SetOrClearSupplierTaxCode(targetState.Supplier.TaxCode).
			SetOrClearSupplierAddressState(targetState.Supplier.Address.State).
			SetOrClearSupplierAddressCity(targetState.Supplier.Address.City).
			SetOrClearSupplierAddressPostalCode(targetState.Supplier.Address.PostalCode).
			SetOrClearSupplierAddressLine1(targetState.Supplier.Address.Line1).
			SetOrClearSupplierAddressLine2(targetState.Supplier.Address.Line2).
			SetOrClearSupplierAddressPhoneNumber(targetState.Supplier.Address.PhoneNumber).
			SetMetadata(targetState.Metadata)

		updatedProfile, err := update.Save(ctx)
		if err != nil {
			return nil, err
		}

		updatedWorkflowConfig, err := tx.updateWorkflowConfig(ctx, targetState.Namespace, input.WorkflowConfigID, targetState.WorkflowConfig)
		if err != nil {
			return nil, err
		}

		updatedProfile.Edges.WorkflowConfig = updatedWorkflowConfig
		return mapProfileFromDB(updatedProfile)
	})
}

func (a *adapter) UnsetDefaultProfile(ctx context.Context, input billing.UnsetDefaultProfileInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		return tx.db.BillingProfile.Update().
			Where(billingprofile.Namespace(input.Namespace)).
			SetDefault(false).
			Exec(ctx)
	})
}

func (a *adapter) updateWorkflowConfig(ctx context.Context, ns string, id string, input billing.WorkflowConfig) (*db.BillingWorkflowConfig, error) {
	return a.db.BillingWorkflowConfig.UpdateOneID(id).
		Where(billingworkflowconfig.Namespace(ns)).
		SetCollectionAlignment(input.Collection.Alignment).
		SetLineCollectionPeriod(input.Collection.Interval.ISOString()).
		SetInvoiceAutoAdvance(input.Invoicing.AutoAdvance).
		SetInvoiceDraftPeriod(input.Invoicing.DraftPeriod.ISOString()).
		SetInvoiceDueAfter(input.Invoicing.DueAfter.ISOString()).
		SetInvoiceCollectionMethod(input.Payment.CollectionMethod).
		SetInvoiceProgressiveBilling(input.Invoicing.ProgressiveBilling).
		Save(ctx)
}

func mapProfileFromDB(dbProfile *db.BillingProfile) (*billing.BaseProfile, error) {
	if dbProfile == nil {
		return nil, nil
	}

	wfConfig, err := mapWorkflowConfigFromDB(dbProfile.Edges.WorkflowConfig)
	if err != nil {
		return nil, fmt.Errorf("cannot map workflow config: %w", err)
	}

	return &billing.BaseProfile{
		Namespace:   dbProfile.Namespace,
		ID:          dbProfile.ID,
		Default:     dbProfile.Default,
		Name:        dbProfile.Name,
		Description: dbProfile.Description,
		Metadata:    dbProfile.Metadata,

		CreatedAt: dbProfile.CreatedAt.In(time.UTC),
		UpdatedAt: dbProfile.UpdatedAt.In(time.UTC),
		DeletedAt: convert.TimePtrIn(dbProfile.DeletedAt, time.UTC),

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
			TaxCode: dbProfile.SupplierTaxCode,
		},

		WorkflowConfig: wfConfig,

		AppReferences: &billing.ProfileAppReferences{
			Tax:       billing.AppReference{ID: dbProfile.TaxAppID},
			Invoicing: billing.AppReference{ID: dbProfile.InvoicingAppID},
			Payment:   billing.AppReference{ID: dbProfile.PaymentAppID},
		},
	}, nil
}

func mapWorkflowConfigToDB(wc billing.WorkflowConfig) *db.BillingWorkflowConfig {
	return &db.BillingWorkflowConfig{
		ID: wc.ID,

		CreatedAt: wc.CreatedAt.In(time.UTC),
		UpdatedAt: wc.UpdatedAt.In(time.UTC),
		DeletedAt: convert.TimePtrIn(wc.DeletedAt, time.UTC),

		CollectionAlignment:     wc.Collection.Alignment,
		LineCollectionPeriod:    wc.Collection.Interval.ISOString(),
		InvoiceAutoAdvance:      wc.Invoicing.AutoAdvance,
		InvoiceDraftPeriod:      wc.Invoicing.DraftPeriod.ISOString(),
		InvoiceDueAfter:         wc.Invoicing.DueAfter.ISOString(),
		InvoiceCollectionMethod: wc.Payment.CollectionMethod,
	}
}

func mapWorkflowConfigFromDB(dbWC *db.BillingWorkflowConfig) (billing.WorkflowConfig, error) {
	collectionInterval, err := dbWC.LineCollectionPeriod.Parse()
	if err != nil {
		return billing.WorkflowConfig{}, fmt.Errorf("cannot parse collection.interval: %w", err)
	}

	draftPeriod, err := dbWC.InvoiceDraftPeriod.Parse()
	if err != nil {
		return billing.WorkflowConfig{}, fmt.Errorf("cannot parse invoicing.draftPeriod: %w", err)
	}

	dueAfter, err := dbWC.InvoiceDueAfter.Parse()
	if err != nil {
		return billing.WorkflowConfig{}, fmt.Errorf("cannot parse invoicing.dueAfter: %w", err)
	}

	return billing.WorkflowConfig{
		ID: dbWC.ID,

		CreatedAt: dbWC.CreatedAt.In(time.UTC),
		UpdatedAt: dbWC.UpdatedAt.In(time.UTC),
		DeletedAt: convert.TimePtrIn(dbWC.DeletedAt, time.UTC),

		Collection: billing.CollectionConfig{
			Alignment: dbWC.CollectionAlignment,
			Interval:  collectionInterval,
		},

		Invoicing: billing.InvoicingConfig{
			AutoAdvance:        dbWC.InvoiceAutoAdvance,
			DraftPeriod:        draftPeriod,
			DueAfter:           dueAfter,
			ProgressiveBilling: dbWC.InvoiceProgressiveBilling,
		},

		Payment: billing.PaymentConfig{
			CollectionMethod: dbWC.InvoiceCollectionMethod,
		},
	}, nil
}
