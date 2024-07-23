package notification

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/openmeterio/openmeter/internal/notification/webhook"
	"github.com/openmeterio/openmeter/internal/productcatalog"
)

type Connector interface {
	FeatureConnector

	ChannelConnector
	RuleConnector
	EventConnector
}

type ChannelConnector interface {
	ListChannels(ctx context.Context, params ListChannelsInput) (ListChannelsResult, error)
	CreateChannel(ctx context.Context, params CreateChannelInput) (*Channel, error)
	DeleteChannel(ctx context.Context, params DeleteChannelInput) error
	GetChannel(ctx context.Context, params GetChannelInput) (*Channel, error)
	UpdateChannel(ctx context.Context, params UpdateChannelInput) (*Channel, error)
}

type RuleConnector interface {
	ListRules(ctx context.Context, params ListRulesInput) (ListRulesResult, error)
	CreateRule(ctx context.Context, params CreateRuleInput) (*Rule, error)
	DeleteRule(ctx context.Context, params DeleteRuleInput) error
	GetRule(ctx context.Context, params GetRuleInput) (*Rule, error)
	UpdateRule(ctx context.Context, params UpdateRuleInput) (*Rule, error)
}

type EventConnector interface {
	ListEvents(ctx context.Context, params ListEventsInput) (ListEventsResult, error)
	GetEvent(ctx context.Context, params GetEventInput) (*Event, error)
	CreateEvent(ctx context.Context, params CreateEventInput) (*Event, error)
	ListEventsDeliveryStatus(ctx context.Context, params ListEventsDeliveryStatusInput) (ListEventsDeliveryStatusResult, error)
	GetEventDeliveryStatus(ctx context.Context, params GetEventDeliveryStatusInput) (*EventDeliveryStatus, error)
	CreateEventDeliveryStatus(ctx context.Context, params CreateEventDeliveryStatusInput) (*EventDeliveryStatus, error)
}

type FeatureConnector interface {
	ListFeature(ctx context.Context, namespace string, features ...string) ([]productcatalog.Feature, error)
}

var _ Connector = (*connector)(nil)

type connector struct {
	feature productcatalog.FeatureConnector

	repo    Repository
	webhook webhook.Handler
}

type ConnectorConfig struct {
	Repository Repository

	FeatureConnector productcatalog.FeatureConnector
	Webhook          webhook.Handler

	Logger *slog.Logger
}

func NewConnector(config ConnectorConfig) (Connector, error) {
	if config.Repository == nil {
		return nil, errors.New("missing repository")
	}

	if config.FeatureConnector == nil {
		return nil, errors.New("missing feature connector")
	}

	if config.Webhook == nil {
		return nil, errors.New("missing webhook connector")
	}

	return &connector{
		repo:    config.Repository,
		feature: config.FeatureConnector,
		webhook: config.Webhook,
	}, nil
}

func (c connector) ListFeature(ctx context.Context, namespace string, features ...string) ([]productcatalog.Feature, error) {
	resp, err := c.feature.ListFeatures(ctx, productcatalog.ListFeaturesParams{
		IDsOrKeys:       features,
		Namespace:       namespace,
		MeterSlugs:      nil,
		IncludeArchived: false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get features: %w", err)
	}

	return resp.Items, nil
}

func (c connector) ListChannels(ctx context.Context, params ListChannelsInput) (ListChannelsResult, error) {
	if err := params.Validate(ctx, c); err != nil {
		return ListChannelsResult{}, fmt.Errorf("invalid params: %w", err)
	}

	return c.repo.ListChannels(ctx, params)
}

func (c connector) CreateChannel(ctx context.Context, params CreateChannelInput) (*Channel, error) {
	if err := params.Validate(ctx, c); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	// FIXME: this should be in transaction

	channel, err := c.repo.CreateChannel(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create channel: %w", err)
	}

	switch params.Type {
	case ChannelTypeWebhook:
		_, err = c.webhook.CreateWebhook(ctx, webhook.CreateWebhookInputs{
			Namespace:     params.Namespace,
			ID:            &channel.ID,
			URL:           channel.Config.WebHook.URL,
			CustomHeaders: nil, // FIXME
			Disabled:      channel.Disabled,
			Secret:        &channel.Config.WebHook.SigningSecret,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create webhook for channel: %w", err)
		}
	default:
		return nil, fmt.Errorf("invalid channel type: %s", channel.Type)
	}

	return channel, nil
}

func (c connector) DeleteChannel(ctx context.Context, params DeleteChannelInput) error {
	if err := params.Validate(ctx, c); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}

	return c.repo.DeleteChannel(ctx, params)
}

func (c connector) GetChannel(ctx context.Context, params GetChannelInput) (*Channel, error) {
	if err := params.Validate(ctx, c); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	return c.repo.GetChannel(ctx, params)
}

func (c connector) UpdateChannel(ctx context.Context, params UpdateChannelInput) (*Channel, error) {
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

	// FIXME: this needs to be in transaction

	channel, err = c.repo.UpdateChannel(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create channel: %w", err)
	}

	switch params.Type {
	case ChannelTypeWebhook:
		_, err = c.webhook.UpdateWebhook(ctx, webhook.UpdateWebhookInputs{
			Namespace:     params.Namespace,
			ID:            channel.ID,
			URL:           channel.Config.WebHook.URL,
			CustomHeaders: nil, // FIXME
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

func (c connector) ListRules(ctx context.Context, params ListRulesInput) (ListRulesResult, error) {
	if err := params.Validate(ctx, c); err != nil {
		return ListRulesResult{}, fmt.Errorf("invalid params: %w", err)
	}

	return c.repo.ListRules(ctx, params)
}

func (c connector) CreateRule(ctx context.Context, params CreateRuleInput) (*Rule, error) {
	if err := params.Validate(ctx, c); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	// FIXME: transaction

	rule, err := c.repo.CreateRule(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create rule: %w", err)
	}

	for _, channel := range rule.Channels {
		switch channel.Type {
		case ChannelTypeWebhook:
			_, err = c.webhook.UpdateWebhookChannels(ctx, webhook.UpdateWebhookChannelsInputs{
				Namespace: params.Namespace,
				ID:        channel.ID,
				AddChannels: []string{
					rule.ID,
				},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to update webhook for channel: %w", err)
			}
		default:
			return nil, fmt.Errorf("invalid channel type: %s", channel.Type)
		}
	}

	return rule, nil
}

func (c connector) DeleteRule(ctx context.Context, params DeleteRuleInput) error {
	if err := params.Validate(ctx, c); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}

	rule, err := c.repo.GetRule(ctx, GetRuleInput{
		Namespace: params.Namespace,
		ID:        params.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to get rule: %w", err)
	}

	for _, channel := range rule.Channels {
		switch channel.Type {
		case ChannelTypeWebhook:
			_, err = c.webhook.UpdateWebhookChannels(ctx, webhook.UpdateWebhookChannelsInputs{
				Namespace: params.Namespace,
				ID:        channel.ID,
				RemoveChannels: []string{
					rule.ID,
				},
			})
			if err != nil {
				return fmt.Errorf("failed to update webhook for channel: %w", err)
			}
		default:
			return fmt.Errorf("invalid channel type: %s", channel.Type)
		}
	}

	return c.repo.DeleteRule(ctx, params)
}

func (c connector) GetRule(ctx context.Context, params GetRuleInput) (*Rule, error) {
	if err := params.Validate(ctx, c); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	return c.repo.GetRule(ctx, params)
}

func (c connector) UpdateRule(ctx context.Context, params UpdateRuleInput) (*Rule, error) {
	if err := params.Validate(ctx, c); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	channel, err := c.repo.GetRule(ctx, GetRuleInput{
		ID:        params.ID,
		Namespace: params.Namespace,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get rule: %w", err)
	}

	if channel.DeletedAt != nil {
		return nil, UpdateAfterDeleteError{
			Err: errors.New("not allowed to update deleted rule"),
		}
	}

	return c.repo.UpdateRule(ctx, params)
}

func (c connector) ListEvents(ctx context.Context, params ListEventsInput) (ListEventsResult, error) {
	if err := params.Validate(ctx, c); err != nil {
		return ListEventsResult{}, fmt.Errorf("invalid params: %w", err)
	}

	return c.repo.ListEvents(ctx, params)
}

func (c connector) GetEvent(ctx context.Context, params GetEventInput) (*Event, error) {
	if err := params.Validate(ctx, c); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	return c.repo.GetEvent(ctx, params)
}

func (c connector) CreateEvent(ctx context.Context, params CreateEventInput) (*Event, error) {
	if err := params.Validate(ctx, c); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	return c.repo.CreateEvent(ctx, params)
}

func (c connector) ListEventsDeliveryStatus(ctx context.Context, params ListEventsDeliveryStatusInput) (ListEventsDeliveryStatusResult, error) {
	if err := params.Validate(ctx, c); err != nil {
		return ListEventsDeliveryStatusResult{}, fmt.Errorf("invalid params: %w", err)
	}

	return c.repo.ListEventsDeliveryStatus(ctx, params)
}

func (c connector) GetEventDeliveryStatus(ctx context.Context, params GetEventDeliveryStatusInput) (*EventDeliveryStatus, error) {
	if err := params.Validate(ctx, c); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	return c.repo.GetEventDeliveryStatus(ctx, params)
}

func (c connector) CreateEventDeliveryStatus(ctx context.Context, params CreateEventDeliveryStatusInput) (*EventDeliveryStatus, error) {
	if err := params.Validate(ctx, c); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	return c.repo.CreateEventDeliveryStatus(ctx, params)
}
