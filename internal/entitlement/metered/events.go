package meteredentitlement

import (
	"time"

	"github.com/openmeterio/openmeter/internal/event/spec"
)

const (
	EventSubsystemMeteredEntitlement spec.EventSubsystem = "meteredEntitlement"

	EventResetEntitlementUsage         spec.EventName = "resetEntitlementUsage"
	EventCreateMeteredEntitlementGrant spec.EventName = "createMeteredEntitlementGrant"
)

type ResetEntitlementEvent struct {
	EntitlementID string    `json:"entitlement_id"`
	Namespace     string    `json:"namespace"`
	SubjectKey    string    `json:"subjectKey"`
	ResetAt       time.Time `json:"resetAt"`
	RetainAnchor  bool      `json:"retainAnchor"`
}

var resetEntitlementEventSpec = spec.EventTypeSpec{
	Subsystem:   EventSubsystemMeteredEntitlement,
	Name:        EventResetEntitlementUsage,
	SpecVersion: "1.0",
	Version:     "v1",
}

func (e ResetEntitlementEvent) Spec() *spec.EventTypeSpec {
	return &resetEntitlementEventSpec
}

type CreateMeteredEntitlementGrantEvent struct {
	EntitlementGrant
	SubjectKey string `json:"subjectKey"`
	Namespace  string `json:"namespace"`
}

var createMeteredEntitlementGrantEventSpec = spec.EventTypeSpec{
	Subsystem:   EventSubsystemMeteredEntitlement,
	Name:        EventCreateMeteredEntitlementGrant,
	SpecVersion: "1.0",
	Version:     "v1",
}

func (e CreateMeteredEntitlementGrantEvent) Spec() *spec.EventTypeSpec {
	return &createMeteredEntitlementGrantEventSpec
}
