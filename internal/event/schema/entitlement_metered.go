package schema

import (
	"time"

	meteredentitlement "github.com/openmeterio/openmeter/internal/entitlement/metered"
	"github.com/openmeterio/openmeter/internal/event/types"
)

const (
	subsystemMeteredEntitlement = "meteredEntitlement"
)

type ResetEntitlementEvent struct {
	EntitlementID string    `json:"entitlement_id"`
	Namespace     string    `json:"namespace"`
	SubjectKey    string    `json:"subjectKey"`
	At            time.Time `json:"at"`
	RetainAnchor  bool      `json:"retainAnchor"`
}

var resetEntitlementEventSpec = types.EventTypeSpec{
	Subsystem:   subsystemMeteredEntitlement,
	Name:        "resetEntitlementUsage",
	SpecVersion: "1.0",
	Version:     "v1",
	SubjectKind: subjectKindEntitlement,
}

func (e ResetEntitlementEvent) Spec() *types.EventTypeSpec {
	return &resetEntitlementEventSpec
}

type CreateMeteredEntitlementGrantEvent struct {
	meteredentitlement.EntitlementGrant
	SubjectKey string `json:"subjectKey"`
	Namespace  string `json:"namespace"`
}

var createMeteredEntitlementGrantEventSpec = types.EventTypeSpec{
	Subsystem:   subsystemMeteredEntitlement,
	Name:        "createMeteredEntitlementGrant",
	SpecVersion: "1.0",
	Version:     "v1",
	SubjectKind: subjectKindEntitlement,
}

func (e CreateMeteredEntitlementGrantEvent) Spec() *types.EventTypeSpec {
	return &createMeteredEntitlementGrantEventSpec
}
