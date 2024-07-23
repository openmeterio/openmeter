package notification

import (
	"context"
	"fmt"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type payloadObjectMapper interface {
	AsNotificationEventBalanceThresholdPayload() api.NotificationEventBalanceThresholdPayload
}

var _ payloadObjectMapper = (*Event)(nil)

type Event struct {
	models.NamespacedModel

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
	}

	event := api.NotificationEvent{
		CreatedAt:      e.CreatedAt,
		DeliveryStatus: deliveryStatuses,
		Id:             e.ID,
		Rule:           rule,
	}

	switch e.Type {
	case EventTypeBalanceThreshold:
		err = event.Payload.FromNotificationEventBalanceThresholdPayload(e.AsNotificationEventBalanceThresholdPayload())
		if err != nil {
			return event, ValidationError{
				Err: fmt.Errorf("invalid event type: %s", e.Type),
			}
		}
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
		Type:      api.NotificationEventType(e.Type),
		Data: struct {
			Balance     api.EntitlementValue                      `json:"balance"`
			Entitlement api.EntitlementMetered                    `json:"entitlement"`
			Feature     api.Feature                               `json:"feature"`
			Subject     api.Subject                               `json:"subject"`
			Threshold   api.NotificationRuleBalanceThresholdValue `json:"threshold"`
		}(struct {
			Balance     api.EntitlementValue
			Entitlement api.EntitlementMetered
			Feature     api.Feature
			Subject     api.Subject
			Threshold   api.NotificationRuleBalanceThresholdValue
		}{
			Balance:     e.Payload.BalanceThreshold.Balance,
			Entitlement: e.Payload.BalanceThreshold.Entitlement,
			Feature:     e.Payload.BalanceThreshold.Feature,
			Subject:     e.Payload.BalanceThreshold.Subject,
			Threshold:   e.Payload.BalanceThreshold.Threshold,
		}),
	}
}

const (
	EventTypeBalanceThreshold = EventType(api.EntitlementsBalanceThreshold)
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

func (c EventPayload) Validate() error {
	switch c.Type {
	case EventTypeBalanceThreshold:
		return c.BalanceThreshold.Validate()
	default:
		return ValidationError{
			Err: fmt.Errorf("invalid event type: %s", c.Type),
		}
	}
}

type BalanceThresholdPayload struct {
	Entitlement api.EntitlementMetered                    `json:"entitlement"`
	Feature     api.Feature                               `json:"feature"`
	Subject     api.Subject                               `json:"subject"`
	Balance     api.EntitlementValue                      `json:"balance"`
	Threshold   api.NotificationRuleBalanceThresholdValue `json:"threshold"`
}

// Validate returns an error if balance threshold payload is invalid.
func (b BalanceThresholdPayload) Validate() error {
	return nil
}

const (
	EventOrderByID        = api.ListNotificationEventsParamsOrderById
	EventOrderByCreatedAt = api.ListNotificationEventsParamsOrderByCreatedAt
)

var _ validator = (*ListEventsInput)(nil)

type ListEventsInput struct {
	pagination.Page

	Namespaces []string

	OrderBy api.ListNotificationEventsParamsOrderBy

	From time.Time
	To   time.Time

	SubjectFilter []string
	FeatureFilter []string
}

func (i *ListEventsInput) Validate(_ context.Context, _ Connector) error {
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

func (i GetEventInput) Validate(_ context.Context, _ Connector) error {
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

	if _, err := ulid.Parse(i.ID); err != nil {
		return ValidationError{
			Err: fmt.Errorf("invalid ULID provided as event id: %w", err),
		}
	}

	return nil
}

var _ validator = (*CreateEventInput)(nil)

type CreateEventInput struct {
	models.NamespacedModel

	// Type of the notification Event (e.g. entitlements.balance.threshold)
	Type EventType `json:"type"`
	// CreatedAt Timestamp when the notification event was created.
	CreatedAt time.Time `json:"createdAt"`
	// Payload is the actual payload sent to Channel as part of the notification Event.
	Payload EventPayload `json:"payload"`
	// Rule defines the notification Rule that generated this Event.
	Rule Rule `json:"rule"`
}

func (i CreateEventInput) Validate(ctx context.Context, connector Connector) error {
	if err := i.Type.Validate(); err != nil {
		return err
	}

	if err := i.Rule.Validate(ctx, connector); err != nil {
		return err
	}

	return nil
}

const (
	EventDeliveryStatusStateSuccess = EventDeliveryStatusState(api.SUCCESS)
	EventDeliveryStatusStateFailed  = EventDeliveryStatusState(api.FAILED)
	EventDeliveryStatusStateSending = EventDeliveryStatusState(api.SENDING)
)

type EventDeliveryStatusState string

func (e EventDeliveryStatusState) Validate() error {
	switch e {
	case EventDeliveryStatusStateSuccess, EventDeliveryStatusStateFailed, EventDeliveryStatusStateSending:
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
	}
}

type EventDeliveryStatusOrderBy string

type EventDeliveryStatus struct {
	models.NamespacedModel

	// ID defines the Event identifier the EventDeliveryStatus belongs to.
	EventID string `json:"eventId"`

	ChannelID string                   `json:"channelId"`
	State     EventDeliveryStatusState `json:"state"`
	UpdatedAt time.Time                `json:"updatedAt"`
}

var _ validator = (*ListEventsDeliveryStatusInput)(nil)

type ListEventsDeliveryStatusInput struct {
	pagination.Page

	Namespaces []string

	From time.Time
	To   time.Time

	EventIDs   []string
	ChannelIDs []string
}

func (i ListEventsDeliveryStatusInput) Validate(_ context.Context, _ Connector) error {
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

	// ID defines the Event identifier the EventDeliveryStatus belongs to.
	EventID string `json:"eventId"`
}

func (i GetEventDeliveryStatusInput) Validate(_ context.Context, _ Connector) error {
	return nil
}

var _ validator = (*CreateEventDeliveryStatusInput)(nil)

type CreateEventDeliveryStatusInput struct {
	models.NamespacedModel

	// ID defines the Event identifier the EventDeliveryStatus belongs to.
	EventID   string                   `json:"eventId"`
	State     EventDeliveryStatusState `json:"state"`
	ChannelID string                   `json:"channelId"`
	Timestamp time.Time                `json:"timestamp"`
}

func (i CreateEventDeliveryStatusInput) Validate(_ context.Context, _ Connector) error {
	if i.Namespace == "" {
		return ValidationError{
			Err: fmt.Errorf("namespace must be provided"),
		}
	}

	if err := i.State.Validate(); err != nil {
		return err
	}

	return nil
}

// GetCreatedAtFromEventID returns the timestamp when the Event was created diverged from the Event.ID
// which must be in ULID format. It returns an error if Event.ID is not a valid ULID.
// Note: it helps with scoping the query used for fetching Event from database.
func GetCreatedAtFromEventID(eventID string) (time.Time, error) {
	id, err := ulid.Parse(eventID)
	if err != nil {
		return time.Time{}, ValidationError{
			Err: fmt.Errorf("failed to parse event id %q: %w", eventID, err),
		}
	}

	createdAt := time.UnixMilli(int64(id.Time()))

	return createdAt, nil
}
