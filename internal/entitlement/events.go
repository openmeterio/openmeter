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
	EventCreateEntitlement spec.EventName = "createEntitlement"
	EventDeleteEntitlement spec.EventName = "deleteEntitlement"
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
	Subsystem:   EventSubsystem,
	Name:        EventCreateEntitlement,
	SpecVersion: "1.0",
	Version:     "v1",
}

func (e EntitlementCreatedEvent) Spec() *spec.EventTypeSpec {
	return &entitlementCreatedEventSpec
}

func (e EntitlementCreatedEvent) Validate() error {
	return entitlementEvent(e).Validate()
}

type EntitlementDeletedEvent entitlementEvent

var entitlementDeletedEventSpec = spec.EventTypeSpec{
	Subsystem:   EventSubsystem,
	Name:        EventDeleteEntitlement,
	SpecVersion: "1.0",
	Version:     "v1",
}

func (e EntitlementDeletedEvent) Spec() *spec.EventTypeSpec {
	return &entitlementDeletedEventSpec
}

func (e EntitlementDeletedEvent) Validate() error {
	return entitlementEvent(e).Validate()
}
