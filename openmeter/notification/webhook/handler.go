package webhook

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	SigningSecretPrefix = "whsec_"
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
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	return nil
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
	if i.Namespace == "" {
		return ValidationError{
			Err: errors.New("namespace is required"),
		}
	}

	if i.URL == "" {
		return ValidationError{
			Err: errors.New("url is required"),
		}
	}

	if i.Secret != nil && *i.Secret != "" {
		if err := ValidateSigningSecret(*i.Secret); err != nil {
			return ValidationError{
				Err: fmt.Errorf("invalid secret: %w", err),
			}
		}
	}

	return nil
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
	if i.Namespace == "" {
		return ValidationError{
			Err: errors.New("namespace is required"),
		}
	}

	if i.ID == "" {
		return ValidationError{
			Err: errors.New("id is required"),
		}
	}

	if i.URL == "" {
		return ValidationError{
			Err: errors.New("url is required"),
		}
	}

	if i.Secret == nil {
		return ValidationError{
			Err: errors.New("secret is required"),
		}
	} else {
		secret, _ := strings.CutPrefix(*i.Secret, SigningSecretPrefix)
		if _, err := base64.StdEncoding.DecodeString(secret); err != nil {
			return ValidationError{
				Err: errors.New("invalid secret: must be base64 encoded"),
			}
		}
	}

	return nil
}

var _ validator = (*UpdateWebhookChannelsInput)(nil)

type UpdateWebhookChannelsInput struct {
	Namespace string

	ID             string
	AddChannels    []string
	RemoveChannels []string
}

func (i UpdateWebhookChannelsInput) Validate() error {
	if i.Namespace == "" {
		return ValidationError{
			Err: errors.New("namespace is required"),
		}
	}

	if i.ID == "" {
		return ValidationError{
			Err: errors.New("id is required"),
		}
	}

	return nil
}

var _ validator = (*GetWebhookInput)(nil)

type GetWebhookInput struct {
	Namespace string

	ID string
}

func (i GetWebhookInput) Validate() error {
	if i.Namespace == "" {
		return ValidationError{
			Err: errors.New("namespace is required"),
		}
	}

	if i.ID == "" {
		return ValidationError{
			Err: errors.New("id is required"),
		}
	}

	return nil
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
	if i.Namespace == "" {
		return ValidationError{
			Err: errors.New("namespace is required"),
		}
	}

	if i.EventType == "" {
		return ValidationError{
			Err: errors.New("event type is required"),
		}
	}

	if len(i.Payload) == 0 {
		return ValidationError{
			Err: errors.New("payload must not be empty"),
		}
	}

	return nil
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

func ValidateSigningSecret(secret string) error {
	s, _ := strings.CutPrefix(secret, SigningSecretPrefix)
	if len(s) < 32 || len(s) > 100 {
		return errors.New("secret length must be between 32 to 100 chars without the optional prefix")
	}

	if _, err := base64.StdEncoding.DecodeString(s); err != nil {
		return errors.New("invalid base64 string")
	}

	return nil
}
