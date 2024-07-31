package entitlement

import (
	"errors"

	"github.com/openmeterio/openmeter/internal/event/models"
	"github.com/openmeterio/openmeter/internal/event/spec"
)

const (
	EventSubsystem spec.EventSubsystem = "entitlement"
)

const (
	entitlementCreatedEventName spec.EventName = "entitlement.created"
	entitlementDeletedEventName spec.EventName = "entitlement.deleted"
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

var entitlementCreatedEventSpec = spec.EventTypeSpec{
	Subsystem: EventSubsystem,
	Name:      entitlementCreatedEventName,
	Version:   "v1",
}

func (e EntitlementCreatedEvent) Spec() *spec.EventTypeSpec {
	return &entitlementCreatedEventSpec
}

func (e EntitlementCreatedEvent) Validate() error {
	return entitlementEvent(e).Validate()
}

func (e EntitlementCreatedEvent) EventName() string {
	return e.Spec().Type()
}

func (e EntitlementCreatedEvent) EventMetadata() spec.EventMetadata {
	return spec.EventMetadata{
		Source:  spec.ComposeResourcePath(e.Namespace.ID, spec.EntityEntitlement, e.ID),
		Subject: spec.ComposeResourcePath(e.Namespace.ID, spec.EntitySubjectKey, e.SubjectKey),
	}
}

type EntitlementDeletedEvent entitlementEvent

var entitlementDeletedEventSpec = spec.EventTypeSpec{
	Subsystem: EventSubsystem,
	Name:      entitlementDeletedEventName,
	Version:   "v1",
}

func (e EntitlementDeletedEvent) Spec() *spec.EventTypeSpec {
	return &entitlementDeletedEventSpec
}

func (e EntitlementDeletedEvent) Validate() error {
	return entitlementEvent(e).Validate()
}

func (e EntitlementDeletedEvent) EventName() string {
	return e.Spec().Type()
}

func (e EntitlementDeletedEvent) EventMetadata() spec.EventMetadata {
	return spec.EventMetadata{
		Source:  spec.ComposeResourcePath(e.Namespace.ID, spec.EntityEntitlement, e.ID),
		Subject: spec.ComposeResourcePath(e.Namespace.ID, spec.EntitySubjectKey, e.SubjectKey),
	}
}
