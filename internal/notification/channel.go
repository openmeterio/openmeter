package notification

import (
	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/pkg/models"
)

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

const (
	ChannelTypeWebhook = ChannelType(api.WEBHOOK)
)

type ChannelType api.NotificationChannelType

func (t ChannelType) Values() []string {
	return []string{
		string(ChannelTypeWebhook),
	}
}

type ChannelConfigMeta struct {
	Type ChannelType `json:"type"`
}

// ChannelConfig is a union type capturing configuration parameters for all type of channels.
type ChannelConfig struct {
	ChannelConfigMeta

	// WebHook
	WebHook WebHookChannelConfig `json:"webhook"`
}

// WebHookChannelConfig defines the configuration specific to channel with webhook type.
type WebHookChannelConfig struct {
	// CustomHeaders stores a set of HTTP headers which are applied to the outgoing webhook message.
	CustomHeaders map[string]interface{} `json:"customHeaders,omitempty"`
	// URL is the webhook endpoint url where the messages are sent to.
	URL string `json:"url"`
	// SigningSecret defines the secret which can be used for validating the signature of the message sent
	// to the webhook endpoint.
	SigningSecret string `json:"signingSecret"`
}
