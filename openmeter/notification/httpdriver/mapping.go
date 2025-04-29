package httpdriver

import (
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/models"
)

func FromChannel(c notification.Channel) (api.NotificationChannel, error) {
	var channel api.NotificationChannel

	switch c.Type {
	case notification.ChannelTypeWebhook:
		channel = FromChannelWebhook(c)
	default:
		return channel, notification.ValidationError{
			Err: fmt.Errorf("invalid channel type: %s", c.Type),
		}
	}

	return channel, nil
}

func FromChannelWebhook(c notification.Channel) api.NotificationChannelWebhook {
	return api.NotificationChannelWebhook{
		CreatedAt: c.CreatedAt,
		CustomHeaders: convert.SafeDeRef(&c.Config.WebHook.CustomHeaders, func(m map[string]string) *map[string]string {
			if len(m) > 0 {
				return &m
			}

			return nil
		}),
		Disabled:      lo.ToPtr(c.Disabled),
		Id:            c.ID,
		Name:          c.Name,
		SigningSecret: lo.ToPtr(c.Config.WebHook.SigningSecret),
		Type:          api.NotificationChannelWebhookTypeWEBHOOK,
		UpdatedAt:     c.UpdatedAt,
		Url:           c.Config.WebHook.URL,
		DeletedAt:     c.DeletedAt,
	}
}

func AsChannelWebhookCreateRequest(r api.NotificationChannelWebhookCreateRequest, namespace string) notification.CreateChannelInput {
	return notification.CreateChannelInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: namespace,
		},
		Name:     r.Name,
		Type:     notification.ChannelType(r.Type),
		Disabled: lo.FromPtrOr(r.Disabled, notification.DefaultDisabled),
		Config: notification.ChannelConfig{
			ChannelConfigMeta: notification.ChannelConfigMeta{
				Type: notification.ChannelType(r.Type),
			},
			WebHook: notification.WebHookChannelConfig{
				CustomHeaders: lo.FromPtr(r.CustomHeaders),
				URL:           r.Url,
				SigningSecret: lo.FromPtr(r.SigningSecret),
			},
		},
	}
}

func AsChannelWebhookUpdateRequest(r api.NotificationChannelWebhookCreateRequest, namespace, channelId string) notification.UpdateChannelInput {
	return notification.UpdateChannelInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: namespace,
		},
		ID:       channelId,
		Name:     r.Name,
		Type:     notification.ChannelType(r.Type),
		Disabled: lo.FromPtrOr(r.Disabled, notification.DefaultDisabled),
		Config: notification.ChannelConfig{
			ChannelConfigMeta: notification.ChannelConfigMeta{
				Type: notification.ChannelType(r.Type),
			},
			WebHook: notification.WebHookChannelConfig{
				CustomHeaders: lo.FromPtr(r.CustomHeaders),
				URL:           r.Url,
				SigningSecret: lo.FromPtr(r.SigningSecret),
			},
		},
	}
}

func AsRuleBalanceThresholdCreateRequest(r api.NotificationRuleBalanceThresholdCreateRequest, namespace string) notification.CreateRuleInput {
	return notification.CreateRuleInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: namespace,
		},
		Name:     r.Name,
		Type:     notification.EventType(r.Type),
		Disabled: lo.FromPtrOr(r.Disabled, notification.DefaultDisabled),
		Config: notification.RuleConfig{
			RuleConfigMeta: notification.RuleConfigMeta{
				Type: notification.EventType(r.Type),
			},
			BalanceThreshold: notification.BalanceThresholdRuleConfig{
				Features:   lo.FromPtr(r.Features),
				Thresholds: r.Thresholds,
			},
		},
		Channels: r.Channels,
	}
}

func AsRuleBalanceThresholdUpdateRequest(r api.NotificationRuleBalanceThresholdCreateRequest, namespace, ruleID string) notification.UpdateRuleInput {
	return notification.UpdateRuleInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: namespace,
		},
		Name:     r.Name,
		Type:     notification.EventType(r.Type),
		Disabled: lo.FromPtrOr(r.Disabled, notification.DefaultDisabled),
		Config: notification.RuleConfig{
			RuleConfigMeta: notification.RuleConfigMeta{
				Type: notification.EventType(r.Type),
			},
			BalanceThreshold: notification.BalanceThresholdRuleConfig{
				Features:   lo.FromPtr(r.Features),
				Thresholds: r.Thresholds,
			},
		},
		Channels: r.Channels,
		ID:       ruleID,
	}
}

func FromRule(r notification.Rule) (api.NotificationRule, error) {
	var rule api.NotificationRule

	switch r.Type {
	case notification.EventTypeBalanceThreshold:
		rule = FromRuleBalanceThreshold(r)
	default:
		return rule, notification.ValidationError{
			Err: fmt.Errorf("invalid rule type: %s", r.Type),
		}
	}

	return rule, nil
}

func FromRuleBalanceThreshold(r notification.Rule) api.NotificationRuleBalanceThreshold {
	channels := make([]api.NotificationChannelMeta, 0, len(r.Channels))
	for _, channel := range r.Channels {
		channels = append(channels, api.NotificationChannelMeta{
			Id:   channel.ID,
			Type: api.NotificationChannelType(channel.Type),
		})
	}

	return api.NotificationRuleBalanceThreshold{
		Channels:  channels,
		CreatedAt: r.CreatedAt,
		Disabled:  lo.ToPtr(r.Disabled),
		Features: convert.SafeDeRef(&r.Config.BalanceThreshold.Features, func(featureIDs []string) *[]notification.FeatureMeta {
			var features []notification.FeatureMeta
			for _, id := range featureIDs {
				features = append(features, notification.FeatureMeta{
					Id: id,
				})
			}

			if len(features) == 0 {
				return nil
			}

			return &features
		}),
		Id:         r.ID,
		Name:       r.Name,
		Thresholds: r.Config.BalanceThreshold.Thresholds,
		Type:       api.NotificationRuleBalanceThresholdTypeEntitlementsBalanceThreshold,
		UpdatedAt:  r.UpdatedAt,
		DeletedAt:  r.DeletedAt,
	}
}

func FromEvent(e notification.Event) (api.NotificationEvent, error) {
	var (
		err  error
		rule api.NotificationRule
	)

	rule, err = FromRule(e.Rule)
	if err != nil {
		return api.NotificationEvent{}, fmt.Errorf("failed to cast notification rule: %w", err)
	}

	// Populate ChannelMeta in EventDeliveryStatus from Even.Rule.Channels as we only store Channel.ID in database
	// for EventDeliveryStatus objects.
	channelsByID := make(map[string]notification.Channel, len(e.Rule.Channels))
	for _, channel := range e.Rule.Channels {
		channelsByID[channel.ID] = channel
	}

	deliveryStatuses := make([]api.NotificationEventDeliveryStatus, 0, len(e.DeliveryStatus))
	for _, deliveryStatus := range e.DeliveryStatus {
		status := api.NotificationEventDeliveryStatus{
			Channel: notification.ChannelMeta{
				Id: deliveryStatus.ChannelID,
			},
			State:     api.NotificationEventDeliveryStatusState(deliveryStatus.State),
			UpdatedAt: deliveryStatus.UpdatedAt,
		}
		if channel, ok := channelsByID[deliveryStatus.ChannelID]; ok {
			status.Channel = api.NotificationChannelMeta{
				Id:   deliveryStatus.ChannelID,
				Type: api.NotificationChannelType(channel.Type),
			}
		}

		deliveryStatuses = append(deliveryStatuses, status)
	}

	var annotations api.Annotations
	if len(e.Annotations) > 0 {
		annotations = make(api.Annotations)
		for k, v := range e.Annotations {
			annotations[k] = v
		}
	}

	event := api.NotificationEvent{
		CreatedAt:      e.CreatedAt,
		DeliveryStatus: deliveryStatuses,
		Id:             e.ID,
		Rule:           rule,
		Annotations:    lo.EmptyableToPtr(annotations),
	}

	switch e.Type {
	case notification.EventTypeBalanceThreshold:
		event.Type = api.NotificationEventTypeEntitlementsBalanceThreshold
		event.Payload = FromEventAsBalanceThresholdPayload(e)
	default:
		return event, notification.ValidationError{
			Err: fmt.Errorf("invalid event type: %s", e.Type),
		}
	}

	return event, nil
}

func FromEventAsBalanceThresholdPayload(e notification.Event) api.NotificationEventBalanceThresholdPayload {
	return api.NotificationEventBalanceThresholdPayload{
		Id:        e.ID,
		Timestamp: e.CreatedAt,
		Type:      api.NotificationEventBalanceThresholdPayloadTypeEntitlementsBalanceThreshold,
		Data: struct {
			Entitlement api.EntitlementMetered                    `json:"entitlement"`
			Feature     api.Feature                               `json:"feature"`
			Subject     api.Subject                               `json:"subject"`
			Threshold   api.NotificationRuleBalanceThresholdValue `json:"threshold"`
			Value       api.EntitlementValue                      `json:"value"`
		}{
			Value:       e.Payload.BalanceThreshold.Value,
			Entitlement: e.Payload.BalanceThreshold.Entitlement,
			Feature:     e.Payload.BalanceThreshold.Feature,
			Subject:     e.Payload.BalanceThreshold.Subject,
			Threshold:   e.Payload.BalanceThreshold.Threshold,
		},
	}
}
