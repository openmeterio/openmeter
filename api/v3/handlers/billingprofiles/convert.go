//go:generate go run github.com/jmattheis/goverter/cmd/goverter gen ./
package billingprofiles

import (
	"github.com/rickb777/period"
	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/labels"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
)

// goverter:variables
// goverter:skipCopySameType
// goverter:output:file ./convert.gen.go
// goverter:useZeroValueOnPointerInconsistency
// goverter:useUnderlyingTypeMethods
// goverter:matchIgnoreCase
// goverter:extend ToAPIBillingWorkflow
// goverter:extend ToAPIBillingParty
// goverter:extend ToAPIBillingAppReference
// goverter:extend FromAPIBillingParty
// goverter:extend ConvertMetadataToLabels
// goverter:extend ConvertLabelsToMetadata
var (
	ToAPIAddress                     func(address models.Address) api.Address
	FromAPIAddress                   func(address api.Address) models.Address
	ToAPIBillingProfileAppReferences func(refs *billing.ProfileAppReferences) api.BillingProfileAppReferences
	// goverter:context namespace
	FromAPIBillingProfileAppReferences func(namespace string, refs api.BillingProfileAppReferences) billing.ProfileAppReferences
	// goverter:context namespace
	// goverter:map Namespace | NamespaceFromContext
	FromAPIBillingAppReferenceToAppID func(namespace string, ref api.BillingAppReference) app.AppID
	ToAPIBillingAppReferenceFromAppID func(appID app.AppID) api.BillingAppReference
	// goverter:autoMap BaseProfile
	// goverter:map BaseProfile.Metadata Labels
	// goverter:map BaseProfile.WorkflowConfig Workflow
	ToAPIBillingProfile  func(profile billing.Profile) (api.BillingProfile, error)
	ToAPIBillingProfiles func(profiles []billing.Profile) ([]api.BillingProfile, error)
	// goverter:context namespace
	// goverter:map Namespace | NamespaceFromContext
	// goverter:map Labels Metadata | ConvertLabelsToMetadata
	// goverter:map Workflow WorkflowConfig | FromAPIBillingWorkflow
	FromAPICreateBillingProfileRequest func(namespace string, req api.CreateBillingProfileRequest) (billing.CreateProfileInput, error)
	// goverter:context namespacedID
	// goverter:map Namespace | ResolveNamespaceFromContext
	// goverter:map ID | ResolveIDFromContext
	// goverter:map Workflow WorkflowConfig | FromAPIBillingWorkflow
	// goverter:map Labels Metadata
	// goverter:ignore CreatedAt
	// goverter:ignore UpdatedAt
	// goverter:ignore DeletedAt
	// goverter:ignore AppReferences
	FromAPIUpsertBillingProfileRequest func(namespacedID models.NamespacedID, req api.UpsertBillingProfileRequest) (billing.UpdateProfileInput, error)
	// goverter:enum:unknown @error
	// goverter:enum:map InclusiveTaxBehavior BillingTaxBehaviorInclusive
	// goverter:enum:map ExclusiveTaxBehavior BillingTaxBehaviorExclusive
	ToAPIBillingTaxBehavior func(behavior productcatalog.TaxBehavior) (api.BillingTaxBehavior, error)
	// goverter:enum:unknown @error
	// goverter:enum:map BillingTaxBehaviorInclusive InclusiveTaxBehavior
	// goverter:enum:map BillingTaxBehaviorExclusive ExclusiveTaxBehavior
	FromAPIBillingTaxBehavior func(behavior api.BillingTaxBehavior) (productcatalog.TaxBehavior, error)
	// goverter:map Stripe ExternalInvoicing
	// goverter:map Stripe Stripe
	ToAPIBillingTaxConfig func(config *productcatalog.TaxConfig) (*api.BillingTaxConfig, error)
	// goverter:map Stripe Stripe
	// goverter:ignore TaxCode
	FromAPIBillingTaxConfig func(config *api.BillingTaxConfig) (*productcatalog.TaxConfig, error)
)

var (
	ConvertMetadataToLabels = labels.FromMetadata[billing.Metadata]
	ConvertLabelsToMetadata = func(l *api.Labels) (billing.Metadata, error) {
		m, err := labels.ToMetadata(l)

		return billing.Metadata(m), err
	}
)

//goverter:context namespace
func NamespaceFromContext(namespace string) string {
	return namespace
}

// goverter:context id
func IDFromContext(id string) string {
	return id
}

// goverter:context namespacedID
func ResolveNamespaceFromContext(namespacedID models.NamespacedID) string {
	return namespacedID.Namespace
}

// goverter:context namespacedID
func ResolveIDFromContext(namespacedID models.NamespacedID) string {
	return namespacedID.ID
}

// ToAPIBillingAppReference converts app.App to API BillingAppReference
func ToAPIBillingAppReference(app app.App) api.BillingAppReference {
	return api.BillingAppReference{
		Id: app.GetID().ID,
	}
}

// ToAPIBillingParty converts billing.SupplierContact to API BillingParty
func ToAPIBillingParty(supplier billing.SupplierContact) api.BillingParty {
	party := api.BillingParty{
		Id:   &supplier.ID,
		Name: &supplier.Name,
	}

	if supplier.Address.Country != nil {
		party.Addresses = &api.BillingPartyAddresses{
			BillingAddress: ToAPIAddress(supplier.Address),
		}
	}

	if supplier.TaxCode != nil {
		party.TaxId = &api.BillingPartyTaxIdentity{
			Code: supplier.TaxCode,
		}
	}

	return party
}

// FromAPIBillingParty converts API BillingParty to billing.SupplierContact
func FromAPIBillingParty(party api.BillingParty) billing.SupplierContact {
	supplier := billing.SupplierContact{
		ID:   lo.FromPtrOr(party.Id, ""),
		Name: lo.FromPtrOr(party.Name, ""),
	}

	if party.Addresses != nil {
		addr := party.Addresses.BillingAddress
		supplier.Address = FromAPIAddress(addr)
	}

	if party.TaxId != nil && party.TaxId.Code != nil {
		supplier.TaxCode = party.TaxId.Code
	}

	return supplier
}

// ToAPIBillingWorkflow converts billing.WorkflowConfig to API BillingWorkflow
func ToAPIBillingWorkflow(config billing.WorkflowConfig) (api.BillingWorkflow, error) {
	workflow := api.BillingWorkflow{}

	// Collection settings
	workflow.Collection = &api.BillingWorkflowCollectionSettings{
		Interval: lo.ToPtr(config.Collection.Interval.String()),
	}

	// Convert alignment
	switch config.Collection.Alignment {
	case billing.AlignmentKindSubscription:
		alignment := api.BillingWorkflowCollectionAlignment{}
		err := alignment.FromBillingWorkflowCollectionAlignmentSubscription(api.BillingWorkflowCollectionAlignmentSubscription{
			Type: "subscription",
		})
		if err != nil {
			return api.BillingWorkflow{}, err
		}
		workflow.Collection.Alignment = &alignment
	case billing.AlignmentKindAnchored:
		if config.Collection.AnchoredAlignmentDetail != nil {
			alignment := api.BillingWorkflowCollectionAlignment{}
			err := alignment.FromBillingWorkflowCollectionAlignmentAnchored(api.BillingWorkflowCollectionAlignmentAnchored{
				Type: "anchored",
				RecurringPeriod: api.RecurringPeriod{
					Anchor:   config.Collection.AnchoredAlignmentDetail.Anchor,
					Interval: config.Collection.AnchoredAlignmentDetail.Interval.String(),
				},
			})
			if err != nil {
				return api.BillingWorkflow{}, err
			}
			workflow.Collection.Alignment = &alignment
		}
	}

	// Invoicing settings
	workflow.Invoicing = &api.BillingWorkflowInvoicingSettings{
		AutoAdvance:        lo.ToPtr(config.Invoicing.AutoAdvance),
		DraftPeriod:        lo.ToPtr(config.Invoicing.DraftPeriod.String()),
		ProgressiveBilling: lo.ToPtr(config.Invoicing.ProgressiveBilling),
	}

	// Tax settings
	defaultTaxConfig, err := ToAPIBillingTaxConfig(config.Invoicing.DefaultTaxConfig)
	if err != nil {
		return api.BillingWorkflow{}, err
	}
	workflow.Tax = &api.BillingWorkflowTaxSettings{
		Enabled:          lo.ToPtr(config.Tax.Enabled),
		Enforced:         lo.ToPtr(config.Tax.Enforced),
		DefaultTaxConfig: defaultTaxConfig,
	}

	// Payment settings
	switch config.Payment.CollectionMethod {
	case billing.CollectionMethodChargeAutomatically:
		payment := api.BillingWorkflowPaymentSettings{}
		err := payment.FromBillingWorkflowPaymentChargeAutomaticallySettings(api.BillingWorkflowPaymentChargeAutomaticallySettings{
			CollectionMethod: "charge_automatically",
		})
		if err != nil {
			return api.BillingWorkflow{}, err
		}
		workflow.Payment = &payment
	case billing.CollectionMethodSendInvoice:
		payment := api.BillingWorkflowPaymentSettings{}
		err := payment.FromBillingWorkflowPaymentSendInvoiceSettings(api.BillingWorkflowPaymentSendInvoiceSettings{
			CollectionMethod: "send_invoice",
			DueAfter:         lo.ToPtr(config.Invoicing.DueAfter.String()),
		})
		if err != nil {
			return api.BillingWorkflow{}, err
		}
		workflow.Payment = &payment
	}

	return workflow, nil
}

// FromAPIBillingWorkflow converts API BillingWorkflow to billing.WorkflowConfig
func FromAPIBillingWorkflow(workflow api.BillingWorkflow) (billing.WorkflowConfig, error) {
	// Start with default configuration
	def := billing.DefaultWorkflowConfig

	// Ensure workflow sections are initialized
	if workflow.Collection == nil {
		workflow.Collection = &api.BillingWorkflowCollectionSettings{}
	}
	if workflow.Invoicing == nil {
		workflow.Invoicing = &api.BillingWorkflowInvoicingSettings{}
	}
	if workflow.Payment == nil {
		workflow.Payment = &api.BillingWorkflowPaymentSettings{}
	}
	if workflow.Tax == nil {
		workflow.Tax = &api.BillingWorkflowTaxSettings{}
	}

	// Parse collection interval with default fallback
	collInterval := def.Collection.Interval
	if workflow.Collection.Interval != nil {
		if parsed, err := period.Parse(*workflow.Collection.Interval); err == nil {
			collInterval = datetime.ISODuration{Period: parsed}
		}
	}

	// Parse collection alignment with default fallback
	alignment := def.Collection.Alignment
	var anchoredDetail *billing.AnchoredAlignmentDetail
	if workflow.Collection.Alignment != nil {
		sub, err := workflow.Collection.Alignment.AsBillingWorkflowCollectionAlignmentSubscription()
		if err == nil && sub.Type == "subscription" {
			alignment = billing.AlignmentKindSubscription
		}

		anchored, err := workflow.Collection.Alignment.AsBillingWorkflowCollectionAlignmentAnchored()
		if err == nil && anchored.Type == "anchored" {
			alignment = billing.AlignmentKindAnchored
			if parsed, err := period.Parse(anchored.RecurringPeriod.Interval); err == nil {
				anchoredDetail = &billing.AnchoredAlignmentDetail{
					Interval: datetime.ISODuration{Period: parsed},
					Anchor:   anchored.RecurringPeriod.Anchor,
				}
			}
		}
	}

	// Parse invoicing draft period with default fallback
	draftPeriod := def.Invoicing.DraftPeriod
	if workflow.Invoicing.DraftPeriod != nil {
		if parsed, err := period.Parse(*workflow.Invoicing.DraftPeriod); err == nil {
			draftPeriod = datetime.ISODuration{Period: parsed}
		}
	}

	// Parse invoicing due after with default fallback
	dueAfter := def.Invoicing.DueAfter
	if workflow.Payment != nil {
		sendInvoice, err := workflow.Payment.AsBillingWorkflowPaymentSendInvoiceSettings()
		if err == nil && sendInvoice.CollectionMethod == "send_invoice" && sendInvoice.DueAfter != nil {
			if parsed, err := period.Parse(*sendInvoice.DueAfter); err == nil {
				dueAfter = datetime.ISODuration{Period: parsed}
			}
		}
	}

	// Parse payment collection method with default fallback
	collectionMethod := def.Payment.CollectionMethod
	if workflow.Payment != nil {
		chargeAuto, err := workflow.Payment.AsBillingWorkflowPaymentChargeAutomaticallySettings()
		if err == nil && chargeAuto.CollectionMethod == "charge_automatically" {
			collectionMethod = billing.CollectionMethodChargeAutomatically
		}

		sendInvoice, err := workflow.Payment.AsBillingWorkflowPaymentSendInvoiceSettings()
		if err == nil && sendInvoice.CollectionMethod == "send_invoice" {
			collectionMethod = billing.CollectionMethodSendInvoice
		}
	}

	defaultTaxConfig := def.Invoicing.DefaultTaxConfig
	if workflow.Tax.DefaultTaxConfig != nil {
		var err error
		defaultTaxConfig, err = FromAPIBillingTaxConfig(workflow.Tax.DefaultTaxConfig)
		if err != nil {
			return billing.WorkflowConfig{}, err
		}
	}

	return billing.WorkflowConfig{
		Collection: billing.CollectionConfig{
			Alignment:               alignment,
			AnchoredAlignmentDetail: anchoredDetail,
			Interval:                collInterval,
		},
		Invoicing: billing.InvoicingConfig{
			AutoAdvance:        lo.FromPtrOr(workflow.Invoicing.AutoAdvance, def.Invoicing.AutoAdvance),
			DraftPeriod:        draftPeriod,
			DueAfter:           dueAfter,
			ProgressiveBilling: lo.FromPtrOr(workflow.Invoicing.ProgressiveBilling, def.Invoicing.ProgressiveBilling),
			DefaultTaxConfig:   defaultTaxConfig,
		},
		Payment: billing.PaymentConfig{
			CollectionMethod: collectionMethod,
		},
		Tax: billing.WorkflowTaxConfig{
			Enabled:  lo.FromPtrOr(workflow.Tax.Enabled, def.Tax.Enabled),
			Enforced: lo.FromPtrOr(workflow.Tax.Enforced, def.Tax.Enforced),
		},
	}, nil
}
