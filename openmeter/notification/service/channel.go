package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/notification/webhook"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (s Service) ListChannels(ctx context.Context, params notification.ListChannelsInput) (notification.ListChannelsResult, error) {
	if err := params.Validate(); err != nil {
		return notification.ListChannelsResult{}, fmt.Errorf("invalid params: %w", err)
	}

	return s.adapter.ListChannels(ctx, params)
}

func (s Service) CreateChannel(ctx context.Context, params notification.CreateChannelInput) (*notification.Channel, error) {
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	fn := func(ctx context.Context) (*notification.Channel, error) {
		logger := s.logger.WithGroup("channel").With(
			"operation", "create",
			"namespace", params.Namespace,
		)

		logger.Debug("creating channel", "type", params.Type)

		channel, err := s.adapter.CreateChannel(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to create channel: %w", err)
		}

		logger = logger.With("id", channel.ID)

		logger.Debug("channel stored in repository")

		switch params.Type {
		case notification.ChannelTypeWebhook:
			var wb *webhook.Webhook
			wb, err = s.webhook.CreateWebhook(ctx, webhook.CreateWebhookInput{
				Namespace:     params.Namespace,
				ID:            &channel.ID,
				URL:           channel.Config.WebHook.URL,
				CustomHeaders: channel.Config.WebHook.CustomHeaders,
				Disabled:      channel.Disabled,
				Secret:        &channel.Config.WebHook.SigningSecret,
				Metadata: map[string]string{
					ChannelIDMetadataKey: channel.ID,
				},
				Description: lo.ToPtr("Notification Channel: " + channel.ID),
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create webhook for channel: %w", err)
			}

			logger.Debug("webhook is created")

			updateIn := notification.UpdateChannelInput{
				NamespacedID: models.NamespacedID{
					Namespace: params.Namespace,
					ID:        channel.ID,
				},
				Type:        channel.Type,
				Name:        channel.Name,
				Disabled:    channel.Disabled,
				Config:      channel.Config,
				Annotations: channel.Annotations,
				Metadata:    channel.Metadata,
			}
			updateIn.Config.WebHook.SigningSecret = wb.Secret

			channel, err = s.adapter.UpdateChannel(ctx, updateIn)
			if err != nil {
				return nil, fmt.Errorf("failed to update channel: %w", err)
			}
			logger.Debug("channel is updated in database with webhook configuration")
		default:
			return nil, fmt.Errorf("invalid channel type: %s", channel.Type)
		}

		return channel, nil
	}

	return transaction.Run(ctx, s.adapter, fn)
}

func (s Service) DeleteChannel(ctx context.Context, params notification.DeleteChannelInput) error {
	if err := params.Validate(); err != nil {
		return fmt.Errorf("invalid delete channel params: %w", err)
	}

	fn := func(ctx context.Context) error {
		logger := s.logger.WithGroup("channel").
			With(
				"operation", "delete",
				"id", params.ID,
				"namespace", params.Namespace,
			)

		logger.Debug("deleting channel")

		rules, err := s.adapter.ListRules(ctx, notification.ListRulesInput{
			Namespaces:      []string{params.Namespace},
			IncludeDisabled: true,
			Channels:        []string{params.ID},
		})
		if err != nil {
			return fmt.Errorf("failed to list rules for channel [namespace=%s channel.id=%s]: %w",
				params.Namespace, params.ID, err)
		}

		if rules.TotalCount > 0 {
			ruleIDs := make([]string, 0, len(rules.Items))

			for _, rule := range rules.Items {
				ruleIDs = append(ruleIDs, rule.ID)
			}

			return models.NewGenericValidationError(
				fmt.Errorf("failed to delete channel as it is assigned to one or more rules [namespace=%s channel.id=%s]: %v",
					params.Namespace, params.ID, ruleIDs),
			)
		}

		if err = s.webhook.DeleteWebhook(ctx, webhook.DeleteWebhookInput{
			Namespace: params.Namespace,
			ID:        params.ID,
		}); err != nil {
			return fmt.Errorf("failed to delete webhook [namespace=%s channel.id=%s]: %w", params.Namespace, params.ID, err)
		}

		logger.Debug("webhook associated with channel deleted")

		return s.adapter.DeleteChannel(ctx, params)
	}

	return transaction.RunWithNoValue(ctx, s.adapter, fn)
}

func (s Service) GetChannel(ctx context.Context, params notification.GetChannelInput) (*notification.Channel, error) {
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	return s.adapter.GetChannel(ctx, params)
}

func (s Service) UpdateChannel(ctx context.Context, params notification.UpdateChannelInput) (*notification.Channel, error) {
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	fn := func(ctx context.Context) (*notification.Channel, error) {
		logger := s.logger.WithGroup("channel").With(
			"operation", "update",
			"id", params.ID,
			"namespace", params.Namespace,
		)

		logger.Debug("updating channel")

		channel, err := s.adapter.GetChannel(ctx, notification.GetChannelInput{
			ID:        params.ID,
			Namespace: params.Namespace,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get channel: %w", err)
		}

		if channel.DeletedAt != nil {
			return nil, notification.UpdateAfterDeleteError{
				Err: errors.New("not allowed to update deleted channel"),
			}
		}

		err = params.ValidateWith(func(i notification.UpdateChannelInput) error {
			if i.Type != channel.Type {
				return fmt.Errorf("cannot update channel type: %s to %s", channel.Type, i.Type)
			}

			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}

		// Fetch rules assigned to channel as we need to make sure that we do not remove rule assignments
		// from channel during update.
		rules, err := s.adapter.ListRules(ctx, notification.ListRulesInput{
			Namespaces:      []string{params.Namespace},
			IncludeDisabled: true,
			Channels:        []string{params.ID},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list rules for channel: %w", err)
		}

		ruleIDs := make([]string, 0, len(rules.Items))
		for _, rule := range rules.Items {
			ruleIDs = append(ruleIDs, rule.ID)
		}

		channel, err = s.adapter.UpdateChannel(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to create channel: %w", err)
		}

		logger.Debug("channel updated in repository")

		switch params.Type {
		case notification.ChannelTypeWebhook:
			_, err = s.webhook.UpdateWebhook(ctx, webhook.UpdateWebhookInput{
				Namespace:     params.Namespace,
				ID:            channel.ID,
				URL:           channel.Config.WebHook.URL,
				CustomHeaders: channel.Config.WebHook.CustomHeaders,
				Disabled:      channel.Disabled,
				Secret:        &channel.Config.WebHook.SigningSecret,
				Metadata: map[string]string{
					ChannelIDMetadataKey: channel.ID,
				},
				Description: lo.ToPtr("Notification Channel: " + channel.ID),
				Channels:    ruleIDs,
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

	return transaction.Run(ctx, s.adapter, fn)
}
