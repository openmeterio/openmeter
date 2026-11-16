//go:generate go run github.com/jmattheis/goverter/cmd/goverter gen ./
package billingprofiles

import (
	"github.com/rickb777/period"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api/v3/labels"
	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
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
	ToAPIAddress                     func(address models.Address) apiv3models.Address
	FromAPIAddress                   func(address apiv3models.Address) models.Address
	ToAPIBillingProfileAppReferences func(refs *billing.ProfileAppReferences) apiv3models.BillingProfileAppReferences
	// goverter:context namespace
	FromAPIBillingProfileAppReferences func(namespace string, refs apiv3models.BillingProfileAppReferences) billing.ProfileAppReferences
	// goverter:context namespace
	// goverter:map Namespace | NamespaceFromContext
	FromAPIBillingAppReferenceToAppID func(namespace string, ref apiv3models.BillingAppReference) app.AppID
	ToAPIBillingAppReferenceFromAppID func(appID app.AppID) apiv3models.BillingAppReference
	// goverter:autoMap BaseProfile
	// goverter:map BaseProfile.Metadata Labels
	// goverter:map BaseProfile.WorkflowConfig Workflow
	ToAPIBillingProfile  func(profile billing.Profile) (apiv3models.BillingProfile, error)
	ToAPIBillingProfiles func(profiles []billing.Profile) ([]apiv3models.BillingProfile, error)
	// goverter:context namespace
	// goverter:map Namespace | NamespaceFromContext
	// goverter:map Labels Metadata | ConvertLabelsToMetadata
	// goverter:map Workflow WorkflowConfig | FromAPIBillingWorkflowCreate
	FromAPICreateBillingProfileRequest func(namespace string, req apiv3models.CreateBillingProfileRequest) (billing.CreateProfileInput, error)
	// goverter:context namespacedID
	// goverter:map Namespace | ResolveNamespaceFromContext
	// goverter:map ID | ResolveIDFromContext
	// goverter:map Workflow WorkflowConfig | FromAPIBillingWorkflow
	// goverter:map Labels Metadata
	// goverter:ignore CreatedAt
	// goverter:ignore UpdatedAt
	// goverter:ignore DeletedAt
	// goverter:ignore AppReferences
	FromAPIUpsertBillingProfileRequest func(namespacedID models.NamespacedID, req apiv3models.UpsertBillingProfileRequest) (billing.UpdateProfileInput, error)
	// goverter:enum:unknown @error
	// goverter:enum:map InclusiveTaxBehavior BillingTaxBehaviorInclusive
	// goverter:enum:map ExclusiveTaxBehavior BillingTaxBehaviorExclusive
	ToAPIBillingTaxBehavior func(behavior productcatalog.TaxBehavior) (apiv3models.BillingTaxBehavior, error)
	// goverter:enum:unknown @error
	// goverter:enum:map BillingTaxBehaviorInclusive InclusiveTaxBehavior
	// goverter:enum:map BillingTaxBehaviorExclusive ExclusiveTaxBehavior
	FromAPIBillingTaxBehavior func(behavior apiv3models.BillingTaxBehavior) (productcatalog.TaxBehavior, error)
	// goverter:map Stripe ExternalInvoicing
	// goverter:map Stripe Stripe
	// goverter:ignore TaxCode
	ToAPIBillingTaxConfig func(config *productcatalog.TaxConfig) (*apiv3models.BillingTaxConfig, error)
	// goverter:map Stripe Stripe
	FromAPIBillingTaxConfig func(config *apiv3models.BillingTaxConfig) (*productcatalog.TaxConfig, error)
)

var (
	ConvertMetadataToLabels = labels.FromMetadata[billing.Metadata]
	ConvertLabelsToMetadata = func(l *apiv3models.Labels) (billing.Metadata, error) {
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

// toAPIBillingTaxConfigFull wraps the goverter ToAPIBillingTaxConfig and also populates
// TaxCode from TaxCodeId so the new and deprecated fields are both set on responses.
func toAPIBillingTaxConfigFull(config *productcatalog.TaxConfig) (*apiv3models.BillingTaxConfig, error) {
	result, err := ToAPIBillingTaxConfig(config)
	if err != nil || result == nil {
		return result, err
	}

	if result.TaxCodeId != nil {
		result.TaxCode = &apiv3models.TaxCodeReference{Id: *result.TaxCodeId}
	}

	return result, nil
}

// FromAPIBillingTaxConfigFull wraps the goverter FromAPIBillingTaxConfig and additionally reads
// TaxCode.Id (new field) which takes precedence over the deprecated TaxCodeId.
func FromAPIBillingTaxConfigFull(config *apiv3models.BillingTaxConfig) (*productcatalog.TaxConfig, error) {
	result, err := FromAPIBillingTaxConfig(config)
	if err != nil || result == nil {
		return result, err
	}

	if config.TaxCode != nil {
		result.TaxCodeID = &config.TaxCode.Id
	}

	return result, nil
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
func ToAPIBillingAppReference(app app.App) apiv3models.BillingAppReference {
	return apiv3models.BillingAppReference{
		Id: app.GetID().ID,
	}
}

// ToAPIBillingParty converts billing.SupplierContact to API BillingParty
func ToAPIBillingParty(supplier billing.SupplierContact) apiv3models.BillingParty {
	party := apiv3models.BillingParty{
		Id:   &supplier.ID,
		Name: &supplier.Name,
	}

	// Emit the billing address whenever any address field is present. Gating on
	// Country alone would drop line1/city/state/phone-only addresses from
	// supplier snapshots, since every address field is optional in the API shape.
	addr := supplier.Address
	if addr.Country != nil || addr.PostalCode != nil || addr.State != nil ||
		addr.City != nil || addr.Line1 != nil || addr.Line2 != nil || addr.PhoneNumber != nil {
		party.Addresses = &apiv3models.BillingPartyAddresses{
			BillingAddress: ToAPIAddress(supplier.Address),
		}
	}

	if supplier.TaxCode != nil {
		party.TaxId = &apiv3models.BillingPartyTaxIdentity{
			Code: supplier.TaxCode,
		}
	}

	return party
}

// ToAPIBillingSupplier converts billing.SupplierContact to the invoice's
// BillingSupplier snapshot, a read-only subset of BillingParty.
func ToAPIBillingSupplier(supplier billing.SupplierContact) apiv3models.BillingSupplier {
	party := ToAPIBillingParty(supplier)

	return apiv3models.BillingSupplier{
		Id:        party.Id,
		Name:      party.Name,
		TaxId:     party.TaxId,
		Addresses: party.Addresses,
	}
}

// FromAPIBillingParty converts API BillingParty to billing.SupplierContact
func FromAPIBillingParty(party apiv3models.BillingParty) billing.SupplierContact {
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
func ToAPIBillingWorkflow(config billing.WorkflowConfig) (apiv3models.BillingWorkflow, error) {
	workflow := apiv3models.BillingWorkflow{}

	// Collection settings
	workflow.Collection = &apiv3models.BillingWorkflowCollectionSettings{
		Interval: lo.ToPtr(config.Collection.Interval.String()),
	}

	// Convert alignment
	switch config.Collection.Alignment {
	case billing.AlignmentKindSubscription:
		alignment := apiv3models.BillingWorkflowCollectionAlignment{}
		err := alignment.FromBillingWorkflowCollectionAlignmentSubscription(apiv3models.BillingWorkflowCollectionAlignmentSubscription{
			Type: "subscription",
		})
		if err != nil {
			return apiv3models.BillingWorkflow{}, err
		}

		workflow.Collection.Alignment = &alignment
	case billing.AlignmentKindAnchored:
		if config.Collection.AnchoredAlignmentDetail != nil {
			alignment := apiv3models.BillingWorkflowCollectionAlignment{}
			err := alignment.FromBillingWorkflowCollectionAlignmentAnchored(apiv3models.BillingWorkflowCollectionAlignmentAnchored{
				Type: "anchored",
				RecurringPeriod: apiv3models.RecurringPeriod{
					Anchor:   config.Collection.AnchoredAlignmentDetail.Anchor,
					Interval: config.Collection.AnchoredAlignmentDetail.Interval.String(),
				},
			})
			if err != nil {
				return apiv3models.BillingWorkflow{}, err
			}

			workflow.Collection.Alignment = &alignment
		}
	}

	// Invoicing settings
	workflow.Invoicing = &apiv3models.BillingWorkflowInvoicingSettings{
		AutoAdvance:                  lo.ToPtr(config.Invoicing.AutoAdvance),
		DraftPeriod:                  lo.ToPtr(config.Invoicing.DraftPeriod.String()),
		ProgressiveBilling:           lo.ToPtr(config.Invoicing.ProgressiveBilling),
		SubscriptionEndProrationMode: lo.ToPtr(apiv3models.BillingWorkflowInvoicingSubscriptionEndProrationMode(config.Invoicing.SubscriptionEndProrationMode)),
	}

	// Tax settings
	defaultTaxConfig, err := toAPIBillingTaxConfigFull(config.Invoicing.DefaultTaxConfig)
	if err != nil {
		return apiv3models.BillingWorkflow{}, err
	}

	workflow.Tax = &apiv3models.BillingWorkflowTaxSettings{
		Enabled:          lo.ToPtr(config.Tax.Enabled),
		Enforced:         lo.ToPtr(config.Tax.Enforced),
		DefaultTaxConfig: defaultTaxConfig,
	}

	// Payment settings
	switch config.Payment.CollectionMethod {
	case billing.CollectionMethodChargeAutomatically:
		payment := apiv3models.BillingWorkflowPaymentSettings{}
		err := payment.FromBillingWorkflowPaymentChargeAutomaticallySettings(apiv3models.BillingWorkflowPaymentChargeAutomaticallySettings{
			CollectionMethod: "charge_automatically",
		})
		if err != nil {
			return apiv3models.BillingWorkflow{}, err
		}

		workflow.Payment = &payment
	case billing.CollectionMethodSendInvoice:
		payment := apiv3models.BillingWorkflowPaymentSettings{}
		err := payment.FromBillingWorkflowPaymentSendInvoiceSettings(apiv3models.BillingWorkflowPaymentSendInvoiceSettings{
			CollectionMethod: "send_invoice",
			DueAfter:         lo.ToPtr(config.Invoicing.DueAfter.String()),
		})
		if err != nil {
			return apiv3models.BillingWorkflow{}, err
		}

		workflow.Payment = &payment
	}

	return workflow, nil
}

// FromAPIBillingWorkflow converts API BillingWorkflow to billing.WorkflowConfig
func FromAPIBillingWorkflow(workflow apiv3models.BillingWorkflow) (billing.WorkflowConfig, error) {
	return fromAPIBillingWorkflow(workflow, "")
}

// FromAPIBillingWorkflowCreate converts API BillingWorkflow to billing.WorkflowConfig for profile creation.
func FromAPIBillingWorkflowCreate(workflow apiv3models.BillingWorkflow) (billing.WorkflowConfig, error) {
	return fromAPIBillingWorkflow(workflow, billing.DefaultWorkflowConfig.Invoicing.SubscriptionEndProrationMode)
}

func fromAPIBillingWorkflow(workflow apiv3models.BillingWorkflow, defaultSubscriptionEndProrationMode billing.SubscriptionEndProrationMode) (billing.WorkflowConfig, error) {
	// Start with default configuration
	def := billing.DefaultWorkflowConfig

	// Ensure workflow sections are initialized
	if workflow.Collection == nil {
		workflow.Collection = &apiv3models.BillingWorkflowCollectionSettings{}
	}

	if workflow.Invoicing == nil {
		workflow.Invoicing = &apiv3models.BillingWorkflowInvoicingSettings{}
	}

	if workflow.Payment == nil {
		workflow.Payment = &apiv3models.BillingWorkflowPaymentSettings{}
	}

	if workflow.Tax == nil {
		workflow.Tax = &apiv3models.BillingWorkflowTaxSettings{}
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
		defaultTaxConfig, err = FromAPIBillingTaxConfigFull(workflow.Tax.DefaultTaxConfig)
		if err != nil {
			return billing.WorkflowConfig{}, err
		}
	}

	subscriptionEndProrationMode := defaultSubscriptionEndProrationMode
	if workflow.Invoicing.SubscriptionEndProrationMode != nil {
		subscriptionEndProrationMode = billing.SubscriptionEndProrationMode(*workflow.Invoicing.SubscriptionEndProrationMode)
	}

	return billing.WorkflowConfig{
		Collection: billing.CollectionConfig{
			Alignment:               alignment,
			AnchoredAlignmentDetail: anchoredDetail,
			Interval:                collInterval,
		},
		Invoicing: billing.InvoicingConfig{
			AutoAdvance:                  lo.FromPtrOr(workflow.Invoicing.AutoAdvance, def.Invoicing.AutoAdvance),
			DraftPeriod:                  draftPeriod,
			DueAfter:                     dueAfter,
			ProgressiveBilling:           lo.FromPtrOr(workflow.Invoicing.ProgressiveBilling, def.Invoicing.ProgressiveBilling),
			SubscriptionEndProrationMode: subscriptionEndProrationMode,
			DefaultTaxConfig:             defaultTaxConfig,
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
