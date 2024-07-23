package webhook

import (
	"context"
	"encoding/base64"
	"errors"
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
	RateLimit     *int32
	Description   string
	EventTypes    []string
	Channels      []string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type validator interface {
	Validate() error
}

var _ validator = (*ListWebhooksInputs)(nil)

type ListWebhooksInputs struct {
	Namespace string

	IDs        []string
	EventTypes []string
	Channels   []string
}

func (i ListWebhooksInputs) Validate() error {
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	return nil
}

var _ validator = (*CreateWebhookInputs)(nil)

type CreateWebhookInputs struct {
	Namespace string

	ID            *string
	URL           string
	CustomHeaders map[string]string
	Disabled      bool
	Secret        *string
	RateLimit     *int32
	Description   *string
	EventTypes    []string
	Channels      []string
}

func (i CreateWebhookInputs) Validate() error {
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	if i.URL == "" {
		return errors.New("url is required")
	}

	if i.Secret != nil {
		secret, _ := strings.CutPrefix(*i.Secret, SigningSecretPrefix)
		if _, err := base64.StdEncoding.DecodeString(secret); err != nil {
			return errors.New("invalid secret: must be base64 encoded")
		}
	}

	return nil
}

var _ validator = (*UpdateWebhookInputs)(nil)

type UpdateWebhookInputs struct {
	Namespace string

	ID            string
	URL           string
	CustomHeaders map[string]string
	Disabled      bool
	Secret        *string
	RateLimit     *int32
	Description   *string
	EventTypes    []string
	Channels      []string
}

func (i UpdateWebhookInputs) Validate() error {
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	if i.ID == "" {
		return errors.New("id is required")
	}

	if i.URL == "" {
		return errors.New("url is required")
	}

	if i.Secret == nil {
		return errors.New("secret is required")
	} else {
		secret, _ := strings.CutPrefix(*i.Secret, SigningSecretPrefix)
		if _, err := base64.StdEncoding.DecodeString(secret); err != nil {
			return errors.New("invalid secret: must be base64 encoded")
		}
	}

	return nil
}

type UpdateWebhookChannelsInputs struct {
	Namespace string

	ID             string
	AddChannels    []string
	RemoveChannels []string
}

func (i UpdateWebhookChannelsInputs) Validate() error {
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	if i.ID == "" {
		return errors.New("id is required")
	}

	return nil
}

var _ validator = (*GetWebhookInputs)(nil)

type GetWebhookInputs struct {
	Namespace string

	ID string
}

func (i GetWebhookInputs) Validate() error {
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	if i.ID == "" {
		return errors.New("id is required")
	}

	return nil
}

type DeleteWebhookInputs = GetWebhookInputs

type Message struct {
	Namespace string

	ID        string
	EventID   string
	EventType string
	Channels  []string
	Payload   map[string]interface{}
}

var _ validator = (*SendMessageInputs)(nil)

type SendMessageInputs struct {
	Namespace string

	EventID   string
	EventType string
	Channels  []string
	Payload   map[string]interface{}
}

func (i SendMessageInputs) Validate() error {
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	if i.EventType == "" {
		return errors.New("event type is required")
	}

	if len(i.Payload) == 0 {
		return errors.New("payload must not be empty")
	}

	return nil
}

type RegisterEventTypesInputs struct {
	EvenTypes   []EventType
	AllowUpdate bool
}

type EventType struct {
	Name        string
	Description string
	GroupName   string
	// Schemas defines the list of schemas for each event type version
	Schemas map[string]map[string]interface{}
}

type Handler interface {
	RegisterEventTypes(ctx context.Context, params RegisterEventTypesInputs) error
	ListWebhooks(ctx context.Context, params ListWebhooksInputs) ([]Webhook, error)
	CreateWebhook(ctx context.Context, params CreateWebhookInputs) (*Webhook, error)
	UpdateWebhook(ctx context.Context, params UpdateWebhookInputs) (*Webhook, error)
	UpdateWebhookChannels(ctx context.Context, params UpdateWebhookChannelsInputs) (*Webhook, error)
	GetWebhook(ctx context.Context, params GetWebhookInputs) (*Webhook, error)
	DeleteWebhook(ctx context.Context, params DeleteWebhookInputs) error
	SendMessage(ctx context.Context, params SendMessageInputs) (*Message, error)
}

type Config = svixConfig

func NewHandler(config Config) (Handler, error) {
	return newSvixWebhookHandler(config)
}
