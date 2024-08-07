package entitlement

import (
	"errors"

	"github.com/openmeterio/openmeter/internal/event/metadata"
	"github.com/openmeterio/openmeter/internal/event/models"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
)

const (
	EventSubsystem metadata.EventSubsystem = "entitlement"
)

type entitlementEvent struct {
	Entitlement
	Namespace models.NamespaceID `json:"namespace"`
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
