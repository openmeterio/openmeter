package meteredentitlement

import (
	"errors"
	"time"

	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/event/models"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
)

const (
	EventSubsystem metadata.EventSubsystem = "meteredEntitlement"
)

type EntitlementResetEvent struct {
	EntitlementID string                 `json:"entitlementId"`
	Namespace     models.NamespaceID     `json:"namespace"`
	Subject       models.SubjectKeyAndID `json:"subject"`
	ResetAt       time.Time              `json:"resetAt"`
	RetainAnchor  bool                   `json:"retainAnchor"`
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
	return nil
}
