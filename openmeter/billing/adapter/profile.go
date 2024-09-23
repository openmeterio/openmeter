package adapter

import (
	"context"
	"errors"
	"time"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/provider"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingprofile"
	"github.com/samber/lo"
)

var ErrDefaultProfileAlreadyExists = errors.New("default profile already exists in namespace")

func (r repository) CreateProfile(ctx context.Context, params billing.CreateProfileInput) (*billing.Profile, error) {
	c := r.client()

	dbProfile, err := c.BillingProfile.Create().
		SetNamespace(params.Namespace).
		SetKey(params.Key).
		SetDefault(params.Default).
		SetTaxProvider(provider.TaxProvider(params.TaxConfiguration.Type)).
		SetTaxProviderConfig(params.TaxConfiguration).
		SetInvoicingProvider(provider.InvoicingProvider(params.InvoicingConfiguration.Type)).
		SetInvoicingProviderConfig(params.InvoicingConfiguration).
		SetPaymentProvider(provider.PaymentProvider(params.PaymentConfiguration.Type)).
		SetPaymentProviderConfig(params.PaymentConfiguration).
		SetWorkflowConfig(
			mapWorkflowConfigToDB(params.Namespace, params.Configuration),
		).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return mapProfileFromDB(dbProfile), nil
}

func (r repository) GetProfileByKey(ctx context.Context, params billing.RepoGetProfileByKeyInput) (*billing.Profile, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	dbProfile, err := r.client().BillingProfile.Query().
		Where(billingprofile.Namespace(params.Namespace)).
		Where(billingprofile.Key(params.Key)).
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

func (r repository) GetDefaultProfile(ctx context.Context, params billing.RepoGetDefaultProfileInput) (*billing.Profile, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	dbProfile, err := r.client().BillingProfile.Query().
		Where(billingprofile.Namespace(params.Namespace)).
		Where(billingprofile.Default(true)).
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

func mapProfileFromDB(db *db.BillingProfile) *billing.Profile {
	return &billing.Profile{
		Namespace: db.Namespace,
		ID:        db.ID,
		Key:       db.Key,
		Default:   db.Default,

		CreatedAt: db.CreatedAt,
		UpdatedAt: db.UpdatedAt,
		DeletedAt: db.DeletedAt,

		TaxConfiguration:       db.TaxProviderConfig,
		InvoicingConfiguration: db.InvoicingProviderConfig,
		PaymentConfiguration:   db.PaymentProviderConfig,

		Configuration: mapWorkflowConfigFromDB(db.Edges.WorkflowConfig),
	}
}

func mapWorkflowConfigFromDB(dbWC *db.BillingWorkflowConfig) billing.Configuration {
	return billing.Configuration{
		ItemCollection: &billing.ItemCollectionConfig{
			Period: time.Duration(dbWC.ItemCollectionPeriodSeconds) * time.Second,
		},
		Workflow: &billing.WorkflowConfig{
			AutoAdvance:      dbWC.InvoiceAutoAdvance,
			DraftPeriod:      time.Duration(dbWC.InvoiceDraftPeriodSeconds) * time.Second,
			DueAfter:         time.Duration(dbWC.InvoiceDueAfterSeconds) * time.Second,
			CollectionMethod: dbWC.InvoiceCollectionMethod,
		},
		Granuality: &billing.GranualityConfig{
			Resolution:        dbWC.InvoiceLineItemResolution,
			PerSubjectDetails: dbWC.InvoiceLineItemPerSubject,
		},
	}
}

func mapWorkflowConfigToDB(ns string, conf billing.Configuration) *db.BillingWorkflowConfig {
	return &db.BillingWorkflowConfig{
		Namespace:                   ns,
		Alignment:                   billing.AlignmentKindSubscription,
		ItemCollectionPeriodSeconds: int64(conf.ItemCollection.Period / time.Second),
		InvoiceAutoAdvance:          conf.Workflow.AutoAdvance,
		InvoiceDraftPeriodSeconds:   int64(conf.Workflow.DraftPeriod / time.Second),
		InvoiceDueAfterSeconds:      int64(conf.Workflow.DueAfter / time.Second),
		InvoiceCollectionMethod:     conf.Workflow.CollectionMethod,
		InvoiceLineItemResolution:   conf.Granuality.Resolution,
		InvoiceLineItemPerSubject:   conf.Granuality.PerSubjectDetails,
	}
}

func secondsToDuration(s *int64) time.Duration {
	return time.Duration(lo.FromPtrOr(s, 0)) * time.Second
}
