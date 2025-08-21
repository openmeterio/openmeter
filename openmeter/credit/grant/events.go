package grant

import (
	"errors"
	"time"

	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/event/models"
	"github.com/openmeterio/openmeter/openmeter/subject"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
)

const (
	EventSubsystem metadata.EventSubsystem = "credit"
)

type grantEvent struct {
	// Core identifiers
	ID      string             `json:"id"`
	OwnerID string             `json:"owner"`
	Subject subject.SubjectKey `json:"subject"`
	// Namespace separate to ensure it is serialized
	Namespace models.NamespaceID `json:"namespace"`

	// Timestamps
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	// Selected grant fields (replicates existing v1 shape)
	Amount           float64           `json:"amount"`
	Priority         uint8             `json:"priority"`
	EffectiveAt      time.Time         `json:"effectiveAt"`
	ExpiresAt        time.Time         `json:"expiresAt"`
	VoidedAt         *time.Time        `json:"voidedAt,omitempty"`
	ResetMaxRollover float64           `json:"resetMaxRollover"`
	ResetMinRollover float64           `json:"resetMinRollover"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}

func (g grantEvent) Validate() error {
	// Basic sanity on grant
	if g.ID == "" {
		return errors.New("GrantID must be set")
	}

	if g.OwnerID == "" {
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

func (e grantEvent) EventMetadata() metadata.EventMetadata {
	return metadata.EventMetadata{
		Source:  metadata.ComposeResourcePath(e.Namespace.ID, metadata.EntityEntitlement, e.OwnerID, metadata.EntityGrant, e.ID),
		Subject: metadata.ComposeResourcePath(e.Namespace.ID, metadata.EntitySubjectKey, e.Subject.Key),
	}
}

type CreatedEvent grantEvent

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
	return grantEvent(e).EventMetadata()
}

func (e CreatedEvent) Validate() error {
	return grantEvent(e).Validate()
}

type VoidedEvent grantEvent

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
	return grantEvent(e).EventMetadata()
}

func (e VoidedEvent) Validate() error {
	return grantEvent(e).Validate()
}

// V2 events (customer-centric, no subject dependency)

type grantEventV2 struct {
	// Core identifiers
	ID         string             `json:"id"`
	OwnerID    string             `json:"owner"`
	Namespace  models.NamespaceID `json:"namespace"`
	CustomerID string             `json:"customer_id"`

	// Timestamps
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	// Selected grant fields (explicit, stable)
	Amount           float64           `json:"amount"`
	Priority         uint8             `json:"priority"`
	EffectiveAt      time.Time         `json:"effectiveAt"`
	ExpiresAt        time.Time         `json:"expiresAt"`
	VoidedAt         *time.Time        `json:"voidedAt,omitempty"`
	ResetMaxRollover float64           `json:"resetMaxRollover"`
	ResetMinRollover float64           `json:"resetMinRollover"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}

func (g grantEventV2) Validate() error {
	if g.ID == "" {
		return errors.New("GrantID must be set")
	}
	if g.OwnerID == "" {
		return errors.New("GrantOwnerID must be set")
	}
	if err := g.Namespace.Validate(); err != nil {
		return err
	}
	if g.CustomerID == "" {
		return errors.New("customer_id must be set")
	}
	return nil
}

func (e grantEventV2) EventMetadata() metadata.EventMetadata {
	return metadata.EventMetadata{
		Source:  metadata.ComposeResourcePath(e.Namespace.ID, metadata.EntityEntitlement, e.OwnerID, metadata.EntityGrant, e.ID),
		Subject: metadata.ComposeResourcePath(e.Namespace.ID, metadata.EntityCustomer, e.CustomerID),
	}
}

type CreatedEventV2 grantEventV2

var (
	_ marshaler.Event = CreatedEventV2{}

	grantCreatedEventNameV2 = metadata.GetEventName(metadata.EventType{
		Subsystem: EventSubsystem,
		Name:      "grant.created",
		Version:   "v2",
	})
)

func (e CreatedEventV2) EventName() string { return grantCreatedEventNameV2 }
func (e CreatedEventV2) EventMetadata() metadata.EventMetadata {
	return grantEventV2(e).EventMetadata()
}
func (e CreatedEventV2) Validate() error { return grantEventV2(e).Validate() }

type VoidedEventV2 grantEventV2

var (
	_ marshaler.Event = VoidedEventV2{}

	grantVoidedEventNameV2 = metadata.GetEventName(metadata.EventType{
		Subsystem: EventSubsystem,
		Name:      "grant.voided",
		Version:   "v2",
	})
)

func (e VoidedEventV2) EventName() string                     { return grantVoidedEventNameV2 }
func (e VoidedEventV2) EventMetadata() metadata.EventMetadata { return grantEventV2(e).EventMetadata() }
func (e VoidedEventV2) Validate() error                       { return grantEventV2(e).Validate() }

// Helper constructors (V2)
func NewCreatedEventV2FromGrant(g Grant, customerID string) CreatedEventV2 {
	return CreatedEventV2{
		ID:               g.ID,
		OwnerID:          g.OwnerID,
		Namespace:        models.NamespaceID{ID: g.Namespace},
		CustomerID:       customerID,
		CreatedAt:        g.CreatedAt,
		UpdatedAt:        g.UpdatedAt,
		Amount:           g.Amount,
		Priority:         g.Priority,
		EffectiveAt:      g.EffectiveAt,
		ExpiresAt:        g.ExpiresAt,
		VoidedAt:         g.VoidedAt,
		ResetMaxRollover: g.ResetMaxRollover,
		ResetMinRollover: g.ResetMinRollover,
		Metadata:         g.Metadata,
	}
}

func NewVoidedEventV2FromGrant(g Grant, customerID string, voidedAt time.Time) VoidedEventV2 {
	return VoidedEventV2{
		ID:               g.ID,
		OwnerID:          g.OwnerID,
		Namespace:        models.NamespaceID{ID: g.Namespace},
		CustomerID:       customerID,
		CreatedAt:        g.CreatedAt,
		UpdatedAt:        voidedAt,
		Amount:           g.Amount,
		Priority:         g.Priority,
		EffectiveAt:      g.EffectiveAt,
		ExpiresAt:        g.ExpiresAt,
		VoidedAt:         &voidedAt,
		ResetMaxRollover: g.ResetMaxRollover,
		ResetMinRollover: g.ResetMinRollover,
		Metadata:         g.Metadata,
	}
}
