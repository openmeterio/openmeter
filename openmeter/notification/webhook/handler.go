package webhook

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/notification/webhook/secret"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Webhook struct {
	Namespace string

	ID            string
	URL           string
	Secret        string
	CustomHeaders map[string]string
	Disabled      bool
	RateLimit     *uint16
	Description   string
	EventTypes    []string
	Channels      []string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

var _ models.Validator = (*ListWebhooksInput)(nil)

type ListWebhooksInput struct {
	Namespace string

	IDs        []string
	EventTypes []string
	Channels   []string
}

func (i ListWebhooksInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	return NewValidationError(errors.Join(errs...))
}

var _ models.Validator = (*CreateWebhookInput)(nil)

type CreateWebhookInput struct {
	Namespace string

	ID            *string
	URL           string
	CustomHeaders map[string]string
	Disabled      bool
	Secret        *string
	RateLimit     *uint16
	Description   *string
	EventTypes    []string
	Channels      []string
	Metadata      map[string]string
}

func (i CreateWebhookInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.URL == "" {
		errs = append(errs, errors.New("url is required"))
	}

	if i.Secret != nil && *i.Secret != "" {
		if err := secret.ValidateSigningSecret(*i.Secret); err != nil {
			errs = append(errs, fmt.Errorf("invalid secret: %w", err))
		}
	}

	return NewValidationError(errors.Join(errs...))
}

var _ models.Validator = (*UpdateWebhookInput)(nil)

type UpdateWebhookInput struct {
	Namespace string

	ID            string
	URL           string
	CustomHeaders map[string]string
	Disabled      bool
	Secret        *string
	RateLimit     *uint16
	Description   *string
	EventTypes    []string
	Channels      []string
	Metadata      map[string]string
}

func (i UpdateWebhookInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.ID == "" {
		errs = append(errs, errors.New("id is required"))
	}

	if i.URL == "" {
		errs = append(errs, errors.New("url is required"))
	}

	if i.Secret == nil {
		errs = append(errs, errors.New("secret is required"))
	} else {
		if err := secret.ValidateSigningSecret(*i.Secret); err != nil {
			errs = append(errs, fmt.Errorf("invalid secret: %w", err))
		}
	}

	return NewValidationError(errors.Join(errs...))
}

var _ models.Validator = (*UpdateWebhookChannelsInput)(nil)

type UpdateWebhookChannelsInput struct {
	Namespace string

	ID             string
	AddChannels    []string
	RemoveChannels []string
}

func (i UpdateWebhookChannelsInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.ID == "" {
		errs = append(errs, errors.New("id is required"))
	}

	return NewValidationError(errors.Join(errs...))
}

var _ models.Validator = (*GetWebhookInput)(nil)

type GetWebhookInput struct {
	Namespace string

	ID string
}

func (i GetWebhookInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.ID == "" {
		errs = append(errs, errors.New("id is required"))
	}

	return NewValidationError(errors.Join(errs...))
}

type DeleteWebhookInput = GetWebhookInput

type Payload = map[string]any

type MessageDeliveryStatus struct {
	NextAttempt *time.Time                            `json:"nextAttempt"`
	State       notification.EventDeliveryStatusState `json:"state"`
	ChannelID   string                                `json:"channel_id"`
	Attempts    []notification.EventDeliveryAttempt   `json:"attempts"`
}

type Message struct {
	Namespace string

	ID        string
	EventID   string
	EventType string
	Channels  []string

	Annotations models.Annotations

	// Expanded attributes

	// Payload stores the message payload if it was requested.
	Payload *Payload

	// DeliveryStatuses stores the message delivery status if it was requested.
	DeliveryStatuses *[]MessageDeliveryStatus

	// Timestamp when the message was created.
	Timestamp time.Time
}

var _ models.Validator = (*SendMessageInput)(nil)

type SendMessageInput struct {
	Namespace string

	EventID   string
	EventType string
	Channels  []string
	Payload   Payload
}

func (i SendMessageInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.EventID == "" {
		errs = append(errs, errors.New("event ID is required"))
	}

	if i.EventType == "" {
		errs = append(errs, errors.New("event type is required"))
	}

	if len(i.Payload) == 0 {
		errs = append(errs, errors.New("payload must not be empty"))
	}

	return NewValidationError(errors.Join(errs...))
}

type ExpandParams struct {
	// Payload stores whether the message payload for the webhook message should be included in the response or not.
	Payload bool
	// DeliveryStatusByChannelID defines whether the delivery status for the webhook message and channel should be included in the response or not.
	DeliveryStatusByChannelID string
}

var _ models.Validator = (*GetMessageInput)(nil)

type GetMessageInput struct {
	Namespace string

	ID      string
	EventID string

	Expand ExpandParams
}

func (i GetMessageInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.ID == "" && i.EventID == "" {
		errs = append(errs, errors.New("message ID or event ID must be provided"))
	}

	return NewValidationError(errors.Join(errs...))
}

var _ models.Validator = (*ResendMessageInput)(nil)

type ResendMessageInput struct {
	Namespace string

	ID        string
	EventID   string
	ChannelID string
}

func (i ResendMessageInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.ID == "" && i.EventID == "" {
		errs = append(errs, errors.New("message ID or event ID must be provided"))
	}

	if i.ChannelID == "" {
		errs = append(errs, errors.New("channel ID must be provided"))
	}

	return NewValidationError(errors.Join(errs...))
}

type RegisterEventTypesInputs struct {
	EventTypes  []EventType
	AllowUpdate bool
}

type (
	EvenTypeSchemaVersion = string
	EventTypeSchema       = interface{}
)

type EventType struct {
	Name        string
	Description string
	GroupName   string
	// Schemas defines the list of schemas for each event type version
	Schemas    map[EvenTypeSchemaVersion]EventTypeSchema
	Deprecated bool
}

type EventTypeHandler interface {
	RegisterEventTypes(ctx context.Context, params RegisterEventTypesInputs) error
}

type WebhookHandler interface {
	ListWebhooks(ctx context.Context, params ListWebhooksInput) ([]Webhook, error)
	CreateWebhook(ctx context.Context, params CreateWebhookInput) (*Webhook, error)
	UpdateWebhook(ctx context.Context, params UpdateWebhookInput) (*Webhook, error)
	UpdateWebhookChannels(ctx context.Context, params UpdateWebhookChannelsInput) (*Webhook, error)
	GetWebhook(ctx context.Context, params GetWebhookInput) (*Webhook, error)
	DeleteWebhook(ctx context.Context, params DeleteWebhookInput) error
}

type MessageHandler interface {
	SendMessage(ctx context.Context, params SendMessageInput) (*Message, error)
	GetMessage(ctx context.Context, params GetMessageInput) (*Message, error)
	ResendMessage(ctx context.Context, params ResendMessageInput) error
}

type Handler interface {
	WebhookHandler
	MessageHandler
	EventTypeHandler
}

const (
	DefaultRegistrationTimeout = 30 * time.Second
	MaxChannelsPerWebhook      = 10
)
