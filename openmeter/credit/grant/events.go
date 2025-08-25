package grant

import (
	"errors"

	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	eventmodels "github.com/openmeterio/openmeter/openmeter/event/models"
	"github.com/openmeterio/openmeter/openmeter/subject"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
)

const (
	EventSubsystem metadata.EventSubsystem = "credit"
)

// Events based on grantEventV1 should slowly be removed. The issue with this old pattern is that domain models
// are embedded inside the event, which means that domain changes break previous events.
// Future versions (starting with v2) will declare the event using only primitives.
// Deprecated: use events_v2.go instead
type grantEventV1 struct {
	Namespace eventmodels.NamespaceID `json:"namespace"`
	Subject   subject.SubjectKey      `json:"subject"`
	Grant
}

func (g grantEventV1) Validate() error {
	// Basic sanity on grant
	if g.Grant.ID == "" {
		return errors.New("GrantID must be set")
	}

	if g.Grant.OwnerID == "" {
		return errors.New("GrantOwnerID must be set")
	}

	if err := g.Subject.Validate(); err != nil {
		return err
	}

	if err := g.Namespace.Validate(); err != nil {
		return err
	}

	return nil
}

func (e grantEventV1) EventMetadata() metadata.EventMetadata {
	return metadata.EventMetadata{
		Source:  metadata.ComposeResourcePath(e.Namespace.ID, metadata.EntityEntitlement, e.OwnerID, metadata.EntityGrant, e.ID),
		Subject: metadata.ComposeResourcePath(e.Namespace.ID, metadata.EntitySubjectKey, e.Subject.Key),
	}
}

type CreatedEvent grantEventV1

var (
	_ marshaler.Event = CreatedEvent{}

	grantCreatedEventName = metadata.GetEventName(metadata.EventType{
		Subsystem: EventSubsystem,
		Name:      "grant.created",
		Version:   "v1",
	})
)

func (e CreatedEvent) EventName() string {
	return grantCreatedEventName
}

func (e CreatedEvent) EventMetadata() metadata.EventMetadata {
	return grantEventV1(e).EventMetadata()
}

func (e CreatedEvent) Validate() error {
	return grantEventV1(e).Validate()
}

type VoidedEvent grantEventV1

var (
	_ marshaler.Event = VoidedEvent{}

	grantVoidedEventName = metadata.GetEventName(metadata.EventType{
		Subsystem: EventSubsystem,
		Name:      "grant.voided",
		Version:   "v1",
	})
)

func (e VoidedEvent) EventName() string {
	return grantVoidedEventName
}

func (e VoidedEvent) EventMetadata() metadata.EventMetadata {
	return grantEventV1(e).EventMetadata()
}

func (e VoidedEvent) Validate() error {
	return grantEventV1(e).Validate()
}
