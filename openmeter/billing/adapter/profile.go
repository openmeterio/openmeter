package billingadapter

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingprofile"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingworkflowconfig"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

var _ billing.ProfileAdapter = (*adapter)(nil)

func (a *adapter) CreateProfile(ctx context.Context, input billing.CreateProfileInput) (*billingentity.BaseProfile, error) {
	if err := input.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	dbWorkflowConfig, err := a.db.BillingWorkflowConfig.Create().
		SetNamespace(input.Namespace).
		SetCollectionAlignment(input.WorkflowConfig.Collection.Alignment).
		SetItemCollectionPeriod(input.WorkflowConfig.Collection.Interval.ISOString()).
		SetInvoiceAutoAdvance(*input.WorkflowConfig.Invoicing.AutoAdvance).
		SetInvoiceDraftPeriod(input.WorkflowConfig.Invoicing.DraftPeriod.ISOString()).
		SetInvoiceDueAfter(input.WorkflowConfig.Invoicing.DueAfter.ISOString()).
		SetInvoiceCollectionMethod(input.WorkflowConfig.Payment.CollectionMethod).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	dbProfile, err := a.db.BillingProfile.Create().
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
}

func (a *adapter) GetProfile(ctx context.Context, input billing.GetProfileInput) (*billingentity.BaseProfile, error) {
	// This needs to be wrapped, as the service expects this to be atomic
	if err := input.Validate(); err != nil {
		return nil, err
	}

	dbProfile, err := a.db.BillingProfile.Query().
		Where(billingprofile.Namespace(input.Namespace)).
		Where(billingprofile.ID(input.ID)).
		WithWorkflowConfig().First(ctx)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, nil
		}

		return nil, err
	}

	return mapProfileFromDB(dbProfile)
}

func (a *adapter) ListProfiles(ctx context.Context, input billing.ListProfilesInput) (pagination.PagedResponse[billingentity.BaseProfile], error) {
	query := a.db.BillingProfile.Query().
		Where(billingprofile.Namespace(input.Namespace))

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

	response := pagination.PagedResponse[billingentity.BaseProfile]{
		Page: input.Page,
	}

	paged, err := query.Paginate(ctx, input.Page)
	if err != nil {
		return response, err
	}

	result := make([]billingentity.BaseProfile, 0, len(paged.Items))
	for _, item := range paged.Items {
		if item == nil {
			a.logger.Warn("invalid query result: nil billing profile received")
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

func (a adapter) GetDefaultProfile(ctx context.Context, input billing.GetDefaultProfileInput) (*billingentity.BaseProfile, error) {
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

	profile, err := a.GetProfile(ctx, billing.GetProfileInput(input))
	if err != nil {
		return err
	}

	_, err = a.db.BillingWorkflowConfig.UpdateOneID(profile.WorkflowConfig.ID).
		Where(billingworkflowconfig.Namespace(profile.Namespace)).
		SetDeletedAt(clock.Now()).
		Save(ctx)
	if err != nil {
		return err
	}

	_, err = a.db.BillingProfile.UpdateOneID(input.ID).
		Where(billingprofile.Namespace(input.Namespace)).
		SetDeletedAt(clock.Now()).
		Save(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (a adapter) UpdateProfile(ctx context.Context, input billing.UpdateProfileAdapterInput) (*billingentity.BaseProfile, error) {
	if err := input.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	targetState := input.TargetState

	update := a.db.BillingProfile.UpdateOneID(targetState.ID).
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

	updatedWorkflowConfig, err := a.db.BillingWorkflowConfig.UpdateOneID(input.WorkflowConfigID).
		Where(billingworkflowconfig.Namespace(targetState.Namespace)).
		SetCollectionAlignment(targetState.WorkflowConfig.Collection.Alignment).
		SetItemCollectionPeriod(targetState.WorkflowConfig.Collection.Interval.ISOString()).
		SetInvoiceAutoAdvance(*targetState.WorkflowConfig.Invoicing.AutoAdvance).
		SetInvoiceDraftPeriod(targetState.WorkflowConfig.Invoicing.DraftPeriod.ISOString()).
		SetInvoiceDueAfter(targetState.WorkflowConfig.Invoicing.DueAfter.ISOString()).
		SetInvoiceCollectionMethod(targetState.WorkflowConfig.Payment.CollectionMethod).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	updatedProfile.Edges.WorkflowConfig = updatedWorkflowConfig
	return mapProfileFromDB(updatedProfile)
}

func mapProfileFromDB(dbProfile *db.BillingProfile) (*billingentity.BaseProfile, error) {
	if dbProfile == nil {
		return nil, nil
	}

	wfConfig, err := mapWorkflowConfigFromDB(dbProfile.Edges.WorkflowConfig)
	if err != nil {
		return nil, fmt.Errorf("cannot map workflow config: %w", err)
	}

	return &billingentity.BaseProfile{
		Namespace:   dbProfile.Namespace,
		ID:          dbProfile.ID,
		Default:     dbProfile.Default,
		Name:        dbProfile.Name,
		Description: dbProfile.Description,
		Metadata:    dbProfile.Metadata,

		CreatedAt: dbProfile.CreatedAt,
		UpdatedAt: dbProfile.UpdatedAt,
		DeletedAt: dbProfile.DeletedAt,

		Supplier: billingentity.SupplierContact{
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

		AppReferences: &billingentity.ProfileAppReferences{
			Tax:       billingentity.AppReference{ID: dbProfile.TaxAppID},
			Invoicing: billingentity.AppReference{ID: dbProfile.InvoicingAppID},
			Payment:   billingentity.AppReference{ID: dbProfile.PaymentAppID},
		},
	}, nil
}

func mapWorkflowConfigFromDB(dbWC *db.BillingWorkflowConfig) (billingentity.WorkflowConfig, error) {
	collectionInterval, err := dbWC.ItemCollectionPeriod.Parse()
	if err != nil {
		return billingentity.WorkflowConfig{}, fmt.Errorf("cannot parse collection.interval: %w", err)
	}

	draftPeriod, err := dbWC.InvoiceDraftPeriod.Parse()
	if err != nil {
		return billingentity.WorkflowConfig{}, fmt.Errorf("cannot parse invoicing.draftPeriod: %w", err)
	}

	dueAfter, err := dbWC.InvoiceDueAfter.Parse()
	if err != nil {
		return billingentity.WorkflowConfig{}, fmt.Errorf("cannot parse invoicing.dueAfter: %w", err)
	}

	return billingentity.WorkflowConfig{
		ID: dbWC.ID,

		CreatedAt: dbWC.CreatedAt,
		UpdatedAt: dbWC.UpdatedAt,
		DeletedAt: dbWC.DeletedAt,

		Collection: billingentity.CollectionConfig{
			Alignment: dbWC.CollectionAlignment,
			Interval:  collectionInterval,
		},

		Invoicing: billingentity.InvoicingConfig{
			AutoAdvance: lo.ToPtr(dbWC.InvoiceAutoAdvance),
			DraftPeriod: draftPeriod,
			DueAfter:    dueAfter,
		},

		Payment: billingentity.PaymentConfig{
			CollectionMethod: dbWC.InvoiceCollectionMethod,
		},
	}, nil
}
