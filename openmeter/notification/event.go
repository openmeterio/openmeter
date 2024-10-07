package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type payloadObjectMapper interface {
	AsNotificationEventBalanceThresholdPayload() api.NotificationEventBalanceThresholdPayload
}

var _ payloadObjectMapper = (*Event)(nil)

type Event struct {
	models.NamespacedModel
	Annotations

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

func (e Event) AsNotificationEvent() (api.NotificationEvent, error) {
	var err error

	var rule api.NotificationRule
	rule, err = e.Rule.AsNotificationRule()
	if err != nil {
		return api.NotificationEvent{}, fmt.Errorf("failed to cast notification rule: %w", err)
	}

	// Populate ChannelMeta in EventDeliveryStatus from Even.Rule.Channels as we only store Channel.ID in database
	// for EventDeliveryStatus objects.
	channelsByID := make(map[string]Channel, len(e.Rule.Channels))
	for _, channel := range e.Rule.Channels {
		channelsByID[channel.ID] = channel
	}

	deliveryStatuses := make([]api.NotificationEventDeliveryStatus, 0, len(e.DeliveryStatus))
	for _, deliveryStatus := range e.DeliveryStatus {
		status := api.NotificationEventDeliveryStatus{
			Channel: ChannelMeta{
				Id: deliveryStatus.ChannelID,
			},
			State:     api.NotificationEventDeliveryStatusState(deliveryStatus.State),
			UpdatedAt: deliveryStatus.UpdatedAt,
		}
		if channel, ok := channelsByID[deliveryStatus.ChannelID]; ok {
			status.Channel = api.NotificationChannelMeta{
				Id:   deliveryStatus.ChannelID,
				Type: api.NotificationChannelType(channel.Type),
			}
		}

		deliveryStatuses = append(deliveryStatuses, status)
	}

	var annotations api.Annotations
	if len(e.Annotations) > 0 {
		annotations = make(api.Annotations)
		for k, v := range e.Annotations {
			annotations[k] = v
		}
	}

	event := api.NotificationEvent{
		CreatedAt:      e.CreatedAt,
		DeliveryStatus: deliveryStatuses,
		Id:             e.ID,
		Rule:           rule,
		Annotations:    lo.EmptyableToPtr(annotations),
	}

	switch e.Type {
	case EventTypeBalanceThreshold:
		event.Type = api.NotificationEventTypeEntitlementsBalanceThreshold
		event.Payload = e.AsNotificationEventBalanceThresholdPayload()
	default:
		return event, ValidationError{
			Err: fmt.Errorf("invalid event type: %s", e.Type),
		}
	}

	return event, nil
}

func (e Event) AsNotificationEventBalanceThresholdPayload() api.NotificationEventBalanceThresholdPayload {
	return api.NotificationEventBalanceThresholdPayload{
		Id:        e.ID,
		Timestamp: e.CreatedAt,
		Type:      api.NotificationEventBalanceThresholdPayloadTypeEntitlementsBalanceThreshold,
		Data: struct {
			Entitlement api.EntitlementMetered                    `json:"entitlement"`
			Feature     api.Feature                               `json:"feature"`
			Subject     api.Subject                               `json:"subject"`
			Threshold   api.NotificationRuleBalanceThresholdValue `json:"threshold"`
			Value       api.EntitlementValue                      `json:"value"`
		}{
			Value:       e.Payload.BalanceThreshold.Value,
			Entitlement: e.Payload.BalanceThreshold.Entitlement,
			Feature:     e.Payload.BalanceThreshold.Feature,
			Subject:     e.Payload.BalanceThreshold.Subject,
			Threshold:   e.Payload.BalanceThreshold.Threshold,
		},
	}
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

func (p EventPayload) FromNotificationEventBalanceThresholdPayload(r api.NotificationEventBalanceThresholdPayload) EventPayload {
	return EventPayload{
		EventPayloadMeta: EventPayloadMeta{
			Type: EventType(r.Type),
		},
		BalanceThreshold: BalanceThresholdPayload{
			Entitlement: r.Data.Entitlement,
			Feature:     r.Data.Feature,
			Subject:     r.Data.Subject,
			Value:       r.Data.Value,
			Threshold:   r.Data.Threshold,
		},
	}
}

func (p EventPayload) AsNotificationEventBalanceThresholdPayload(eventId string, ts time.Time) api.NotificationEventBalanceThresholdPayload {
	return api.NotificationEventBalanceThresholdPayload{
		Data: struct {
			Entitlement api.EntitlementMetered                    `json:"entitlement"`
			Feature     api.Feature                               `json:"feature"`
			Subject     api.Subject                               `json:"subject"`
			Threshold   api.NotificationRuleBalanceThresholdValue `json:"threshold"`
			Value       api.EntitlementValue                      `json:"value"`
		}{
			Entitlement: p.BalanceThreshold.Entitlement,
			Feature:     p.BalanceThreshold.Feature,
			Subject:     p.BalanceThreshold.Subject,
			Threshold:   p.BalanceThreshold.Threshold,
			Value:       p.BalanceThreshold.Value,
		},
		Id:        eventId,
		Timestamp: ts,
		Type:      api.NotificationEventBalanceThresholdPayloadTypeEntitlementsBalanceThreshold,
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

const (
	EventOrderByID        = api.NotificationEventOrderById
	EventOrderByCreatedAt = api.NotificationEventOrderByCreatedAt
)

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

	OrderBy api.NotificationEventOrderBy
	Order   sortx.Order
}

func (i *ListEventsInput) Validate(_ context.Context, _ Service) error {
	if i.From.After(i.To) {
		return ValidationError{
			Err: fmt.Errorf("invalid time period: period start (%s) is after the period end (%s)", i.From, i.To),
		}
	}

	switch i.OrderBy {
	case EventOrderByID, EventOrderByCreatedAt:
	case "":
		i.OrderBy = EventOrderByID
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
	Annotations Annotations `json:"annotations,omitempty"`

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
	EventDeliveryStatusStateSuccess = EventDeliveryStatusState(api.SUCCESS)
	EventDeliveryStatusStateFailed  = EventDeliveryStatusState(api.FAILED)
	EventDeliveryStatusStateSending = EventDeliveryStatusState(api.SENDING)
	EventDeliveryStatusStatePending = EventDeliveryStatusState(api.PENDING)
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

type EventDeliveryStatusOrderBy string

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
