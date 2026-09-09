package notification

import (
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/filter"
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

	OrderBy OrderBy
	Order   sortx.Order

	// Filters backed by columns on the notification_event table.
	ID        *filter.FilterULID   `json:"id,omitempty"`
	Type      *filter.FilterString `json:"type,omitempty"`
	CreatedAt *filter.FilterTime   `json:"createdAt,omitempty"`
	RuleID    *filter.FilterULID   `json:"ruleId,omitempty"`

	// Filters resolved through edges. Both are existential: ChannelID matches events
	// whose generating rule targets the channel, DeliveryStatus matches events with at
	// least one delivery status in the given state. Negation is therefore not
	// expressible and is rejected by the API layer rather than silently mismatched.
	ChannelID      *filter.FilterString `json:"channelId,omitempty"`
	DeliveryStatus *filter.FilterString `json:"deliveryStatus,omitempty"`

	// Filters resolved against JSONB annotation keys. Missing annotations behave like
	// NULL columns: negated operators do not match events without the annotation.
	SubjectKey *filter.FilterString `json:"subjectKey,omitempty"`
	SubjectID  *filter.FilterULID   `json:"subjectId,omitempty"`
	FeatureKey *filter.FilterString `json:"featureKey,omitempty"`
	FeatureID  *filter.FilterULID   `json:"featureId,omitempty"`

	// Internal-only filters used by the delivery reconciliation loop and the event
	// deduplication path. Not exposed on any HTTP API.
	DeduplicationHashes []string  `json:"deduplicationHashes,omitempty"`
	NextAttemptBefore   time.Time `json:"nextAttemptBefore,omitempty"`
}

func (i ListEventsInput) ValidateWith(validators ...models.ValidatorFunc[ListEventsInput]) error {
	return models.Validate(i, validators...)
}

// Validate deliberately does not require Namespaces, unlike ListChannelsInput. The
// delivery reconciliation loop (notification/eventhandler) lists events across all
// tenants to find deliveries due for retry, so an unscoped list is a legitimate
// internal use. Tenant scoping for API traffic is applied by the HTTP handlers.
func (i ListEventsInput) Validate() error {
	var errs []error

	switch i.OrderBy {
	case OrderByID, OrderByType, OrderByCreatedAt, "":
	default:
		errs = append(errs, fmt.Errorf("invalid event order_by: %s", i.OrderBy))
	}

	if i.ID != nil {
		if err := i.ID.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid id filter: %w", err))
		}
	}

	if i.Type != nil {
		if err := i.Type.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid type filter: %w", err))
		}
	}

	if i.CreatedAt != nil {
		if err := i.CreatedAt.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid created_at filter: %w", err))
		}
	}

	if i.RuleID != nil {
		if err := i.RuleID.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid rule_id filter: %w", err))
		}
	}

	if i.ChannelID != nil {
		if err := i.ChannelID.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid channel_id filter: %w", err))
		}
	}

	if i.DeliveryStatus != nil {
		if err := i.DeliveryStatus.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid delivery_status filter: %w", err))
		}
	}

	if i.SubjectKey != nil {
		if err := i.SubjectKey.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid subject_key filter: %w", err))
		}
	}

	if i.SubjectID != nil {
		if err := i.SubjectID.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid subject_id filter: %w", err))
		}
	}

	if i.FeatureKey != nil {
		if err := i.FeatureKey.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid feature_key filter: %w", err))
		}
	}

	if i.FeatureID != nil {
		if err := i.FeatureID.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid feature_id filter: %w", err))
		}
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
