package grant

import (
	"errors"

	"github.com/openmeterio/openmeter/internal/event/models"
	"github.com/openmeterio/openmeter/internal/event/spec"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
)

const (
	EventSubsystem spec.EventSubsystem = "credit"
)

type grantEvent struct {
	Grant

	Subject models.SubjectKeyAndID `json:"subject"`
	// Namespace from Grant cannot be used as it will never be serialized
	Namespace models.NamespaceID `json:"namespace"`
}

func (g grantEvent) Validate() error {
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

func (e grantEvent) EventMetadata() spec.EventMetadata {
	return spec.EventMetadata{
		Source:  spec.ComposeResourcePath(e.Namespace.ID, spec.EntityEntitlement, string(e.OwnerID), spec.EntityGrant, e.ID),
		Subject: spec.ComposeResourcePath(e.Namespace.ID, spec.EntitySubjectKey, e.Subject.Key),
	}
}

type CreatedEvent grantEvent

var (
	_ marshaler.Event = CreatedEvent{}

	grantCreatedEventName = spec.GetEventName(spec.EventTypeSpec{
		Subsystem: EventSubsystem,
		Name:      "grant.created",
		Version:   "v1",
	})
)

func (e CreatedEvent) EventName() string {
	return grantCreatedEventName
}

func (e CreatedEvent) EventMetadata() spec.EventMetadata {
	return grantEvent(e).EventMetadata()
}

func (e CreatedEvent) Validate() error {
	return grantEvent(e).Validate()
}

type VoidedEvent grantEvent

var (
	_ marshaler.Event = VoidedEvent{}

	grantVoidedEventName = spec.GetEventName(spec.EventTypeSpec{
		Subsystem: EventSubsystem,
		Name:      "grant.voided",
		Version:   "v1",
	})
)

func (e VoidedEvent) EventName() string {
	return grantVoidedEventName
}

func (e VoidedEvent) EventMetadata() spec.EventMetadata {
	return grantEvent(e).EventMetadata()
}

func (e VoidedEvent) Validate() error {
	return grantEvent(e).Validate()
}
