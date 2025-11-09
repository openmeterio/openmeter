package notification

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

const (
	EventDeliveryStatusStateSuccess   EventDeliveryStatusState = "SUCCESS"
	EventDeliveryStatusStateFailed    EventDeliveryStatusState = "FAILED"
	EventDeliveryStatusStateSending   EventDeliveryStatusState = "SENDING"
	EventDeliveryStatusStatePending   EventDeliveryStatusState = "PENDING"
	EventDeliveryStatusStateResending EventDeliveryStatusState = "RESENDING"
)

var (
	_ fmt.Stringer     = (*EventDeliveryStatusState)(nil)
	_ models.Validator = (*EventDeliveryStatusState)(nil)
)

type EventDeliveryStatusState string

func (e EventDeliveryStatusState) String() string {
	return string(e)
}

func (e EventDeliveryStatusState) Validate() error {
	switch e {
	case EventDeliveryStatusStateSuccess, EventDeliveryStatusStateFailed,
		EventDeliveryStatusStateSending, EventDeliveryStatusStatePending, EventDeliveryStatusStateResending:
		return nil
	default:
		return models.NewGenericValidationError(fmt.Errorf("invalid event delivery status state: %s", e))
	}
}

func (e EventDeliveryStatusState) Values() []string {
	return []string{
		string(EventDeliveryStatusStateSuccess),
		string(EventDeliveryStatusStateFailed),
		string(EventDeliveryStatusStateSending),
		string(EventDeliveryStatusStatePending),
		string(EventDeliveryStatusStateResending),
	}
}

type EventDeliveryAttemptResponse struct {
	StatusCode *int          `json:"status_code,omitempty,omitzero"`
	Body       string        `json:"body,omitzero"`
	Duration   time.Duration `json:"duration,omitempty,omitzero"`
	URL        *string       `json:"url,omitempty,omitzero"`
}

type EventDeliveryAttempt struct {
	State     EventDeliveryStatusState     `json:"state"`
	Response  EventDeliveryAttemptResponse `json:"response"`
	Timestamp time.Time                    `json:"timestamp"`
}

// SortEventDeliveryAttemptsInDescOrder sorts the EventDeliveryAttempts in descending order by timestamp.
func SortEventDeliveryAttemptsInDescOrder(attempts []EventDeliveryAttempt) {
	slices.SortFunc(attempts, func(a, b EventDeliveryAttempt) int {
		return a.Timestamp.Compare(b.Timestamp) * -1
	})
}

type EventDeliveryStatus struct {
	models.NamespacedID

	// EventID defines the Event identifier the EventDeliveryStatus belongs to.
	EventID string

	ChannelID string
	State     EventDeliveryStatusState
	Reason    string
	CreatedAt time.Time
	UpdatedAt time.Time

	Attempts    []EventDeliveryAttempt
	NextAttempt *time.Time

	Annotations models.Annotations
}

var (
	_ models.Validator                                      = (*ListEventsDeliveryStatusInput)(nil)
	_ models.CustomValidator[ListEventsDeliveryStatusInput] = (*ListEventsDeliveryStatusInput)(nil)
)

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

func (i ListEventsDeliveryStatusInput) ValidateWith(validators ...models.ValidatorFunc[ListEventsDeliveryStatusInput]) error {
	return models.Validate(i, validators...)
}

func (i ListEventsDeliveryStatusInput) Validate() error {
	var errs []error

	if i.From.After(i.To) {
		errs = append(errs, fmt.Errorf("invalid time range: parameter From (%s) is after To (%s)", i.From, i.To))
	}

	for _, s := range i.States {
		if err := s.Validate(); err != nil {
			errs = append(errs, err)
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type ListEventsDeliveryStatusResult = pagination.Result[EventDeliveryStatus]

var (
	_ models.Validator                                    = (*GetEventDeliveryStatusInput)(nil)
	_ models.CustomValidator[GetEventDeliveryStatusInput] = (*GetEventDeliveryStatusInput)(nil)
)

type GetEventDeliveryStatusInput struct {
	models.NamespacedID

	// EventID defines the Event identifier the EventDeliveryStatus belongs to. Must be provided if ID is empty.
	EventID string
	// ChannelID defines the Channel identifier the EventDeliveryStatus belongs to. Must be provided if ID is empty.
	ChannelID string
}

func (i GetEventDeliveryStatusInput) ValidateWith(validators ...models.ValidatorFunc[GetEventDeliveryStatusInput]) error {
	return models.Validate(i, validators...)
}

func (i GetEventDeliveryStatusInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.ID == "" && (i.EventID == "" || i.ChannelID == "") {
		errs = append(errs, fmt.Errorf("delivery status ID or both channel ID and event ID must be provided"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

var (
	_ models.Validator                                       = (*UpdateEventDeliveryStatusInput)(nil)
	_ models.CustomValidator[UpdateEventDeliveryStatusInput] = (*UpdateEventDeliveryStatusInput)(nil)
)

type UpdateEventDeliveryStatusInput struct {
	models.NamespacedID

	// State is the delivery state of the Event.
	State EventDeliveryStatusState
	// Reason describes the reason for the latest State transition.
	Reason string
	// Annotations
	Annotations models.Annotations
	// NextAttempt defines the next time the Event should be attempted to be delivered.
	NextAttempt *time.Time
	// Attempts is a list of delivery attempts for the Event.
	Attempts []EventDeliveryAttempt
}

func (i UpdateEventDeliveryStatusInput) ValidateWith(validators ...models.ValidatorFunc[UpdateEventDeliveryStatusInput]) error {
	return models.Validate(i, validators...)
}

func (i UpdateEventDeliveryStatusInput) Validate() error {
	var errs []error

	if err := i.NamespacedID.Validate(); err != nil {
		errs = append(errs, err)
	}

	if err := i.State.Validate(); err != nil {
		errs = append(errs, err)
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
