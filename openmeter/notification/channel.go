package notification

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/notification/webhook"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

const (
	ChannelTypeWebhook ChannelType = "WEBHOOK"
)

type ChannelType string

func (t ChannelType) Values() []string {
	return []string{
		string(ChannelTypeWebhook),
	}
}

func (t ChannelType) Validate() error {
	switch t {
	case ChannelTypeWebhook:
		return nil
	default:
		return ValidationError{
			Err: fmt.Errorf("invalid channel type: %s", t),
		}
	}
}

// Channel represents a notification channel with specific type and configuration.
type Channel struct {
	models.NamespacedModel
	models.ManagedModel

	// ID is the unique identifier for Channel.
	ID string `json:"id"`
	// Type of the notification channel (e.g. webhook)
	Type ChannelType `json:"type"`
	// Name of is the user provided name of the Channel.
	Name string `json:"name"`
	// Disabled defines whether the Channel is disabled or not.
	Disabled bool `json:"disabled"`
	// Config stores the actual Channel configuration specific to the Type.
	Config ChannelConfig `json:"config"`
}

type ChannelConfigMeta struct {
	Type ChannelType `json:"type"`
}

func (m ChannelConfigMeta) Validate() error {
	return m.Type.Validate()
}

// ChannelConfig is a union type capturing configuration parameters for all type of channels.
type ChannelConfig struct {
	ChannelConfigMeta

	// WebHook
	WebHook WebHookChannelConfig `json:"webhook"`
}

// Validate invokes channel type specific validator and returns an error if channel configuration is invalid.
func (c ChannelConfig) Validate() error {
	switch c.Type {
	case ChannelTypeWebhook:
		return c.WebHook.Validate()
	default:
		return ValidationError{
			Err: fmt.Errorf("invalid channel type: %s", c.Type),
		}
	}
}

// WebHookChannelConfig defines the configuration specific to channel with webhook type.
type WebHookChannelConfig struct {
	// CustomHeaders stores a set of HTTP headers which are applied to the outgoing webhook message.
	CustomHeaders map[string]string `json:"customHeaders,omitempty"`
	// URL is the webhook endpoint url where the messages are sent to.
	URL string `json:"url"`
	// SigningSecret defines the secret which can be used for validating the signature of the message sent
	// to the webhook endpoint.
	SigningSecret string `json:"signingSecret"`
}

// Validate returns an error if webhook channel configuration is invalid.
func (w WebHookChannelConfig) Validate() error {
	if w.URL == "" {
		return ValidationError{
			Err: errors.New("missing URL"),
		}
	}

	if w.SigningSecret != "" {
		if err := webhook.ValidateSigningSecret(w.SigningSecret); err != nil {
			return ValidationError{
				Err: fmt.Errorf("invalid signing secret: %w", err),
			}
		}
	}

	return nil
}

var _ validator = (*ListChannelsInput)(nil)

type ListChannelsInput struct {
	pagination.Page

	Namespaces      []string
	Channels        []string
	IncludeDisabled bool

	OrderBy OrderBy
	Order   sortx.Order
}

func (i ListChannelsInput) Validate(_ context.Context, _ Service) error {
	return nil
}

type ListChannelsResult = pagination.Result[Channel]

var _ validator = (*CreateChannelInput)(nil)

type CreateChannelInput struct {
	models.NamespacedModel

	// Type defines the Channel type (e.g. webhook)
	Type ChannelType
	// Name stores the user defined name of the Channel.
	Name string
	// Disabled defines whether the Channel is disabled or not. Deleted Channels are always disabled.
	Disabled bool
	// Config stores the Channel Type specific configuration.
	Config ChannelConfig
}

func (i CreateChannelInput) Validate(_ context.Context, _ Service) error {
	if i.Namespace == "" {
		return ValidationError{
			Err: errors.New("namespace is required"),
		}
	}

	if err := i.Type.Validate(); err != nil {
		return err
	}

	if i.Name == "" {
		return ValidationError{
			Err: errors.New("channel name is required"),
		}
	}

	if err := i.Config.Validate(); err != nil {
		return err
	}

	return nil
}

var _ validator = (*UpdateChannelInput)(nil)

type UpdateChannelInput struct {
	models.NamespacedModel

	// Type defines the Channel type (e.g. webhook)
	Type ChannelType
	// Name stores the user defined name of the Channel.
	Name string
	// Disabled defines whether the Channel is disabled or not. Deleted Channels are always disabled.
	Disabled bool
	// Config stores the Channel Type specific configuration.
	Config ChannelConfig

	// ID is the unique identifier for Channel.
	ID string
}

func (i UpdateChannelInput) Validate(_ context.Context, _ Service) error {
	if i.Namespace == "" {
		return ValidationError{
			Err: errors.New("namespace is required"),
		}
	}

	if err := i.Type.Validate(); err != nil {
		return err
	}

	if i.Name == "" {
		return ValidationError{
			Err: errors.New("channel name is required"),
		}
	}

	if err := i.Config.Validate(); err != nil {
		return err
	}

	if i.ID == "" {
		return ValidationError{
			Err: errors.New("channel id is required"),
		}
	}

	return nil
}

var _ validator = (*GetChannelInput)(nil)

type GetChannelInput models.NamespacedID

func (i GetChannelInput) Validate(_ context.Context, _ Service) error {
	if i.Namespace == "" {
		return ValidationError{
			Err: errors.New("namespace is required"),
		}
	}

	if i.ID == "" {
		return ValidationError{
			Err: errors.New("channel id is required"),
		}
	}

	return nil
}

var _ validator = (*DeleteChannelInput)(nil)

type DeleteChannelInput models.NamespacedID

func (i DeleteChannelInput) Validate(_ context.Context, _ Service) error {
	if i.Namespace == "" {
		return ValidationError{
			Err: errors.New("namespace is required"),
		}
	}

	if i.ID == "" {
		return ValidationError{
			Err: errors.New("channel id is required"),
		}
	}

	return nil
}
