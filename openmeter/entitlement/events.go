package entitlement

import "github.com/openmeterio/openmeter/internal/entitlement"

const (
	EventSubsystem = entitlement.EventSubsystem
)

const (
	EventCreateEntitlement = entitlement.EventCreateEntitlement
	EventDeleteEntitlement = entitlement.EventDeleteEntitlement
)

type (
	EntitlementCreatedEvent = entitlement.EntitlementCreatedEvent
	EntitlementDeletedEvent = entitlement.EntitlementDeletedEvent
)
