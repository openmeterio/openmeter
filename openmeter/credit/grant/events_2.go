package grant

import (
	"errors"
	"time"

	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	eventmodels "github.com/openmeterio/openmeter/openmeter/event/models"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// Literal types for versioning

type grantEventV2 struct {
	Namespace  eventmodels.NamespaceID  `json:"namespace"`
	Grant      grantEventV2GrantLiteral `json:"grant"`
	CustomerID string                   `json:"customerId"`
}

type grantEventV2GrantLiteral struct {
	ManagedModel    models.ManagedModel
	NamespacedModel models.NamespacedModel

	ID               string               `json:"id,omitempty"`
	OwnerID          string               `json:"owner"`
	Amount           float64              `json:"amount"`
	Priority         uint8                `json:"priority"`
	EffectiveAt      time.Time            `json:"effectiveAt"`
	Expiration       *ExpirationPeriod    `json:"expiration"`
	ExpiresAt        *time.Time           `json:"expiresAt"`
	Metadata         map[string]string    `json:"metadata,omitempty"`
	VoidedAt         *time.Time           `json:"voidedAt,omitempty"`
	ResetMaxRollover float64              `json:"resetMaxRollover"`
	ResetMinRollover float64              `json:"resetMinRollover"`
	Recurrence       *timeutil.Recurrence `json:"recurrence,omitempty"`
}

func (g grantEventV2GrantLiteral) ToDomainGrant() Grant {
	return Grant{
		ManagedModel:     g.ManagedModel,
		NamespacedModel:  g.NamespacedModel,
		ID:               g.ID,
		OwnerID:          g.OwnerID,
		Amount:           g.Amount,
		Priority:         g.Priority,
		EffectiveAt:      g.EffectiveAt,
		Expiration:       g.Expiration,
		ExpiresAt:        g.ExpiresAt,
		VoidedAt:         g.VoidedAt,
		ResetMaxRollover: g.ResetMaxRollover,
		ResetMinRollover: g.ResetMinRollover,
		Recurrence:       g.Recurrence,
		Metadata:         g.Metadata,
	}
}

func (g grantEventV2GrantLiteral) Validate() error {
	domainGrant := g.ToDomainGrant()
	if err := domainGrant.Validate(); err != nil {
		return err
	}
	return nil
}

func mapGrantToV2Literal(g Grant) grantEventV2GrantLiteral {
	return grantEventV2GrantLiteral{
		ManagedModel:     g.ManagedModel,
		NamespacedModel:  g.NamespacedModel,
		ID:               g.ID,
		OwnerID:          g.OwnerID,
		Amount:           g.Amount,
		Priority:         g.Priority,
		EffectiveAt:      g.EffectiveAt,
		Expiration:       g.Expiration,
		ExpiresAt:        g.ExpiresAt,
		VoidedAt:         g.VoidedAt,
		ResetMaxRollover: g.ResetMaxRollover,
		ResetMinRollover: g.ResetMinRollover,
		Recurrence:       g.Recurrence,
		Metadata:         g.Metadata,
	}
}

func (g grantEventV2) Validate() error {
	if err := g.Grant.Validate(); err != nil {
		return err
	}

	if g.CustomerID == "" {
		return errors.New("customer_id must be set")
	}

	if err := g.Namespace.Validate(); err != nil {
		return err
	}

	return nil
}

func (e grantEventV2) EventMetadata() metadata.EventMetadata {
	return metadata.EventMetadata{
		Source:  metadata.ComposeResourcePath(e.Namespace.ID, metadata.EntityEntitlement, e.Grant.OwnerID, metadata.EntityGrant, e.Grant.ID),
		Subject: metadata.ComposeResourcePath(e.Namespace.ID, metadata.EntityCustomer, e.CustomerID),
	}
}

// V2 events

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

func NewCreatedEventV2FromGrant(g Grant, streamingCustomer streaming.Customer) CreatedEventV2 {
	return CreatedEventV2{
		Namespace:  eventmodels.NamespaceID{ID: g.Namespace},
		Grant:      mapGrantToV2Literal(g),
		CustomerID: streamingCustomer.GetUsageAttribution().ID,
	}
}

func NewVoidedEventV2FromGrant(g Grant, streamingCustomer streaming.Customer, voidedAt time.Time) VoidedEventV2 {
	literal := mapGrantToV2Literal(g)

	literal.VoidedAt = &voidedAt

	return VoidedEventV2{
		Namespace:  eventmodels.NamespaceID{ID: g.Namespace},
		Grant:      literal,
		CustomerID: streamingCustomer.GetUsageAttribution().ID,
	}
}
