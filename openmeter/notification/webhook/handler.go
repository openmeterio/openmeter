package webhook

import (
	"context"
	"errors"
	"fmt"
	"time"
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

type validator interface {
	Validate() error
}

var _ validator = (*ListWebhooksInput)(nil)

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

var _ validator = (*CreateWebhookInput)(nil)

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
		if err := ValidateSigningSecret(*i.Secret); err != nil {
			errs = append(errs, fmt.Errorf("invalid secret: %w", err))
		}
	}

	return NewValidationError(errors.Join(errs...))
}

var _ validator = (*UpdateWebhookInput)(nil)

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
		if err := ValidateSigningSecret(*i.Secret); err != nil {
			errs = append(errs, fmt.Errorf("invalid secret: %w", err))
		}
	}

	return NewValidationError(errors.Join(errs...))
}

var _ validator = (*UpdateWebhookChannelsInput)(nil)

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

var _ validator = (*GetWebhookInput)(nil)

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

type Message struct {
	Namespace string

	ID        string
	EventID   string
	EventType string
	Channels  []string
	Payload   map[string]interface{}
}

var _ validator = (*SendMessageInput)(nil)

type SendMessageInput struct {
	Namespace string

	EventID   string
	EventType string
	Channels  []string
	Payload   map[string]interface{}
}

func (i SendMessageInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.EventType == "" {
		errs = append(errs, errors.New("event type is required"))
	}

	if len(i.Payload) == 0 {
		errs = append(errs, errors.New("payload must not be empty"))
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
}

type Handler interface {
	WebhookHandler
	MessageHandler
	EventTypeHandler
}

const (
	DefaultRegistrationTimeout = 30 * time.Second
)
