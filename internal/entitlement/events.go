package entitlement

import (
	"errors"

	"github.com/openmeterio/openmeter/internal/event/models"
	"github.com/openmeterio/openmeter/internal/event/spec"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
)

const (
	EventSubsystem spec.EventSubsystem = "entitlement"
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

	entitlementCreatedEventName = spec.GetEventName(spec.EventTypeSpec{
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

func (e EntitlementCreatedEvent) EventMetadata() spec.EventMetadata {
	return spec.EventMetadata{
		Source:  spec.ComposeResourcePath(e.Namespace.ID, spec.EntityEntitlement, e.ID),
		Subject: spec.ComposeResourcePath(e.Namespace.ID, spec.EntitySubjectKey, e.SubjectKey),
	}
}

type EntitlementDeletedEvent entitlementEvent

var (
	_ marshaler.Event = EntitlementDeletedEvent{}

	entitlementDeletedEventName = spec.GetEventName(spec.EventTypeSpec{
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

func (e EntitlementDeletedEvent) EventMetadata() spec.EventMetadata {
	return spec.EventMetadata{
		Source:  spec.ComposeResourcePath(e.Namespace.ID, spec.EntityEntitlement, e.ID),
		Subject: spec.ComposeResourcePath(e.Namespace.ID, spec.EntitySubjectKey, e.SubjectKey),
	}
}
