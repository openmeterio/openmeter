package notification

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/notification/webhook"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type Service interface {
	FeatureService

	ChannelService
	RuleService
	EventService

	Close() error
}

type ChannelService interface {
	ListChannels(ctx context.Context, params ListChannelsInput) (ListChannelsResult, error)
	CreateChannel(ctx context.Context, params CreateChannelInput) (*Channel, error)
	DeleteChannel(ctx context.Context, params DeleteChannelInput) error
	GetChannel(ctx context.Context, params GetChannelInput) (*Channel, error)
	UpdateChannel(ctx context.Context, params UpdateChannelInput) (*Channel, error)
}

type RuleService interface {
	ListRules(ctx context.Context, params ListRulesInput) (ListRulesResult, error)
	CreateRule(ctx context.Context, params CreateRuleInput) (*Rule, error)
	DeleteRule(ctx context.Context, params DeleteRuleInput) error
	GetRule(ctx context.Context, params GetRuleInput) (*Rule, error)
	UpdateRule(ctx context.Context, params UpdateRuleInput) (*Rule, error)
}

type EventService interface {
	ListEvents(ctx context.Context, params ListEventsInput) (ListEventsResult, error)
	GetEvent(ctx context.Context, params GetEventInput) (*Event, error)
	CreateEvent(ctx context.Context, params CreateEventInput) (*Event, error)
	ListEventsDeliveryStatus(ctx context.Context, params ListEventsDeliveryStatusInput) (ListEventsDeliveryStatusResult, error)
	GetEventDeliveryStatus(ctx context.Context, params GetEventDeliveryStatusInput) (*EventDeliveryStatus, error)
	UpdateEventDeliveryStatus(ctx context.Context, params UpdateEventDeliveryStatusInput) (*EventDeliveryStatus, error)
}

type FeatureService interface {
	ListFeature(ctx context.Context, namespace string, features ...string) ([]productcatalog.Feature, error)
}

var _ Service = (*service)(nil)

const (
	ChannelIDMetadataKey = "om-channel-id"
)

type service struct {
	feature productcatalog.FeatureConnector

	repo    Repository
	webhook webhook.Handler

	eventHandler EventHandler

	logger *slog.Logger
}

func (c service) Close() error {
	return c.eventHandler.Close()
}

type Config struct {
	FeatureConnector productcatalog.FeatureConnector

	Repository Repository
	Webhook    webhook.Handler

	Logger *slog.Logger
}

func New(config Config) (Service, error) {
	if config.Repository == nil {
		return nil, errors.New("missing repository")
	}

	if config.FeatureConnector == nil {
		return nil, errors.New("missing feature connector")
	}

	if config.Webhook == nil {
		return nil, errors.New("missing webhook handler")
	}

	if config.Logger == nil {
		return nil, errors.New("missing logger")
	}

	eventHandler, err := NewEventHandler(EventHandlerConfig{
		Repository: config.Repository,
		Webhook:    config.Webhook,
		Logger:     config.Logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize notification event handler: %w", err)
	}

	if err = eventHandler.Start(); err != nil {
		return nil, fmt.Errorf("failed to initialize notification event handler: %w", err)
	}

	return &service{
		repo:         config.Repository,
		feature:      config.FeatureConnector,
		webhook:      config.Webhook,
		eventHandler: eventHandler,
		logger:       config.Logger,
	}, nil
}

func (c service) ListFeature(ctx context.Context, namespace string, features ...string) ([]productcatalog.Feature, error) {
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

	logger := c.logger.WithGroup("channel").With(
		"operation", "create",
		"namespace", params.Namespace,
	)

	logger.Debug("creating channel", "type", params.Type)

	txFunc := func(ctx context.Context, repo TxRepository) (*Channel, error) {
		channel, err := repo.CreateChannel(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to create channel: %w", err)
		}

		logger = logger.With("id", channel.ID)

		logger.Debug("channel stored in repository")

		switch params.Type {
		case ChannelTypeWebhook:
			var headers map[string]string
			headers, err = StrictInterfaceMapToStringMap(channel.Config.WebHook.CustomHeaders)
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
				Metadata: map[string]string{
					ChannelIDMetadataKey: channel.ID,
				},
				Description: convert.ToPointer("Notification Channel: " + channel.ID),
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create webhook for channel: %w", err)
			}

			logger.Debug("webhook is created")

			updateIn := UpdateChannelInput{
				NamespacedModel: channel.NamespacedModel,
				Type:            channel.Type,
				Name:            channel.Name,
				Disabled:        channel.Disabled,
				Config:          channel.Config,
				ID:              channel.ID,
			}
			updateIn.Config.WebHook.SigningSecret = wb.Secret

			channel, err = repo.UpdateChannel(ctx, updateIn)
			if err != nil {
				return nil, fmt.Errorf("failed to update channel: %w", err)
			}
			logger.Debug("channel is updated in database with webhook configuration")
		default:
			return nil, fmt.Errorf("invalid channel type: %s", channel.Type)
		}

		return channel, nil
	}

	return WithTx[*Channel](ctx, c.repo, txFunc)
}

func (c service) DeleteChannel(ctx context.Context, params DeleteChannelInput) error {
	if err := params.Validate(ctx, c); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}

	logger := c.logger.WithGroup("channel").With(
		"operation", "delete",
		"id", params.ID,
		"namespace", params.Namespace,
	)

	logger.Debug("deleting channel")

	rules, err := c.repo.ListRules(ctx, ListRulesInput{
		Namespaces:      []string{params.Namespace},
		IncludeDisabled: true,
		Channels:        []string{params.ID},
	})
	if err != nil {
		return fmt.Errorf("failed to list rules for channel: %w", err)
	}

	if rules.TotalCount > 0 {
		ruleIDs := make([]string, 0, len(rules.Items))

		for _, rule := range rules.Items {
			ruleIDs = append(ruleIDs, rule.ID)
		}

		return ValidationError{
			Err: fmt.Errorf("cannot delete channel as it is assigned to one or more rules: %v", ruleIDs),
		}
	}

	txFunc := func(ctx context.Context, repo TxRepository) error {
		if err := c.webhook.DeleteWebhook(ctx, webhook.DeleteWebhookInput{
			Namespace: params.Namespace,
			ID:        params.ID,
		}); err != nil {
			return fmt.Errorf("failed to delete webhook: %w", err)
		}

		logger.Debug("webhook associated with channel deleted")

		return repo.DeleteChannel(ctx, params)
	}

	return WithTxNoValue(ctx, c.repo, txFunc)
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

	logger := c.logger.WithGroup("channel").With(
		"operation", "update",
		"id", params.ID,
		"namespace", params.Namespace,
	)

	logger.Debug("updating channel")

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

	txFunc := func(ctx context.Context, repo TxRepository) (*Channel, error) {
		channel, err = repo.UpdateChannel(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to create channel: %w", err)
		}

		logger.Debug("channel updated in repository")

		switch params.Type {
		case ChannelTypeWebhook:
			var headers map[string]string
			headers, err = StrictInterfaceMapToStringMap(channel.Config.WebHook.CustomHeaders)
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
				Metadata: map[string]string{
					ChannelIDMetadataKey: channel.ID,
				},
				Description: convert.ToPointer("Notification Channel: " + channel.ID),
			})
			if err != nil {
				return nil, fmt.Errorf("failed to update webhook for channel: %w", err)
			}

			logger.Debug("webhook is updated")

		default:
			return nil, fmt.Errorf("invalid channel type: %s", channel.Type)
		}

		return channel, nil
	}

	return WithTx[*Channel](ctx, c.repo, txFunc)
}

func (c service) ListRules(ctx context.Context, params ListRulesInput) (ListRulesResult, error) {
	if err := params.Validate(ctx, c); err != nil {
		return ListRulesResult{}, fmt.Errorf("invalid params: %w", err)
	}

	return c.repo.ListRules(ctx, params)
}

func (c service) CreateRule(ctx context.Context, params CreateRuleInput) (*Rule, error) {
	if err := params.Validate(ctx, c); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	logger := c.logger.WithGroup("rule").With(
		"operation", "create",
		"namespace", params.Namespace,
	)

	logger.Debug("creating rule", "type", params.Type)

	txFunc := func(ctx context.Context, repo TxRepository) (*Rule, error) {
		rule, err := repo.CreateRule(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to create rule: %w", err)
		}

		for _, channel := range rule.Channels {
			switch channel.Type {
			case ChannelTypeWebhook:
				_, err = c.webhook.UpdateWebhookChannels(ctx, webhook.UpdateWebhookChannelsInput{
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

	return WithTx[*Rule](ctx, c.repo, txFunc)
}

func (c service) DeleteRule(ctx context.Context, params DeleteRuleInput) error {
	if err := params.Validate(ctx, c); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}

	txFunc := func(ctx context.Context, repo TxRepository) error {
		rule, err := c.repo.GetRule(ctx, GetRuleInput(params))
		if err != nil {
			return fmt.Errorf("failed to get rule: %w", err)
		}

		for _, channel := range rule.Channels {
			switch channel.Type {
			case ChannelTypeWebhook:
				_, err = c.webhook.UpdateWebhookChannels(ctx, webhook.UpdateWebhookChannelsInput{
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

	return WithTxNoValue(ctx, c.repo, txFunc)
}

func (c service) GetRule(ctx context.Context, params GetRuleInput) (*Rule, error) {
	if err := params.Validate(ctx, c); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	return c.repo.GetRule(ctx, params)
}

func (c service) UpdateRule(ctx context.Context, params UpdateRuleInput) (*Rule, error) {
	if err := params.Validate(ctx, c); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	logger := c.logger.WithGroup("rule").With(
		"operation", "update",
		"id", params.ID,
		"namespace", params.Namespace,
	)

	logger.Debug("updating rule")

	rule, err := c.repo.GetRule(ctx, GetRuleInput{
		ID:        params.ID,
		Namespace: params.Namespace,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get rule: %w", err)
	}

	if rule.DeletedAt != nil {
		return nil, UpdateAfterDeleteError{
			Err: errors.New("not allowed to update deleted rule"),
		}
	}

	// Get list of channel IDs currently assigned to rule
	oldChannelIDs := lo.Map(rule.Channels, func(channel Channel, _ int) string {
		return channel.ID
	})
	logger.Debug("currently assigned channels", "channels", oldChannelIDs)

	// Calculate channels diff for the update
	channelIDsDiff := NewChannelIDsDifference(params.Channels, oldChannelIDs)

	logger.WithGroup("channels").Debug("difference in channels assignment",
		"changed", channelIDsDiff.HasChanged(),
		"additions", channelIDsDiff.Additions(),
		"removals", channelIDsDiff.Removals(),
	)

	// We can return early ff there is no change in the list of channels assigned to rule.
	if !channelIDsDiff.HasChanged() {
		return c.repo.UpdateRule(ctx, params)
	}

	txFunc := func(ctx context.Context, repo TxRepository) (*Rule, error) {
		// Fetch all the channels from repo which are either added or removed from rule
		channels, err := repo.ListChannels(ctx, ListChannelsInput{
			Page: pagination.Page{
				// In order to avoid under-fetching. There cannot be more affected channels than
				// twice as the maximum number of allowed channels per rule.
				PageSize:   2 * MaxChannelsPerRule,
				PageNumber: 1,
			},
			Namespaces:      []string{params.Namespace},
			Channels:        channelIDsDiff.All(),
			IncludeDisabled: true,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list channels for rule: %w", err)
		}
		logger.Debug("fetched all affected channels", "channels", channels.Items)

		// Update affected channels
		for _, channel := range channels.Items {
			switch channel.Type {
			case ChannelTypeWebhook:
				input := webhook.UpdateWebhookChannelsInput{
					Namespace: params.Namespace,
					ID:        channel.ID,
				}

				if channelIDsDiff.InAdditions(channel.ID) {
					input.AddChannels = []string{rule.ID}
				}

				if channelIDsDiff.InRemovals(channel.ID) {
					input.RemoveChannels = []string{rule.ID}
				}

				logger.Debug("updating webhook for channel", "id", channel.ID, "input", input)

				_, err = c.webhook.UpdateWebhookChannels(ctx, input)
				if err != nil {
					return nil, fmt.Errorf("failed to update webhook for channel: %w", err)
				}
			default:
				return nil, fmt.Errorf("invalid channel type: %s", channel.Type)
			}
		}

		return c.repo.UpdateRule(ctx, params)
	}

	return WithTx[*Rule](ctx, c.repo, txFunc)
}

func (c service) ListEvents(ctx context.Context, params ListEventsInput) (ListEventsResult, error) {
	if err := params.Validate(ctx, c); err != nil {
		return ListEventsResult{}, fmt.Errorf("invalid params: %w", err)
	}

	return c.repo.ListEvents(ctx, params)
}

func (c service) GetEvent(ctx context.Context, params GetEventInput) (*Event, error) {
	if err := params.Validate(ctx, c); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	return c.repo.GetEvent(ctx, params)
}

func (c service) CreateEvent(ctx context.Context, params CreateEventInput) (*Event, error) {
	if err := params.Validate(ctx, c); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	logger := c.logger.WithGroup("event").With(
		"operation", "create",
		"namespace", params.Namespace,
	)

	logger.Debug("creating event")

	rule, err := c.repo.GetRule(ctx, GetRuleInput{
		Namespace: params.Namespace,
		ID:        params.RuleID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get rule: %w", err)
	}

	if rule.DeletedAt != nil {
		return nil, NotFoundError{
			NamespacedID: models.NamespacedID{
				Namespace: params.Namespace,
				ID:        params.RuleID,
			},
		}
	}

	if rule.Disabled {
		return nil, ValidationError{
			Err: errors.New("failed to send event: rule is disabled"),
		}
	}

	event, err := c.repo.CreateEvent(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	if err = c.eventHandler.Dispatch(event); err != nil {
		return nil, fmt.Errorf("failed to dispatch event: %w", err)
	}

	return event, nil
}

func (c service) UpdateEventDeliveryStatus(ctx context.Context, params UpdateEventDeliveryStatusInput) (*EventDeliveryStatus, error) {
	if err := params.Validate(ctx, c); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	return c.repo.UpdateEventDeliveryStatus(ctx, params)
}

func (c service) ListEventsDeliveryStatus(ctx context.Context, params ListEventsDeliveryStatusInput) (ListEventsDeliveryStatusResult, error) {
	if err := params.Validate(ctx, c); err != nil {
		return ListEventsDeliveryStatusResult{}, fmt.Errorf("invalid params: %w", err)
	}

	return c.repo.ListEventsDeliveryStatus(ctx, params)
}

func (c service) GetEventDeliveryStatus(ctx context.Context, params GetEventDeliveryStatusInput) (*EventDeliveryStatus, error) {
	if err := params.Validate(ctx, c); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	return c.repo.GetEventDeliveryStatus(ctx, params)
}
