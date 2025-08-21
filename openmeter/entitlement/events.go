package entitlement

import (
	"errors"
	"time"

	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	eventmodels "github.com/openmeterio/openmeter/openmeter/event/models"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
	pkgmodels "github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

const (
	EventSubsystem metadata.EventSubsystem = "entitlement"
)

type entitlementEvent struct {
	// Core identifiers
	ID        string                  `json:"id"`
	Namespace eventmodels.NamespaceID `json:"namespace"`

	// Subject linkage (current v1 behavior)
	SubjectKey string `json:"subjectKey"`

	// Feature linkage
	FeatureID  string `json:"featureId"`
	FeatureKey string `json:"featureKey"`

	// Managed model
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`

	// Metadata and annotations
	Metadata    map[string]string     `json:"metadata,omitempty"`
	Annotations pkgmodels.Annotations `json:"annotations,omitempty"`

	// Type and scheduling
	EntitlementType EntitlementType `json:"type,omitempty"`
	ActiveFrom      *time.Time      `json:"activeFrom,omitempty"`
	ActiveTo        *time.Time      `json:"activeTo,omitempty"`

	// Period fields needed by consumers
	CurrentUsagePeriod *timeutil.ClosedPeriod `json:"currentUsagePeriod,omitempty"`
}

func (e entitlementEvent) Validate() error {
	if e.ID == "" {
		return errors.New("ID must not be empty")
	}

	if e.SubjectKey == "" {
		return errors.New("subjectKey must not be empty")
	}

	if err := e.Namespace.Validate(); err != nil {
		return err
	}

	return nil
}

// Mapping helpers
func mapEntitlementToEventPayload(ent Entitlement) entitlementEvent {
	return entitlementEvent{
		ID:                 ent.ID,
		Namespace:          eventmodels.NamespaceID{ID: ent.Namespace},
		SubjectKey:         ent.SubjectKey,
		FeatureID:          ent.FeatureID,
		FeatureKey:         ent.FeatureKey,
		CreatedAt:          ent.CreatedAt,
		UpdatedAt:          ent.UpdatedAt,
		DeletedAt:          ent.DeletedAt,
		Metadata:           ent.Metadata,
		Annotations:        ent.Annotations,
		EntitlementType:    ent.EntitlementType,
		ActiveFrom:         ent.ActiveFrom,
		ActiveTo:           ent.ActiveTo,
		CurrentUsagePeriod: ent.CurrentUsagePeriod,
	}
}

func (e entitlementEvent) ToDomainEntitlement() Entitlement {
	return Entitlement{
		GenericProperties: GenericProperties{
			NamespacedModel:    pkgmodels.NamespacedModel{Namespace: e.Namespace.ID},
			ManagedModel:       pkgmodels.ManagedModel{CreatedAt: e.CreatedAt, UpdatedAt: e.UpdatedAt, DeletedAt: e.DeletedAt},
			MetadataModel:      pkgmodels.MetadataModel{Metadata: e.Metadata},
			Annotations:        e.Annotations,
			ID:                 e.ID,
			FeatureID:          e.FeatureID,
			FeatureKey:         e.FeatureKey,
			SubjectKey:         e.SubjectKey,
			EntitlementType:    e.EntitlementType,
			ActiveFrom:         e.ActiveFrom,
			ActiveTo:           e.ActiveTo,
			CurrentUsagePeriod: e.CurrentUsagePeriod,
		},
	}
}

// Exported helper for external packages
func (e EntitlementDeletedEvent) ToDomainEntitlement() Entitlement {
	return entitlementEvent(e).ToDomainEntitlement()
}

type EntitlementCreatedEvent entitlementEvent

var (
	_ marshaler.Event = EntitlementCreatedEvent{}

	entitlementCreatedEventName = metadata.GetEventName(metadata.EventType{
		Subsystem: EventSubsystem,
		Name:      "entitlement.created",
		Version:   "v1",
	})
)

func (e EntitlementCreatedEvent) Validate() error {
	return entitlementEvent(e).Validate()
}

func (e EntitlementCreatedEvent) EventName() string {
	return entitlementCreatedEventName
}

func (e EntitlementCreatedEvent) EventMetadata() metadata.EventMetadata {
	return metadata.EventMetadata{
		Source:  metadata.ComposeResourcePath(e.Namespace.ID, metadata.EntityEntitlement, e.ID),
		Subject: metadata.ComposeResourcePath(e.Namespace.ID, metadata.EntitySubjectKey, e.SubjectKey),
	}
}

// Helper constructor
func NewEntitlementCreatedEventPayload(ent Entitlement) EntitlementCreatedEvent {
	return EntitlementCreatedEvent(mapEntitlementToEventPayload(ent))
}

type EntitlementDeletedEvent entitlementEvent

var (
	_ marshaler.Event = EntitlementDeletedEvent{}

	entitlementDeletedEventName = metadata.GetEventName(metadata.EventType{
		Subsystem: EventSubsystem,
		Name:      "entitlement.deleted",
		Version:   "v1",
	})
)

func (e EntitlementDeletedEvent) Validate() error {
	return entitlementEvent(e).Validate()
}

func (e EntitlementDeletedEvent) EventName() string {
	return entitlementDeletedEventName
}

func (e EntitlementDeletedEvent) EventMetadata() metadata.EventMetadata {
	return metadata.EventMetadata{
		Source:  metadata.ComposeResourcePath(e.Namespace.ID, metadata.EntityEntitlement, e.ID),
		Subject: metadata.ComposeResourcePath(e.Namespace.ID, metadata.EntitySubjectKey, e.SubjectKey),
	}
}

// Helper constructor
func NewEntitlementDeletedEventPayload(ent Entitlement) EntitlementDeletedEvent {
	return EntitlementDeletedEvent(mapEntitlementToEventPayload(ent))
}
