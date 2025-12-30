package entitlement

import (
	"time"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	eventmodels "github.com/openmeterio/openmeter/openmeter/event/models"
	"github.com/openmeterio/openmeter/openmeter/subject"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

const (
	EventSubsystem metadata.EventSubsystem = "entitlement"
)

// Literal types for versioning
type entitlementEventV2 struct {
	Namespace   eventmodels.NamespaceID              `json:"namespace"`
	Entitlement entitlementEventV2EntitlementLiteral `json:"entitlement"`
}

func (e entitlementEventV2) Validate() error {
	if err := e.Namespace.Validate(); err != nil {
		return err
	}

	if err := e.Entitlement.Validate(); err != nil {
		return err
	}
	return nil
}

// To make versioning possible, we'll type out the entitlement literal fields to the best of our ability.
// The ideal solution would still be to version the domain models but that lift is too large for now.
type entitlementEventV2EntitlementLiteral struct {
	// - Generic Properties
	NamespacedModel models.NamespacedModel `json:"namespacedModel"`
	ManagedModel    models.ManagedModel    `json:"managedModel"`
	MetadataModel   models.MetadataModel   `json:"metadataModel"`
	Annotations     models.Annotations     `json:"annotations"`

	ActiveFrom *time.Time `json:"activeFrom,omitempty"`
	ActiveTo   *time.Time `json:"activeTo,omitempty"`

	ID         string `json:"id,omitempty"`
	FeatureID  string `json:"featureId,omitempty"`
	FeatureKey string `json:"featureKey,omitempty"`

	SubjectKey string             `json:"subjectKey,omitempty"`
	Subject    subject.Subject    `json:"subject,omitempty"`
	Customer   *customer.Customer `json:"customer,omitempty"`

	EntitlementType           EntitlementType        `json:"type,omitempty"`
	UsagePeriod               *UsagePeriod           `json:"usagePeriod,omitempty"`
	CurrentUsagePeriod        *timeutil.ClosedPeriod `json:"currentUsagePeriod,omitempty"`
	OriginalUsagePeriodAnchor *time.Time             `json:"originalUsagePeriodAnchor,omitempty"`

	// - Non-Generic Properties
	MeasureUsageFrom        *time.Time `json:"measureUsageFrom,omitempty"`
	IssueAfterReset         *float64   `json:"issueAfterReset,omitempty"`
	IssueAfterResetPriority *uint8     `json:"issueAfterResetPriority,omitempty"`
	IsSoftLimit             *bool      `json:"isSoftLimit,omitempty"`
	LastReset               *time.Time `json:"lastReset,omitempty"`
	PreserveOverageAtReset  *bool      `json:"preserveOverageAtReset,omitempty"`

	// static
	Config *string `json:"config,omitempty"`
}

func (e entitlementEventV2EntitlementLiteral) ToDomainEntitlement() Entitlement {
	return Entitlement{
		GenericProperties: GenericProperties{
			NamespacedModel:           e.NamespacedModel,
			ManagedModel:              e.ManagedModel,
			MetadataModel:             e.MetadataModel,
			Annotations:               e.Annotations,
			ID:                        e.ID,
			FeatureID:                 e.FeatureID,
			FeatureKey:                e.FeatureKey,
			Customer:                  e.Customer,
			EntitlementType:           e.EntitlementType,
			UsagePeriod:               e.UsagePeriod,
			CurrentUsagePeriod:        e.CurrentUsagePeriod,
			OriginalUsagePeriodAnchor: e.OriginalUsagePeriodAnchor,
			ActiveFrom:                e.ActiveFrom,
			ActiveTo:                  e.ActiveTo,
		},
		MeasureUsageFrom:        e.MeasureUsageFrom,
		IssueAfterReset:         e.IssueAfterReset,
		IssueAfterResetPriority: e.IssueAfterResetPriority,
		IsSoftLimit:             e.IsSoftLimit,
		LastReset:               e.LastReset,
		PreserveOverageAtReset:  e.PreserveOverageAtReset,
		Config:                  e.Config,
	}
}

func (e entitlementEventV2EntitlementLiteral) Validate() error {
	domainEnt := e.ToDomainEntitlement()

	if err := domainEnt.Validate(); err != nil {
		return err
	}

	return nil
}

func mapEntitlementToV2Literal(ent Entitlement) entitlementEventV2EntitlementLiteral {
	return entitlementEventV2EntitlementLiteral{
		NamespacedModel:           ent.NamespacedModel,
		ManagedModel:              ent.ManagedModel,
		MetadataModel:             ent.MetadataModel,
		Annotations:               ent.Annotations,
		ActiveFrom:                ent.ActiveFrom,
		ActiveTo:                  ent.ActiveTo,
		ID:                        ent.ID,
		FeatureID:                 ent.FeatureID,
		FeatureKey:                ent.FeatureKey,
		Customer:                  ent.Customer,
		EntitlementType:           ent.EntitlementType,
		UsagePeriod:               ent.UsagePeriod,
		CurrentUsagePeriod:        ent.CurrentUsagePeriod,
		OriginalUsagePeriodAnchor: ent.OriginalUsagePeriodAnchor,
		MeasureUsageFrom:          ent.MeasureUsageFrom,
		IssueAfterReset:           ent.IssueAfterReset,
		IssueAfterResetPriority:   ent.IssueAfterResetPriority,
		IsSoftLimit:               ent.IsSoftLimit,
		LastReset:                 ent.LastReset,
		PreserveOverageAtReset:    ent.PreserveOverageAtReset,
		Config:                    ent.Config,
	}
}

func mapEntitlementToV2(ent Entitlement) entitlementEventV2 {
	return entitlementEventV2{
		Namespace:   eventmodels.NamespaceID{ID: ent.Namespace},
		Entitlement: mapEntitlementToV2Literal(ent),
	}
}

// Events

type EntitlementCreatedEventV2 entitlementEventV2

var (
	_ marshaler.Event = EntitlementCreatedEventV2{}

	entitlementCreatedEventV2Name = metadata.GetEventName(metadata.EventType{
		Subsystem: EventSubsystem,
		Name:      "entitlement.created",
		Version:   "v2",
	})
)

func (e EntitlementCreatedEventV2) Validate() error {
	return entitlementEventV2(e).Validate()
}

func (e EntitlementCreatedEventV2) EventName() string {
	return entitlementCreatedEventV2Name
}

func (e EntitlementCreatedEventV2) EventMetadata() metadata.EventMetadata {
	return metadata.EventMetadata{
		Source:  metadata.ComposeResourcePath(e.Namespace.ID, metadata.EntityEntitlement, e.Entitlement.ID),
		Subject: metadata.ComposeResourcePath(e.Namespace.ID, metadata.EntityCustomer, e.Entitlement.Customer.ID),
	}
}

func NewEntitlementCreatedEventPayloadV2(ent Entitlement) EntitlementCreatedEventV2 {
	return EntitlementCreatedEventV2(mapEntitlementToV2(ent))
}

type EntitlementDeletedEventV2 entitlementEventV2

var (
	_ marshaler.Event = EntitlementDeletedEventV2{}

	entitlementDeletedEventV2Name = metadata.GetEventName(metadata.EventType{
		Subsystem: EventSubsystem,
		Name:      "entitlement.deleted",
		Version:   "v2",
	})
)

func (e EntitlementDeletedEventV2) Validate() error {
	return entitlementEventV2(e).Validate()
}

func (e EntitlementDeletedEventV2) EventName() string {
	return entitlementDeletedEventV2Name
}

func (e EntitlementDeletedEventV2) EventMetadata() metadata.EventMetadata {
	return metadata.EventMetadata{
		Source:  metadata.ComposeResourcePath(e.Namespace.ID, metadata.EntityEntitlement, e.Entitlement.ID),
		Subject: metadata.ComposeResourcePath(e.Namespace.ID, metadata.EntityCustomer, e.Entitlement.Customer.ID),
	}
}

func NewEntitlementDeletedEventPayloadV2(ent Entitlement) EntitlementDeletedEventV2 {
	return EntitlementDeletedEventV2(mapEntitlementToV2(ent))
}
