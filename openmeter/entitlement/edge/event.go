package edge

import (
	"errors"
	"time"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/event/models"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
)

type EntitlementCacheMissEvent struct {
	Namespace                 models.NamespaceID
	EntitlementNamespace      string
	SubjectKey                string
	EntitlementIdOrFeatureKey string
	At                        time.Time
}

var _ marshaler.Event = EntitlementCacheMissEvent{}

var eventName = metadata.GetEventName(metadata.EventType{
	Subsystem: entitlement.EventSubsystem,
	Name:      "entitlement.cachemiss",
	Version:   "v1",
})

func (e EntitlementCacheMissEvent) EventName() string {
	return eventName
}

func (e EntitlementCacheMissEvent) EventMetadata() metadata.EventMetadata {
	return metadata.EventMetadata{
		Subject: metadata.ComposeResourcePath(e.Namespace.ID, metadata.EntitySubjectKey, e.SubjectKey),
	}
}

func (e EntitlementCacheMissEvent) Validate() error {
	if err := e.Namespace.Validate(); err != nil {
		return err
	}

	if e.EntitlementNamespace == "" {
		return errors.New("namespace is required")
	}

	if e.SubjectKey == "" {
		return errors.New("subjectKey is required")
	}

	if e.EntitlementIdOrFeatureKey == "" {
		return errors.New("entitlementIdOrFeatureKey is required")
	}

	if e.At.IsZero() {
		return errors.New("at is required")
	}

	return nil
}
