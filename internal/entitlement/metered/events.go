package meteredentitlement

import (
	"errors"
	"time"

	"github.com/openmeterio/openmeter/internal/event/models"
	"github.com/openmeterio/openmeter/internal/event/spec"
)

const (
	EventSubsystem spec.EventSubsystem = "meteredEntitlement"

	EventResetEntitlementUsage         spec.EventName = "resetEntitlementUsage"
	EventCreateMeteredEntitlementGrant spec.EventName = "createMeteredEntitlementGrant"
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

type CreateMeteredEntitlementGrantEvent struct {
	EntitlementGrant
	Subject   models.SubjectKeyAndID `json:"subjectKey"`
	Namespace models.NamespaceID     `json:"namespace"`
}

var createMeteredEntitlementGrantEventSpec = spec.EventTypeSpec{
	Subsystem:   EventSubsystem,
	Name:        EventCreateMeteredEntitlementGrant,
	SpecVersion: "1.0",
	Version:     "v1",
}

func (e CreateMeteredEntitlementGrantEvent) Spec() *spec.EventTypeSpec {
	return &createMeteredEntitlementGrantEventSpec
}

func (e CreateMeteredEntitlementGrantEvent) Validate() error {
	if e.ID == "" {
		return errors.New("ID must not be empty")
	}

	if err := e.Subject.Validate(); err != nil {
		return err
	}

	if err := e.Namespace.Validate(); err != nil {
		return err
	}
	return nil
}
