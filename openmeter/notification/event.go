package notification

import (
	"errors"
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

var (
	_ fmt.Stringer     = (*EventType)(nil)
	_ models.Validator = (*EventType)(nil)
)

type EventType string

func (t EventType) String() string {
	return string(t)
}

func (t EventType) Validate() error {
	if !lo.Contains(eventTypes, t) {
		return models.NewGenericValidationError(fmt.Errorf("invalid notification event type: %q", t))
	}

	return nil
}

func (t EventType) Values() []string {
	return lo.Map(eventTypes, func(item EventType, index int) string {
		return string(item)
	})
}

type Event struct {
	models.NamespacedID
	models.Annotations

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

var (
	_ models.Validator                        = (*ListEventsInput)(nil)
	_ models.CustomValidator[ListEventsInput] = (*ListEventsInput)(nil)
)

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

	NextAttemptBefore time.Time `json:"nextAttemptBefore,omitempty"`

	OrderBy OrderBy
	Order   sortx.Order
}

func (i ListEventsInput) ValidateWith(validators ...models.ValidatorFunc[ListEventsInput]) error {
	return models.Validate(i, validators...)
}

func (i ListEventsInput) Validate() error {
	var errs []error

	if i.From.After(i.To) {
		errs = append(errs, fmt.Errorf("invalid time period: period start (%s) is after the period end (%s)", i.From, i.To))
	}

	switch i.OrderBy {
	case OrderByID, OrderByCreatedAt, "":
	default:
		errs = append(errs, fmt.Errorf("invalid event order_by: %s", i.OrderBy))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type ListEventsResult = pagination.Result[Event]

var (
	_ models.Validator                      = (*GetEventInput)(nil)
	_ models.CustomValidator[GetEventInput] = (*GetEventInput)(nil)
)

type GetEventInput models.NamespacedID

func (i GetEventInput) ValidateWith(validators ...models.ValidatorFunc[GetEventInput]) error {
	return models.Validate(i, validators...)
}

func (i GetEventInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.ID == "" {
		errs = append(errs, errors.New("id is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

var (
	_ models.Validator                         = (*CreateEventInput)(nil)
	_ models.CustomValidator[CreateEventInput] = (*CreateEventInput)(nil)
)

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

func (i CreateEventInput) ValidateWith(validators ...models.ValidatorFunc[CreateEventInput]) error {
	return models.Validate(i, validators...)
}

func (i CreateEventInput) Validate() error {
	var errs []error

	if err := i.Type.Validate(); err != nil {
		errs = append(errs, err)
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

var (
	_ models.Validator                         = (*ResendEventInput)(nil)
	_ models.CustomValidator[ResendEventInput] = (*ResendEventInput)(nil)
)

type ResendEventInput struct {
	models.NamespacedID

	Channels []string `json:"channels,omitempty"`
}

func (i ResendEventInput) ValidateWith(validators ...models.ValidatorFunc[ResendEventInput]) error {
	return models.Validate(i, validators...)
}

func (i ResendEventInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.ID == "" {
		errs = append(errs, errors.New("id is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
