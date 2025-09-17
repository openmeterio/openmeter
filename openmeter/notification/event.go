package notification

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

var eventTypes = []EventType{
	EventTypeBalanceThreshold,
	EventTypeEntitlementReset,
	EventTypeInvoiceCreated,
	EventTypeInvoiceUpdated,
}

func EventTypes() []EventType {
	return eventTypes
}

type EventType string

func (t EventType) Validate() error {
	if !lo.Contains(eventTypes, t) {
		return ValidationError{
			Err: fmt.Errorf("invalid notification event type: %q", t),
		}
	}

	return nil
}

func (t EventType) Values() []string {
	return lo.Map(eventTypes, func(item EventType, index int) string {
		return string(item)
	})
}

type Event struct {
	models.NamespacedModel
	models.Annotations

	// ID is the unique identifier for Event.
	ID string `json:"id"`
	// Type of the notification Event (e.g. entitlements.balance.threshold)
	// TODO(chrisgacsal): this is redundant as it is always the same as the payload type. Deprecate this field.
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

type ListEventsResult = pagination.Result[Event]

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
