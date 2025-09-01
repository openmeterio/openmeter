package entitlement

import (
	"errors"

	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	eventmodels "github.com/openmeterio/openmeter/openmeter/event/models"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
)

const (
	EventSubsystem metadata.EventSubsystem = "entitlement"
)

// Events based on entitlementEventV1 should slowly be removed. The issue with this old pattern is that domain models
// are embedded inside the event, which means that domain changes break previous events.
// Future versions (starting with v2) will declare the event using only primitives.
// Deprecated: use events_v2.go instead
type entitlementEventV1 struct {
	Entitlement
	Namespace eventmodels.NamespaceID `json:"namespace"`
}

func (e entitlementEventV1) Validate() error {
	if e.ID == "" {
		return errors.New("ID must not be empty")
	}

	if err := e.Subject.Validate(); err != nil {
		return err
	}

	if err := e.Customer.Validate(); err != nil {
		return err
	}

	if err := e.Namespace.Validate(); err != nil {
		return err
	}

	return nil
}

func (e entitlementEventV1) ToDomainEntitlement() Entitlement {
	return e.Entitlement
}

func (e EntitlementDeletedEvent) ToDomainEntitlement() Entitlement {
	return entitlementEventV1(e).ToDomainEntitlement()
}

// Deprecated: use EntitlementCreatedEventV2 instead
type EntitlementCreatedEvent entitlementEventV1

var (
	_ marshaler.Event = EntitlementCreatedEvent{}

	entitlementCreatedEventName = metadata.GetEventName(metadata.EventType{
		Subsystem: EventSubsystem,
		Name:      "entitlement.created",
		Version:   "v1",
	})
)

func (e EntitlementCreatedEvent) Validate() error {
	return entitlementEventV1(e).Validate()
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

// Deprecated: use EntitlementDeletedEventV2 instead
type EntitlementDeletedEvent entitlementEventV1

var (
	_ marshaler.Event = EntitlementDeletedEvent{}

	entitlementDeletedEventName = metadata.GetEventName(metadata.EventType{
		Subsystem: EventSubsystem,
		Name:      "entitlement.deleted",
		Version:   "v1",
	})
)

func (e EntitlementDeletedEvent) Validate() error {
	return entitlementEventV1(e).Validate()
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
