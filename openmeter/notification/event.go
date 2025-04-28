package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type Event struct {
	models.NamespacedModel
	models.Annotations

	// ID is the unique identifier for Event.
	ID string `json:"id"`
	// Type of the notification Event (e.g. entitlements.balance.threshold)
	Type EventType `json:"type"`
	// CreatedAt Timestamp when the notification event was created.
	CreatedAt time.Time `json:"createdAt"`
	// DeliveryStatus defines the delivery status of the notification Event per Channel.
	DeliveryStatus []EventDeliveryStatus `json:"deliveryStatus"`
	// Payload is the actual payload sent to Channel as part of the notification Event.
	Payload EventPayload `json:"payload"`
	// Rule defines the notification Rule that generated this Event.
	Rule Rule `json:"rule"`
	// DeduplicationHash is a hash that the handler can use to deduplicate events if needed
	HandlerDeduplicationHash string `json:"-"`
}

const (
	EventTypeBalanceThreshold = EventType(api.NotificationEventTypeEntitlementsBalanceThreshold)
)

type EventType api.NotificationEventType

func (t EventType) Validate() error {
	switch t {
	case EventTypeBalanceThreshold:
		return nil
	default:
		return fmt.Errorf("unknown notification event type: %q", t)
	}
}

func (t EventType) Values() []string {
	return []string{
		string(EventTypeBalanceThreshold),
	}
}

type EventPayloadMeta struct {
	Type EventType `json:"type"`
}

func (m EventPayloadMeta) Validate() error {
	return m.Type.Validate()
}

// EventPayload is a union type capturing payload for all EventType of Events.
type EventPayload struct {
	EventPayloadMeta

	// Balance Threshold
	BalanceThreshold BalanceThresholdPayload `json:"balanceThreshold"`
}

func (p EventPayload) Validate() error {
	switch p.Type {
	case EventTypeBalanceThreshold:
		return p.BalanceThreshold.Validate()
	default:
		return ValidationError{
			Err: fmt.Errorf("invalid event type: %s", p.Type),
		}
	}
}

type BalanceThresholdPayload struct {
	Entitlement api.EntitlementMetered                    `json:"entitlement"`
	Feature     api.Feature                               `json:"feature"`
	Subject     api.Subject                               `json:"subject"`
	Threshold   api.NotificationRuleBalanceThresholdValue `json:"threshold"`
	Value       api.EntitlementValue                      `json:"value"`
}

// Validate returns an error if balance threshold payload is invalid.
func (b BalanceThresholdPayload) Validate() error {
	return nil
}

var _ validator = (*ListEventsInput)(nil)

type ListEventsInput struct {
	pagination.Page

	Namespaces []string `json:"namespaces,omitempty"`
	Events     []string `json:"events,omitempty"`

	From time.Time `json:"from,omitempty"`
	To   time.Time `json:"to,omitempty"`

	Subjects []string `json:"subjects,omitempty"`
	Features []string `json:"features,omitempty"`

	Rules    []string `json:"rules,omitempty"`
	Channels []string `json:"channels,omitempty"`

	DeduplicationHashes []string `json:"deduplicationHashes,omitempty"`

	DeliveryStatusStates []EventDeliveryStatusState `json:"deliveryStatusStates,omitempty"`

	OrderBy OrderBy
	Order   sortx.Order
}

func (i *ListEventsInput) Validate(_ context.Context, _ Service) error {
	if i.From.After(i.To) {
		return ValidationError{
			Err: fmt.Errorf("invalid time period: period start (%s) is after the period end (%s)", i.From, i.To),
		}
	}

	switch i.OrderBy {
	case OrderByID, OrderByCreatedAt:
	case "":
		i.OrderBy = OrderByID
	default:
		return ValidationError{
			Err: fmt.Errorf("invalid event order_by: %s", i.OrderBy),
		}
	}

	return nil
}

type ListEventsResult = pagination.PagedResponse[Event]

var _ validator = (*GetEventInput)(nil)

type GetEventInput struct {
	models.NamespacedID
}

func (i GetEventInput) Validate(_ context.Context, _ Service) error {
	if i.Namespace == "" {
		return ValidationError{
			Err: fmt.Errorf("namespace must be provided"),
		}
	}

	if i.ID == "" {
		return ValidationError{
			Err: fmt.Errorf("event id must be provided"),
		}
	}

	return nil
}

var _ validator = (*CreateEventInput)(nil)

type CreateEventInput struct {
	models.NamespacedModel
	Annotations models.Annotations `json:"annotations,omitempty"`

	// Type of the notification Event (e.g. entitlements.balance.threshold)
	Type EventType `json:"type"`
	// Payload is the actual payload sent to Channel as part of the notification Event.
	Payload EventPayload `json:"payload"`
	// RuleID defines the notification Rule that generated this Event.
	RuleID string `json:"ruleId"`
	// HandlerDeduplicationHash is a hash that the handler can use to deduplicate events if needed
	HandlerDeduplicationHash string `json:"handlerDeduplicationHash"`
}

func (i CreateEventInput) Validate(ctx context.Context, service Service) error {
	if err := i.Type.Validate(); err != nil {
		return err
	}

	return nil
}

const (
	EventDeliveryStatusStateSuccess = EventDeliveryStatusState(api.NotificationEventDeliveryStatusStateSuccess)
	EventDeliveryStatusStateFailed  = EventDeliveryStatusState(api.NotificationEventDeliveryStatusStateFailed)
	EventDeliveryStatusStateSending = EventDeliveryStatusState(api.NotificationEventDeliveryStatusStateSending)
	EventDeliveryStatusStatePending = EventDeliveryStatusState(api.NotificationEventDeliveryStatusStatePending)
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

	return nil
}

type ListEventsDeliveryStatusResult = pagination.PagedResponse[EventDeliveryStatus]

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

func PayloadToMapInterface(t any) (map[string]interface{}, error) {
	b, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}

	var m map[string]interface{}
	if err = json.Unmarshal(b, &m); err != nil {
		return nil, err
	}

	return m, nil
}
