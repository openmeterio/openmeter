package billingadapter

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingcustomeroverride"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoice"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceline"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingprofile"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingworkflowconfig"
	dbcustomer "github.com/openmeterio/openmeter/openmeter/ent/db/customer"
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

		createdProfile, err := mapProfileFromDB(dbProfile)
		if err != nil {
			return nil, err
		}

		return &createdProfile.BaseProfile, nil
	})
}

func (a *adapter) createWorkflowConfig(ctx context.Context, ns string, input billing.WorkflowConfig) (*db.BillingWorkflowConfig, error) {
	cmd := a.db.BillingWorkflowConfig.Create().
		SetNamespace(ns).
		SetCollectionAlignment(input.Collection.Alignment).
		SetLineCollectionPeriod(input.Collection.Interval.ISOString()).
		SetInvoiceAutoAdvance(input.Invoicing.AutoAdvance).
		SetInvoiceDraftPeriod(input.Invoicing.DraftPeriod.ISOString()).
		SetInvoiceDueAfter(input.Invoicing.DueAfter.ISOString()).
		SetInvoiceCollectionMethod(input.Payment.CollectionMethod).
		SetInvoiceProgressiveBilling(input.Invoicing.ProgressiveBilling).
		SetNillableInvoiceDefaultTaxSettings(input.Invoicing.DefaultTaxConfig).
		SetTaxEnabled(input.Tax.Enabled).
		SetTaxEnforced(input.Tax.Enforced)

	if input.Collection.AnchoredAlignmentDetail != nil {
		cmd = cmd.SetAnchoredAlignmentDetail(input.Collection.AnchoredAlignmentDetail)
	}

	return cmd.Save(ctx)
}

func (a *adapter) GetProfile(ctx context.Context, input billing.GetProfileInput) (*billing.AdapterGetProfileResponse, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	dbProfile, err := a.db.BillingProfile.Query().
		Where(billingprofile.Namespace(input.Profile.Namespace)).
		Where(billingprofile.ID(input.Profile.ID)).
		WithWorkflowConfig().First(ctx)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, billing.NotFoundError{
				Err: fmt.Errorf("%w [id=%s]", billing.ErrProfileNotFound, input.Profile.ID),
			}
		}

		return nil, err
	}

	return mapProfileFromDB(dbProfile)
}

func (a *adapter) ListProfiles(ctx context.Context, input billing.ListProfilesInput) (pagination.Result[billing.BaseProfile], error) {
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

	response := pagination.Result[billing.BaseProfile]{
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

		result = append(result, profile.BaseProfile)
	}

	response.TotalCount = paged.TotalCount
	response.Items = result

	return response, nil
}

func (a *adapter) GetDefaultProfile(ctx context.Context, input billing.GetDefaultProfileInput) (*billing.AdapterGetProfileResponse, error) {
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

		_, err = tx.db.BillingWorkflowConfig.UpdateOneID(profile.WorkflowConfigID).
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

		updatedProfileEntity, err := mapProfileFromDB(updatedProfile)
		if err != nil {
			return nil, err
		}

		return &updatedProfileEntity.BaseProfile, nil
	})
}

func (a *adapter) GetUnpinnedCustomerIDsWithPaidSubscription(ctx context.Context, input billing.GetUnpinnedCustomerIDsWithPaidSubscriptionInput) ([]customer.CustomerID, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]customer.CustomerID, error) {
		var out []customer.CustomerID

		err := tx.db.Customer.Query().
			Where(
				dbcustomer.NamespaceEQ(input.Namespace),
				dbcustomer.DeletedAtIsNil(),
				// Has outstanding line items belonging to a subscription (a paid subscription always has at least
				// one gathering line item, if there are still upcoming lines)
				dbcustomer.HasBillingInvoiceWith(
					billinginvoice.NamespaceEQ(input.Namespace),
					billinginvoice.StatusEQ(billing.StandardInvoiceStatusGathering),
					billinginvoice.DeletedAtIsNil(),
					billinginvoice.HasBillingInvoiceLinesWith(
						billinginvoiceline.DeletedAtIsNil(),
						billinginvoiceline.StatusEQ(billing.InvoiceLineStatusValid),
						billinginvoiceline.NamespaceEQ(input.Namespace),
						billinginvoiceline.SubscriptionIDNotNil(),
					),
				),
				// Has no customer override with explicit billing profile pinning
				dbcustomer.Or(
					// Either has a customer override with no billing profile id set
					dbcustomer.HasBillingCustomerOverrideWith(
						billingcustomeroverride.NamespaceEQ(input.Namespace),
						billingcustomeroverride.DeletedAtIsNil(),
						billingcustomeroverride.BillingProfileIDIsNil(),
					),
					// Or has no customer override at all
					dbcustomer.Not(
						dbcustomer.HasBillingCustomerOverrideWith(
							billingcustomeroverride.NamespaceEQ(input.Namespace),
							billingcustomeroverride.DeletedAtIsNil(),
						),
					),
				),
			).
			Select(dbcustomer.FieldNamespace, dbcustomer.FieldID).
			Scan(ctx, &out)
		if err != nil {
			return nil, err
		}

		return out, nil
	})
}

// isBillingProfileUsed checks if the app is used in any billing profile
func (a *adapter) isBillingProfileUsed(ctx context.Context, appID app.AppID) error {
	if err := appID.Validate(); err != nil {
		return fmt.Errorf("invalid app id: %w", err)
	}

	profiles, err := a.db.BillingProfile.Query().
		Where(

			billingprofile.Namespace(appID.Namespace),
			billingprofile.Or(
				billingprofile.InvoicingAppID(appID.ID),
				billingprofile.PaymentAppID(appID.ID),
				billingprofile.TaxAppID(appID.ID),
			),
			billingprofile.DeletedAtIsNil(),
		).
		All(ctx)
	if err != nil {
		return err
	}

	if len(profiles) > 0 {
		return models.NewGenericConflictError(fmt.Errorf("app is used in %d billing profiles: %s", len(profiles), strings.Join(lo.Map(profiles, func(profile *db.BillingProfile, _ int) string {
			return fmt.Sprintf("%s[%s]", profile.Name, profile.ID)
		}), ",")))
	}

	return nil
}

func (a *adapter) updateWorkflowConfig(ctx context.Context, ns string, id string, input billing.WorkflowConfig) (*db.BillingWorkflowConfig, error) {
	return a.db.BillingWorkflowConfig.UpdateOneID(id).
		Where(billingworkflowconfig.Namespace(ns)).
		SetCollectionAlignment(input.Collection.Alignment).
		SetAnchoredAlignmentDetail(input.Collection.AnchoredAlignmentDetail).
		SetLineCollectionPeriod(input.Collection.Interval.ISOString()).
		SetInvoiceAutoAdvance(input.Invoicing.AutoAdvance).
		SetInvoiceDraftPeriod(input.Invoicing.DraftPeriod.ISOString()).
		SetInvoiceDueAfter(input.Invoicing.DueAfter.ISOString()).
		SetInvoiceCollectionMethod(input.Payment.CollectionMethod).
		SetInvoiceProgressiveBilling(input.Invoicing.ProgressiveBilling).
		SetOrClearInvoiceDefaultTaxSettings(input.Invoicing.DefaultTaxConfig).
		SetTaxEnabled(input.Tax.Enabled).
		SetTaxEnforced(input.Tax.Enforced).
		Save(ctx)
}

func mapProfileFromDB(dbProfile *db.BillingProfile) (*billing.AdapterGetProfileResponse, error) {
	if dbProfile == nil {
		return nil, nil
	}

	wfConfig, err := mapWorkflowConfigFromDB(dbProfile.Edges.WorkflowConfig)
	if err != nil {
		return nil, fmt.Errorf("cannot map workflow config: %w", err)
	}

	return &billing.AdapterGetProfileResponse{
		BaseProfile: billing.BaseProfile{
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
				Tax:       app.AppID{Namespace: dbProfile.Namespace, ID: dbProfile.TaxAppID},
				Invoicing: app.AppID{Namespace: dbProfile.Namespace, ID: dbProfile.InvoicingAppID},
				Payment:   app.AppID{Namespace: dbProfile.Namespace, ID: dbProfile.PaymentAppID},
			},
		},
		WorkflowConfigID: dbProfile.Edges.WorkflowConfig.ID,
	}, nil
}

func mapWorkflowConfigToDB(wc billing.WorkflowConfig, id string) *db.BillingWorkflowConfig {
	return &db.BillingWorkflowConfig{
		ID: id,

		CollectionAlignment:     wc.Collection.Alignment,
		AnchoredAlignmentDetail: wc.Collection.AnchoredAlignmentDetail,
		LineCollectionPeriod:    wc.Collection.Interval.ISOString(),
		InvoiceAutoAdvance:      wc.Invoicing.AutoAdvance,
		InvoiceDraftPeriod:      wc.Invoicing.DraftPeriod.ISOString(),
		InvoiceDueAfter:         wc.Invoicing.DueAfter.ISOString(),
		InvoiceCollectionMethod: wc.Payment.CollectionMethod,
		TaxEnabled:              wc.Tax.Enabled,
		TaxEnforced:             wc.Tax.Enforced,
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
		Collection: billing.CollectionConfig{
			Alignment:               dbWC.CollectionAlignment,
			AnchoredAlignmentDetail: dbWC.AnchoredAlignmentDetail,
			Interval:                collectionInterval,
		},

		Invoicing: billing.InvoicingConfig{
			AutoAdvance:        dbWC.InvoiceAutoAdvance,
			DraftPeriod:        draftPeriod,
			DueAfter:           dueAfter,
			ProgressiveBilling: dbWC.InvoiceProgressiveBilling,
			DefaultTaxConfig:   lo.EmptyableToPtr(dbWC.InvoiceDefaultTaxSettings),
		},

		Payment: billing.PaymentConfig{
			CollectionMethod: dbWC.InvoiceCollectionMethod,
		},

		Tax: billing.WorkflowTaxConfig{
			Enabled:  dbWC.TaxEnabled,
			Enforced: dbWC.TaxEnforced,
		},
	}, nil
}
