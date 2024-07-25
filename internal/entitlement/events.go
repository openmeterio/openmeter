package entitlement

import (
	"github.com/openmeterio/openmeter/internal/event/spec"
)

const (
	EventSubsystem spec.EventSubsystem = "entitlement"

	EventCreateEntitlement spec.EventName = "createEntitlement"
	EventDeleteEntitlement spec.EventName = "deleteEntitlement"
)

type entitlementEvent struct {
	Entitlement
	Namespace string `json:"namespace"`
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
