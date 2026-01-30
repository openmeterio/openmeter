package meteredentitlement

import (
	"errors"
	"time"

	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/event/models"
	"github.com/openmeterio/openmeter/openmeter/subject"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
)

const (
	EventSubsystem metadata.EventSubsystem = "meteredEntitlement"
)

// Deprecated: use EntitlementResetEventV2 instead
type EntitlementResetEvent struct {
	EntitlementID    string             `json:"entitlementId"`
	Namespace        models.NamespaceID `json:"namespace"`
	Subject          subject.SubjectKey `json:"subject"`
	ResetAt          time.Time          `json:"resetAt"`
	RetainAnchor     bool               `json:"retainAnchor"`
	ResetRequestedAt time.Time          `json:"resetRequestedAt"`
}

var (
	_ marshaler.Event = EntitlementResetEvent{}

	resetEntitlementEventName = metadata.GetEventName(metadata.EventType{
		Subsystem: EventSubsystem,
		Name:      "entitlement.reset",
		Version:   "v1",
	})
)

func (e EntitlementResetEvent) EventName() string {
	return resetEntitlementEventName
}

func (e EntitlementResetEvent) EventMetadata() metadata.EventMetadata {
	return metadata.EventMetadata{
		Source:  metadata.ComposeResourcePath(e.Namespace.ID, metadata.EntityEntitlement, e.EntitlementID),
		Subject: metadata.ComposeResourcePath(e.Namespace.ID, metadata.EntitySubjectKey, e.Subject.Key),
	}
}

func (e EntitlementResetEvent) Validate() error {
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

	if e.ResetRequestedAt.IsZero() {
		return errors.New("resetRequestedAt must be set")
	}

	return nil
}

type EntitlementResetEventV3 struct {
	EntitlementID    string             `json:"entitlementId"`
	Namespace        models.NamespaceID `json:"namespace"`
	CustomerID       string             `json:"customerId"`
	ResetAt          time.Time          `json:"resetAt"`
	RetainAnchor     bool               `json:"retainAnchor"`
	ResetRequestedAt time.Time          `json:"resetRequestedAt"`
}

var (
	_ marshaler.Event = EntitlementResetEvent{}

	resetEntitlementEventNameV3 = metadata.GetEventName(metadata.EventType{
		Subsystem: EventSubsystem,
		Name:      "entitlement.reset",
		Version:   "v3",
	})
)

func (e EntitlementResetEventV3) EventName() string {
	return resetEntitlementEventNameV3
}

func (e EntitlementResetEventV3) EventMetadata() metadata.EventMetadata {
	return metadata.EventMetadata{
		Source:  metadata.ComposeResourcePath(e.Namespace.ID, metadata.EntityEntitlement, e.EntitlementID),
		Subject: metadata.ComposeResourcePath(e.Namespace.ID, metadata.EntityCustomer, e.CustomerID),
	}
}

func (e EntitlementResetEventV3) Validate() error {
	if e.EntitlementID == "" {
		return errors.New("entitlementID must be set")
	}

	if err := e.Namespace.Validate(); err != nil {
		return err
	}

	if e.CustomerID == "" {
		return errors.New("customerID must be set")
	}

	if e.ResetAt.IsZero() {
		return errors.New("resetAt must be set")
	}

	if e.ResetRequestedAt.IsZero() {
		return errors.New("resetRequestedAt must be set")
	}

	return nil
}
