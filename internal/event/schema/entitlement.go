package schema

import (
	"github.com/openmeterio/openmeter/internal/event/types"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
)

const (
	subjectKindEntitlement = "entitlement"
	subsystemEntitlement   = "entitlement"
)

type entitlementEvent struct {
	entitlement.Entitlement
	Namespace string `json:"namespace"`
}

type EntitlementCreatedEvent entitlementEvent

var entitlementCreatedEventSpec = types.EventTypeSpec{
	Subsystem:   subsystemEntitlement,
	Name:        "createEntitlement",
	SpecVersion: "1.0",
	Version:     "v1",
	SubjectKind: subjectKindEntitlement,
}

func (e EntitlementCreatedEvent) Spec() *types.EventTypeSpec {
	return &entitlementCreatedEventSpec
}

type EntitlementDeletedEvent entitlementEvent

var entitlementDeletedEventSpec = types.EventTypeSpec{
	Subsystem:   subsystemEntitlement,
	Name:        "deleteEntitlement",
	SpecVersion: "1.0",
	Version:     "v1",
	SubjectKind: subjectKindEntitlement,
}

func (e EntitlementDeletedEvent) Spec() *types.EventTypeSpec {
	return &entitlementDeletedEventSpec
}
