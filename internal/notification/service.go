package notification

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/internal/notification/webhook"
	"github.com/openmeterio/openmeter/internal/productcatalog"
)

type Service interface {
	ChannelService
}

type ChannelService interface {
	ListChannels(ctx context.Context, params ListChannelsInput) (ListChannelsResult, error)
	CreateChannel(ctx context.Context, params CreateChannelInput) (*Channel, error)
	DeleteChannel(ctx context.Context, params DeleteChannelInput) error
	GetChannel(ctx context.Context, params GetChannelInput) (*Channel, error)
	UpdateChannel(ctx context.Context, params UpdateChannelInput) (*Channel, error)
}

type FeatureService interface {
	ListFeature(ctx context.Context, namespace string, features ...string) ([]productcatalog.Feature, error)
}

var _ Service = (*service)(nil)

type service struct {
	repo    Repository
	webhook webhook.Handler
}

type Config struct {
	Repository Repository
	Webhook    webhook.Handler
}

func New(config Config) (Service, error) {
	if config.Repository == nil {
		return nil, errors.New("missing repository")
	}

	if config.Webhook == nil {
		return nil, errors.New("missing webhook handler")
	}

	return &service{
		repo:    config.Repository,
		webhook: config.Webhook,
	}, nil
}

func (c service) ListChannels(ctx context.Context, params ListChannelsInput) (ListChannelsResult, error) {
	if err := params.Validate(ctx, c); err != nil {
		return ListChannelsResult{}, fmt.Errorf("invalid params: %w", err)
	}

	return c.repo.ListChannels(ctx, params)
}

func (c service) CreateChannel(ctx context.Context, params CreateChannelInput) (*Channel, error) {
	if err := params.Validate(ctx, c); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	// FIXME: this must be in transaction

	channel, err := c.repo.CreateChannel(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create channel: %w", err)
	}

	switch params.Type {
	case ChannelTypeWebhook:
		var headers map[string]string
		headers, err = interfaceMapToStringMap(channel.Config.WebHook.CustomHeaders)
		if err != nil {
			return nil, fmt.Errorf("failed to cast custom headers: %w", err)
		}

		var wb *webhook.Webhook
		wb, err = c.webhook.CreateWebhook(ctx, webhook.CreateWebhookInput{
			Namespace:     params.Namespace,
			ID:            &channel.ID,
			URL:           channel.Config.WebHook.URL,
			CustomHeaders: headers,
			Disabled:      channel.Disabled,
			Secret:        &channel.Config.WebHook.SigningSecret,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create webhook for channel: %w", err)
		}

		updateIn := UpdateChannelInput{
			NamespacedModel: channel.NamespacedModel,
			Type:            channel.Type,
			Name:            channel.Name,
			Disabled:        channel.Disabled,
			Config:          channel.Config,
			ID:              channel.ID,
		}
		updateIn.Config.WebHook.SigningSecret = wb.Secret

		channel, err = c.repo.UpdateChannel(ctx, updateIn)
		if err != nil {
			return nil, fmt.Errorf("failed to update channel: %w", err)
		}
	default:
		return nil, fmt.Errorf("invalid channel type: %s", channel.Type)
	}

	return channel, nil
}

func (c service) DeleteChannel(ctx context.Context, params DeleteChannelInput) error {
	if err := params.Validate(ctx, c); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}

	return c.repo.DeleteChannel(ctx, params)
}

func (c service) GetChannel(ctx context.Context, params GetChannelInput) (*Channel, error) {
	if err := params.Validate(ctx, c); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	return c.repo.GetChannel(ctx, params)
}

func (c service) UpdateChannel(ctx context.Context, params UpdateChannelInput) (*Channel, error) {
	if err := params.Validate(ctx, c); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	channel, err := c.repo.GetChannel(ctx, GetChannelInput{
		ID:        params.ID,
		Namespace: params.Namespace,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get channel: %w", err)
	}

	if channel.DeletedAt != nil {
		return nil, UpdateAfterDeleteError{
			Err: errors.New("not allowed to update deleted channel"),
		}
	}

	// FIXME: this must to be in transaction

	channel, err = c.repo.UpdateChannel(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create channel: %w", err)
	}

	switch params.Type {
	case ChannelTypeWebhook:
		var headers map[string]string
		headers, err = interfaceMapToStringMap(channel.Config.WebHook.CustomHeaders)
		if err != nil {
			return nil, fmt.Errorf("failed to cast custom headers: %w", err)
		}

		_, err = c.webhook.UpdateWebhook(ctx, webhook.UpdateWebhookInput{
			Namespace:     params.Namespace,
			ID:            channel.ID,
			URL:           channel.Config.WebHook.URL,
			CustomHeaders: headers,
			Disabled:      channel.Disabled,
			Secret:        &channel.Config.WebHook.SigningSecret,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to update webhook for channel: %w", err)
		}
	default:
		return nil, fmt.Errorf("invalid channel type: %s", channel.Type)
	}

	return channel, nil
}

func interfaceMapToStringMap(m map[string]interface{}) (map[string]string, error) {
	var s map[string]string
	if len(m) > 0 {
		s = make(map[string]string, len(m))
		for k, v := range m {
			switch t := v.(type) {
			case string:
				s[k] = t
			case fmt.Stringer:
				s[k] = t.String()
			default:
				return s, fmt.Errorf("failed to cast value with %T to string", t)
			}
		}
	}

	return s, nil
}
