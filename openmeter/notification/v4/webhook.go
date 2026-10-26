package v4

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	webhooksecret "github.com/openmeterio/openmeter/openmeter/notification/webhook/secret"
	"github.com/openmeterio/openmeter/pkg/models"
)

type WebhookProvider Provider[WebhookConfig, WebhookConfigUpdate]

const ChannelTypeWebhook ChannelType = "webhook"

var _ ChannelConfig = (*WebhookConfig)(nil)

// WebhookConfig describes webhook delivery independently of its provider.
type WebhookConfig struct {
	// CustomHeaders stores a set of HTTP headers which are applied to the outgoing webhook message.
	CustomHeaders map[string]string `json:"custom_headers,omitempty"`

	// URL is the webhook endpoint url where the messages are sent to.
	URL string `json:"url"`

	// SigningSecret defines the secret which can be used for validating the signature of the message sent
	// to the webhook endpoint. Empty requests generation on creation.
	SigningSecret string `json:"signing_secret,omitempty"`
}

// Validate checks the endpoint URL and any explicitly supplied signing secret.
func (w WebhookConfig) Validate() error {
	var errs []error

	if u, err := url.Parse(w.URL); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		errs = append(errs, errors.New("invalid endpoint URL: must be an absolute http(s) URL"))
	}

	if w.SigningSecret != "" {
		if err := webhooksecret.ValidateSigningSecret(w.SigningSecret); err != nil {
			errs = append(errs, fmt.Errorf("invalid signing secret: %w", err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (WebhookConfig) isInternal() {}

// WebhookConfigUpdate changes only explicitly supplied webhook configuration fields.
type WebhookConfigUpdate struct {
	URL *string `json:"url,omitempty"`

	// CustomHeaders replaces all headers when provided. A pointer to a nil or
	// empty map removes all headers; nil preserves them.
	// In JSON, explicit null or an empty object removes all headers; omission preserves them.
	CustomHeaders *map[string]string `json:"custom_headers,omitempty"`

	// SigningSecret sets an explicit secret. Nil preserves the current key;
	// empty values are invalid and do not request secret generation.
	SigningSecret *string `json:"signing_secret,omitempty"`
}

var _ ChannelConfigUpdate = WebhookConfigUpdate{}

// UnmarshalJSON preserves the distinction between omitted headers and
// explicit null, which requests removal of all headers.
func (w *WebhookConfigUpdate) UnmarshalJSON(data []byte) error {
	type wire WebhookConfigUpdate

	var decoded wire

	input := struct {
		*wire
		CustomHeaders json.RawMessage `json:"custom_headers"`
	}{
		wire: &decoded,
	}

	if err := json.Unmarshal(data, &input); err != nil {
		return fmt.Errorf("invalid webhook configuration update: %w", err)
	}

	if input.CustomHeaders != nil {
		var headers map[string]string

		if err := json.Unmarshal(input.CustomHeaders, &headers); err != nil {
			return fmt.Errorf("invalid custom headers: %w", err)
		}

		decoded.CustomHeaders = &headers
	}

	*w = WebhookConfigUpdate(decoded)

	return nil
}

// Validate checks only supplied fields, allowing independent configuration updates.
func (w WebhookConfigUpdate) Validate() error {
	var errs []error

	if w.URL != nil {
		u, err := url.Parse(*w.URL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			errs = append(errs, errors.New("invalid endpoint URL: must be an absolute http(s) URL"))
		}
	}

	if w.SigningSecret != nil {
		if err := webhooksecret.ValidateSigningSecret(*w.SigningSecret); err != nil {
			errs = append(errs, fmt.Errorf("invalid signing secret: %w", err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (WebhookConfigUpdate) isChannelConfigUpdate() {}

// WebhookChannel binds shared channel attributes to webhook configuration.
type WebhookChannel channel[WebhookConfig]

var _ Channel = (*WebhookChannel)(nil)

// ChannelType identifies webhook channels independently of their configuration.
func (WebhookChannel) ChannelType() ChannelType {
	return ChannelTypeWebhook
}
