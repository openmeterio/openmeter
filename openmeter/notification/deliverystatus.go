package notification

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

const (
	EventDeliveryStatusStateSuccess EventDeliveryStatusState = "SUCCESS"
	EventDeliveryStatusStateFailed  EventDeliveryStatusState = "FAILED"
	EventDeliveryStatusStateSending EventDeliveryStatusState = "SENDING"
	EventDeliveryStatusStatePending EventDeliveryStatusState = "PENDING"
)

type EventDeliveryStatusState string

func (e EventDeliveryStatusState) Validate() error {
	switch e {
	case EventDeliveryStatusStateSuccess, EventDeliveryStatusStateFailed, EventDeliveryStatusStateSending, EventDeliveryStatusStatePending:
		return nil
	default:
		return ValidationError{
			Err: fmt.Errorf("invalid event delivery status state: %s", e),
		}
	}
}

func (e EventDeliveryStatusState) Values() []string {
	return []string{
		string(EventDeliveryStatusStateSuccess),
		string(EventDeliveryStatusStateFailed),
		string(EventDeliveryStatusStateSending),
		string(EventDeliveryStatusStatePending),
	}
}

type EventDeliveryStatus struct {
	models.NamespacedModel

	// ID is the unique identifier for Event.
	ID string `json:"id"`
	// EventID defines the Event identifier the EventDeliveryStatus belongs to.
	EventID string `json:"eventId"`

	ChannelID string                   `json:"channelId"`
	State     EventDeliveryStatusState `json:"state"`
	Reason    string                   `json:"reason,omitempty"`
	CreatedAt time.Time                `json:"createdAt"`
	UpdatedAt time.Time                `json:"updatedAt,omitempty"`
}

var _ validator = (*ListEventsDeliveryStatusInput)(nil)

type ListEventsDeliveryStatusInput struct {
	pagination.Page

	// Namespaces is a list of namespaces to be used to filter the list of EventDeliveryStatus to be returned.
	Namespaces []string

	// From limits the scope fo the request by defining the earliest date to be used for lookup.
	// This filter is applied to EventDeliveryStatus.UpdatedAt field.
	From time.Time
	// To limits the scope fo the request by defining the latest date to be used for lookup.
	// This filter is applied to EventDeliveryStatus.UpdatedAt field.
	To time.Time

	// Events is a list of Event identifiers used as filter.
	Events []string
	// Channels is a list of Channel identifiers used as filter.
	Channels []string
	// State is a list of Event State used as filter.
	States []EventDeliveryStatusState
}

func (i ListEventsDeliveryStatusInput) Validate(_ context.Context, _ Service) error {
	if i.From.After(i.To) {
		return ValidationError{
			Err: fmt.Errorf("invalid time range: parameter From (%s) is after To (%s)", i.From, i.To),
		}
	}

	for _, s := range i.States {
		if err := s.Validate(); err != nil {
			return err
		}
	}

	return nil
}

type ListEventsDeliveryStatusResult = pagination.Result[EventDeliveryStatus]

var _ validator = (*GetEventDeliveryStatusInput)(nil)

type GetEventDeliveryStatusInput struct {
	models.NamespacedModel

	// ID the unique identifier of the EventDeliveryStatus.
	ID string
	// EventID defines the Event identifier the EventDeliveryStatus belongs to. Must be provided if ID is empty.
	EventID string
	// ChannelID defines the Channel identifier the EventDeliveryStatus belongs to. Must be provided if ID is empty.
	ChannelID string
}

func (i GetEventDeliveryStatusInput) Validate(_ context.Context, _ Service) error {
	if i.Namespace == "" {
		return ValidationError{
			Err: fmt.Errorf("namespace must be provided"),
		}
	}

	if i.ID == "" && (i.EventID == "" || i.ChannelID == "") {
		return ValidationError{
			Err: fmt.Errorf("delivery status ID or both channel ID and event ID must be provided"),
		}
	}

	return nil
}

var _ validator = (*UpdateEventDeliveryStatusInput)(nil)

type UpdateEventDeliveryStatusInput struct {
	models.NamespacedModel

	// ID the unique identifier of the EventDeliveryStatus.
	ID string
	// State is the delivery state of the Event.
	State EventDeliveryStatusState
	// Reason describes the reason for the latest State transition.
	Reason string
	// EventID defines the Event identifier the EventDeliveryStatus belongs to. Must be provided if ID is empty.
	EventID string
	// ChannelID defines the Channel identifier the EventDeliveryStatus belongs to. Must be provided if ID is empty.
	ChannelID string
}

func (i UpdateEventDeliveryStatusInput) Validate(_ context.Context, _ Service) error {
	if i.Namespace == "" {
		return ValidationError{
			Err: fmt.Errorf("namespace must be provided"),
		}
	}

	if err := i.State.Validate(); err != nil {
		return err
	}

	if i.ID == "" && (i.EventID == "" || i.ChannelID == "") {
		return ValidationError{
			Err: fmt.Errorf("delivery status ID or both channel ID and event ID must be provided"),
		}
	}

	return nil
}
