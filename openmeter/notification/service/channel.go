package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/notification/webhook"
	"github.com/openmeterio/openmeter/pkg/convert"
)

func (s Service) ListChannels(ctx context.Context, params notification.ListChannelsInput) (notification.ListChannelsResult, error) {
	if err := params.Validate(ctx, s); err != nil {
		return notification.ListChannelsResult{}, fmt.Errorf("invalid params: %w", err)
	}

	return s.repo.ListChannels(ctx, params)
}

func (s Service) CreateChannel(ctx context.Context, params notification.CreateChannelInput) (*notification.Channel, error) {
	if err := params.Validate(ctx, s); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	logger := s.logger.WithGroup("channel").With(
		"operation", "create",
		"namespace", params.Namespace,
	)

	logger.Debug("creating channel", "type", params.Type)

	txFunc := func(ctx context.Context, repo notification.TxRepository) (*notification.Channel, error) {
		channel, err := repo.CreateChannel(ctx, params)
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
				Description: convert.ToPointer("Notification Channel: " + channel.ID),
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create webhook for channel: %w", err)
			}

			logger.Debug("webhook is created")

			updateIn := notification.UpdateChannelInput{
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

	return notification.WithTx[*notification.Channel](ctx, s.repo, txFunc)
}

func (s Service) DeleteChannel(ctx context.Context, params notification.DeleteChannelInput) error {
	if err := params.Validate(ctx, s); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}

	logger := s.logger.WithGroup("channel").With(
		"operation", "delete",
		"id", params.ID,
		"namespace", params.Namespace,
	)

	logger.Debug("deleting channel")

	rules, err := s.repo.ListRules(ctx, notification.ListRulesInput{
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

		return notification.ValidationError{
			Err: fmt.Errorf("cannot delete channel as it is assigned to one or more rules: %v", ruleIDs),
		}
	}

	txFunc := func(ctx context.Context, repo notification.TxRepository) error {
		if err := s.webhook.DeleteWebhook(ctx, webhook.DeleteWebhookInput{
			Namespace: params.Namespace,
			ID:        params.ID,
		}); err != nil {
			return fmt.Errorf("failed to delete webhook: %w", err)
		}

		logger.Debug("webhook associated with channel deleted")

		return repo.DeleteChannel(ctx, params)
	}

	return notification.WithTxNoValue(ctx, s.repo, txFunc)
}

func (s Service) GetChannel(ctx context.Context, params notification.GetChannelInput) (*notification.Channel, error) {
	if err := params.Validate(ctx, s); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	return s.repo.GetChannel(ctx, params)
}

func (s Service) UpdateChannel(ctx context.Context, params notification.UpdateChannelInput) (*notification.Channel, error) {
	if err := params.Validate(ctx, s); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	logger := s.logger.WithGroup("channel").With(
		"operation", "update",
		"id", params.ID,
		"namespace", params.Namespace,
	)

	logger.Debug("updating channel")

	channel, err := s.repo.GetChannel(ctx, notification.GetChannelInput{
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

	txFunc := func(ctx context.Context, repo notification.TxRepository) (*notification.Channel, error) {
		// Fetch rules assigned to channel as we need to make sure that we do not remove rule assignments
		// from channel during update.
		rules, err := s.repo.ListRules(ctx, notification.ListRulesInput{
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

		channel, err = repo.UpdateChannel(ctx, params)
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
				Description: convert.ToPointer("Notification Channel: " + channel.ID),
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

	return notification.WithTx[*notification.Channel](ctx, s.repo, txFunc)
}
