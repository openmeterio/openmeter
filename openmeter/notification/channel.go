package notification

import (
	"errors"
	"fmt"
	"net/url"

	webhooksecret "github.com/openmeterio/openmeter/openmeter/notification/webhook/secret"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

const (
	ChannelTypeWebhook ChannelType = "WEBHOOK"
)

var _ models.Validator = (*ChannelType)(nil)

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
		return models.NewGenericValidationError(fmt.Errorf("invalid channel type: %s", t))
	}
}

// Channel represents a notification channel with specific type and configuration.
type Channel struct {
	models.NamespacedID
	models.ManagedModel
	models.Annotations
	models.Metadata

	// Type of the notification channel (e.g. webhook)
	Type ChannelType `json:"type"`
	// Name of is the user provided name of the Channel.
	Name string `json:"name"`
	// Disabled defines whether the Channel is disabled or not.
	Disabled bool `json:"disabled"`
	// Config stores the actual Channel configuration specific to the Type.
	Config ChannelConfig `json:"config"`
}

var _ models.Validator = (*ChannelConfigMeta)(nil)

type ChannelConfigMeta struct {
	Type ChannelType `json:"type"`
}

func (m ChannelConfigMeta) Validate() error {
	return m.Type.Validate()
}

var _ models.Validator = (*ChannelConfig)(nil)

// ChannelConfig is a union type capturing configuration parameters for all type of channels.
type ChannelConfig struct {
	ChannelConfigMeta

	// WebHook
	WebHook WebHookChannelConfig `json:"webhook"`
}

// Validate invokes channel type specific validator and returns an error if channel configuration is invalid.
func (c ChannelConfig) Validate() error {
	switch c.ChannelConfigMeta.Type {
	case ChannelTypeWebhook:
		return c.WebHook.Validate()
	default:
		return models.NewGenericValidationError(fmt.Errorf("invalid channel type: %s", c.Type))
	}
}

var _ models.Validator = (*WebHookChannelConfig)(nil)

// WebHookChannelConfig defines the configuration specific to a channel with a webhook type.
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
	var errs []error

	if w.URL == "" {
		errs = append(errs, errors.New("missing URL"))
	} else if u, err := url.Parse(w.URL); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		errs = append(errs, errors.New("invalid URL: must be an absolute http(s) URL"))
	}

	if w.SigningSecret != "" {
		if err := webhooksecret.ValidateSigningSecret(w.SigningSecret); err != nil {
			errs = append(errs, fmt.Errorf("invalid signing secret: %w", err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

var (
	_ models.Validator                          = (*ListChannelsInput)(nil)
	_ models.CustomValidator[ListChannelsInput] = (*ListChannelsInput)(nil)
)

type ListChannelsInput struct {
	pagination.Page

	Namespaces []string

	OrderBy OrderBy
	Order   sortx.Order

	ID        *filter.FilterULID
	Name      *filter.FilterString
	Type      *filter.FilterString
	Disabled  *filter.FilterBoolean
	CreatedAt *filter.FilterTime
	UpdatedAt *filter.FilterTime
}

func (i ListChannelsInput) ValidateWith(validators ...models.ValidatorFunc[ListChannelsInput]) error {
	return models.Validate(i, validators...)
}

func (i ListChannelsInput) Validate() error {
	var errs []error

	// The adapter skips tenant scoping entirely when Namespaces is empty, which
	// would turn the query into a cross-tenant list (including signing secrets), so
	// an unscoped list must never reach it.
	if len(i.Namespaces) == 0 {
		errs = append(errs, errors.New("namespaces is required"))
	}

	if i.ID != nil {
		if err := i.ID.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid id filter: %w", err))
		}
	}

	if i.Name != nil {
		if err := i.Name.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid name filter: %w", err))
		}
	}

	if i.Type != nil {
		if err := i.Type.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid type filter: %w", err))
		}
	}

	if i.Disabled != nil {
		if err := i.Disabled.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid disabled filter: %w", err))
		}
	}

	if i.CreatedAt != nil {
		if err := i.CreatedAt.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid created_at filter: %w", err))
		}
	}

	if i.UpdatedAt != nil {
		if err := i.UpdatedAt.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid updated_at filter: %w", err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type ListChannelsResult = pagination.Result[Channel]

var (
	_ models.Validator                           = (*CreateChannelInput)(nil)
	_ models.CustomValidator[CreateChannelInput] = (*CreateChannelInput)(nil)
)

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
	// Metadata
	Metadata models.Metadata
	// Annotations
	Annotations models.Annotations
}

func (i CreateChannelInput) ValidateWith(validators ...models.ValidatorFunc[CreateChannelInput]) error {
	return models.Validate(i, validators...)
}

func (i CreateChannelInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if err := i.Type.Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.Name == "" {
		errs = append(errs, errors.New("channel name is required"))
	}

	if err := i.Config.Validate(); err != nil {
		errs = append(errs, err)
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

var (
	_ models.Validator                           = (*UpdateChannelInput)(nil)
	_ models.CustomValidator[UpdateChannelInput] = (*UpdateChannelInput)(nil)
)

type UpdateChannelInput struct {
	models.NamespacedID

	// Type defines the Channel type (e.g. webhook)
	Type ChannelType
	// Name stores the user-defined name of the Channel.
	Name string
	// Disabled defines whether the Channel is disabled or not. Deleted Channels are always disabled.
	Disabled bool
	// Config stores the Channel Type specific configuration.
	Config ChannelConfig
	// Metadata
	Metadata models.Metadata
	// Annotations
	Annotations models.Annotations
}

func (i UpdateChannelInput) ValidateWith(validators ...models.ValidatorFunc[UpdateChannelInput]) error {
	return models.Validate(i, validators...)
}

func (i UpdateChannelInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.ID == "" {
		errs = append(errs, errors.New("id is required"))
	}

	if err := i.Type.Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.Name == "" {
		errs = append(errs, errors.New("channel name is required"))
	}

	if err := i.Config.Validate(); err != nil {
		errs = append(errs, err)
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

var (
	_ models.Validator                        = (*GetChannelInput)(nil)
	_ models.CustomValidator[GetChannelInput] = (*GetChannelInput)(nil)
)

type GetChannelInput models.NamespacedID

func (i GetChannelInput) ValidateWith(validators ...models.ValidatorFunc[GetChannelInput]) error {
	return models.Validate(i, validators...)
}

func (i GetChannelInput) Validate() error {
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
	_ models.Validator                        = (*DeleteChannelInput)(nil)
	_ models.CustomValidator[GetChannelInput] = (*DeleteChannelInput)(nil)
)

type DeleteChannelInput = GetChannelInput
