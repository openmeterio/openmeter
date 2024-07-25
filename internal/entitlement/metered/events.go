package meteredentitlement

import (
	"errors"
	"time"

	"github.com/openmeterio/openmeter/internal/event/models"
	"github.com/openmeterio/openmeter/internal/event/spec"
)

const (
	EventSubsystem spec.EventSubsystem = "meteredEntitlement"
)

const (
	EventResetEntitlementUsage spec.EventName = "resetEntitlementUsage"
)

type ResetEntitlementEvent struct {
	EntitlementID string                 `json:"entitlementId"`
	Namespace     models.NamespaceID     `json:"namespace"`
	Subject       models.SubjectKeyAndID `json:"subject"`
	ResetAt       time.Time              `json:"resetAt"`
	RetainAnchor  bool                   `json:"retainAnchor"`
}

var resetEntitlementEventSpec = spec.EventTypeSpec{
	Subsystem:   EventSubsystem,
	Name:        EventResetEntitlementUsage,
	SpecVersion: "1.0",
	Version:     "v1",
}

func (e ResetEntitlementEvent) Spec() *spec.EventTypeSpec {
	return &resetEntitlementEventSpec
}

func (e ResetEntitlementEvent) Validate() error {
	if e.EntitlementID == "" {
		return errors.New("entitlementID must be set")
	}

	if err := e.Namespace.Validate(); err != nil {
		return err
	}

	if err := e.Subject.Validate(); err != nil {
		return err
	}

	if e.ResetAt.IsZero() {
		return errors.New("resetAt must be set")
	}
	return nil
}
