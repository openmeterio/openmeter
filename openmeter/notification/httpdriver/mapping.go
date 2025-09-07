package httpdriver

import (
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	billinghttp "github.com/openmeterio/openmeter/openmeter/billing/httpdriver"
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
			BalanceThreshold: &notification.BalanceThresholdRuleConfig{
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
			BalanceThreshold: &notification.BalanceThresholdRuleConfig{
				Features:   lo.FromPtr(r.Features),
				Thresholds: r.Thresholds,
			},
		},
		Channels: r.Channels,
		ID:       ruleID,
	}
}

func AsRuleEntitlementResetCreateRequest(r api.NotificationRuleEntitlementResetCreateRequest, namespace string) notification.CreateRuleInput {
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
			EntitlementReset: &notification.EntitlementResetRuleConfig{
				Features: lo.FromPtr(r.Features),
			},
		},
		Channels: r.Channels,
	}
}

func AsRuleEntitlementResetUpdateRequest(r api.NotificationRuleEntitlementResetCreateRequest, namespace, ruleID string) notification.UpdateRuleInput {
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
			EntitlementReset: &notification.EntitlementResetRuleConfig{
				Features: lo.FromPtr(r.Features),
			},
		},
		Channels: r.Channels,
		ID:       ruleID,
	}
}

func AsRuleInvoiceCreatedCreateRequest(r api.NotificationRuleInvoiceCreatedCreateRequest, namespace string) notification.CreateRuleInput {
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
			Invoice: &notification.InvoiceRuleConfig{},
		},
		Channels: r.Channels,
	}
}

func AsRuleInvoiceCreatedUpdateRequest(r api.NotificationRuleInvoiceCreatedCreateRequest, namespace, id string) notification.UpdateRuleInput {
	return notification.UpdateRuleInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: namespace,
		},
		Type:     notification.EventType(r.Type),
		Name:     r.Name,
		Disabled: lo.FromPtrOr(r.Disabled, notification.DefaultDisabled),
		Config: notification.RuleConfig{
			RuleConfigMeta: notification.RuleConfigMeta{
				Type: notification.EventType(r.Type),
			},
			Invoice: &notification.InvoiceRuleConfig{},
		},
		Channels: r.Channels,
		ID:       id,
	}
}

func AsRuleInvoiceUpdatedCreateRequest(r api.NotificationRuleInvoiceUpdatedCreateRequest, namespace string) notification.CreateRuleInput {
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
			Invoice: &notification.InvoiceRuleConfig{},
		},
		Channels: r.Channels,
	}
}

func AsRuleInvoiceUpdatedUpdateRequest(r api.NotificationRuleInvoiceUpdatedCreateRequest, namespace, id string) notification.UpdateRuleInput {
	return notification.UpdateRuleInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: namespace,
		},
		Type:     notification.EventType(r.Type),
		Name:     r.Name,
		Disabled: lo.FromPtrOr(r.Disabled, notification.DefaultDisabled),
		Config: notification.RuleConfig{
			RuleConfigMeta: notification.RuleConfigMeta{
				Type: notification.EventType(r.Type),
			},
			Invoice: &notification.InvoiceRuleConfig{},
		},
		Channels: r.Channels,
		ID:       id,
	}
}

func FromRule(r notification.Rule) (api.NotificationRule, error) {
	var (
		rule api.NotificationRule
		err  error
	)

	switch r.Type {
	case notification.EventTypeBalanceThreshold:
		err = rule.FromNotificationRuleBalanceThreshold(FromRuleBalanceThreshold(r))
		if err != nil {
			return rule, fmt.Errorf("failed to cast notification rule with type: %s: %w", r.Type, err)
		}
	case notification.EventTypeEntitlementReset:
		err = rule.FromNotificationRuleEntitlementReset(FromRuleEntitlementReset(r))
		if err != nil {
			return rule, fmt.Errorf("failed to cast notification rule with type: %s: %w", r.Type, err)
		}
	case notification.EventTypeInvoiceCreated:
		err = rule.FromNotificationRuleInvoiceCreated(FromRuleInvoiceCreated(r))
		if err != nil {
			return rule, fmt.Errorf("failed to cast notification rule with type: %s: %w", r.Type, err)
		}
	case notification.EventTypeInvoiceUpdated:
		err = rule.FromNotificationRuleInvoiceUpdated(FromRuleInvoiceUpdated(r))
		if err != nil {
			return rule, fmt.Errorf("failed to cast notification rule with type: %s: %w", r.Type, err)
		}
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

func FromRuleEntitlementReset(r notification.Rule) api.NotificationRuleEntitlementReset {
	channels := make([]api.NotificationChannelMeta, 0, len(r.Channels))
	for _, channel := range r.Channels {
		channels = append(channels, api.NotificationChannelMeta{
			Id:   channel.ID,
			Type: api.NotificationChannelType(channel.Type),
		})
	}

	return api.NotificationRuleEntitlementReset{
		Channels:  channels,
		CreatedAt: r.CreatedAt,
		Disabled:  lo.ToPtr(r.Disabled),
		Features: convert.SafeDeRef(&r.Config.EntitlementReset.Features, func(featureIDs []string) *[]notification.FeatureMeta {
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
		Id:        r.ID,
		Name:      r.Name,
		Type:      api.NotificationRuleEntitlementResetTypeEntitlementsReset,
		UpdatedAt: r.UpdatedAt,
		DeletedAt: r.DeletedAt,
	}
}

func FromRuleInvoiceCreated(r notification.Rule) api.NotificationRuleInvoiceCreated {
	channels := make([]api.NotificationChannelMeta, 0, len(r.Channels))
	for _, channel := range r.Channels {
		channels = append(channels, api.NotificationChannelMeta{
			Id:   channel.ID,
			Type: api.NotificationChannelType(channel.Type),
		})
	}

	return api.NotificationRuleInvoiceCreated{
		Channels:  channels,
		CreatedAt: r.CreatedAt,
		Disabled:  lo.ToPtr(r.Disabled),
		Id:        r.ID,
		Name:      r.Name,
		Type:      api.NotificationRuleInvoiceCreatedTypeInvoiceCreated,
		UpdatedAt: r.UpdatedAt,
		DeletedAt: r.DeletedAt,
	}
}

func FromRuleInvoiceUpdated(r notification.Rule) api.NotificationRuleInvoiceUpdated {
	channels := make([]api.NotificationChannelMeta, 0, len(r.Channels))
	for _, channel := range r.Channels {
		channels = append(channels, api.NotificationChannelMeta{
			Id:   channel.ID,
			Type: api.NotificationChannelType(channel.Type),
		})
	}

	return api.NotificationRuleInvoiceUpdated{
		Channels:  channels,
		CreatedAt: r.CreatedAt,
		Disabled:  lo.ToPtr(r.Disabled),
		Id:        r.ID,
		Name:      r.Name,
		Type:      api.NotificationRuleInvoiceUpdatedTypeInvoiceUpdated,
		UpdatedAt: r.UpdatedAt,
		DeletedAt: r.DeletedAt,
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
			Channel: api.NotificationChannelMeta{
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

	event.Type, err = FromEventType(e.Type)
	if err != nil {
		return event, fmt.Errorf("failed to cast notification event type: %w", err)
	}

	switch e.Type {
	case notification.EventTypeBalanceThreshold:
		payload, err := FromEventAsBalanceThresholdPayload(e)
		if err != nil {
			return event, fmt.Errorf("failed to cast notification event payload: %w", err)
		}

		err = event.Payload.FromNotificationEventBalanceThresholdPayload(payload)
		if err != nil {
			return event, fmt.Errorf("failed to cast notification event payload: %w", err)
		}
	case notification.EventTypeEntitlementReset:
		payload, err := FromEventAsEntitlementResetPayload(e)
		if err != nil {
			return event, fmt.Errorf("failed to cast notification event payload: %w", err)
		}

		err = event.Payload.FromNotificationEventResetPayload(payload)
		if err != nil {
			return event, fmt.Errorf("failed to cast notification event payload: %w", err)
		}
	case notification.EventTypeInvoiceCreated:
		payload, err := FromEventAsInvoiceCreatedPayload(e)
		if err != nil {
			return event, fmt.Errorf("failed to cast notification event payload: %w", err)
		}

		err = event.Payload.FromNotificationEventInvoiceCreatedPayload(payload)
		if err != nil {
			return event, fmt.Errorf("failed to cast notification event payload: %w", err)
		}
	case notification.EventTypeInvoiceUpdated:
		payload, err := FromEventAsInvoiceUpdatedPayload(e)
		if err != nil {
			return event, fmt.Errorf("failed to cast notification event payload: %w", err)
		}

		err = event.Payload.FromNotificationEventInvoiceUpdatedPayload(payload)
		if err != nil {
			return event, fmt.Errorf("failed to cast notification event payload: %w", err)
		}
	default:
		return event, notification.ValidationError{
			Err: fmt.Errorf("invalid event payload type: %s", e.Type),
		}
	}

	return event, nil
}

func FromEventType(t notification.EventType) (api.NotificationEventType, error) {
	switch t {
	case notification.EventTypeBalanceThreshold:
		return api.NotificationEventTypeEntitlementsBalanceThreshold, nil
	case notification.EventTypeEntitlementReset:
		return api.NotificationEventTypeEntitlementsReset, nil
	case notification.EventTypeInvoiceCreated:
		return api.NotificationEventTypeInvoiceCreated, nil
	case notification.EventTypeInvoiceUpdated:
		return api.NotificationEventTypeInvoiceUpdated, nil
	default:
		return "", fmt.Errorf("invalid notification event type: %s", t)
	}
}

func FromEventAsBalanceThresholdPayload(e notification.Event) (api.NotificationEventBalanceThresholdPayload, error) {
	if e.Payload.BalanceThreshold == nil {
		return api.NotificationEventBalanceThresholdPayload{}, fmt.Errorf("balance threshold is nil")
	}

	return api.NotificationEventBalanceThresholdPayload{
		Id:        e.ID,
		Timestamp: e.CreatedAt,
		Type:      api.NotificationEventBalanceThresholdPayloadTypeEntitlementsBalanceThreshold,
		Data: api.NotificationEventBalanceThresholdPayloadData{
			Value:       e.Payload.BalanceThreshold.Value,
			Entitlement: e.Payload.BalanceThreshold.Entitlement,
			Feature:     e.Payload.BalanceThreshold.Feature,
			Subject:     e.Payload.BalanceThreshold.Subject,
			Customer:    lo.ToPtr(e.Payload.BalanceThreshold.Customer),
			Threshold:   e.Payload.BalanceThreshold.Threshold,
		},
	}, nil
}

func FromEventAsEntitlementResetPayload(e notification.Event) (api.NotificationEventResetPayload, error) {
	if e.Payload.EntitlementReset == nil {
		return api.NotificationEventResetPayload{}, fmt.Errorf("entitlement reset is nil")
	}

	var customer *api.Customer
	if e.Payload.BalanceThreshold != nil {
		customer = lo.ToPtr(e.Payload.BalanceThreshold.Customer)
	}

	return api.NotificationEventResetPayload{
		Id:        e.ID,
		Timestamp: e.CreatedAt,
		Type:      api.NotificationEventResetPayloadTypeEntitlementsReset,
		Data: api.NotificationEventEntitlementValuePayloadBase{
			Value:       e.Payload.EntitlementReset.Value,
			Entitlement: e.Payload.EntitlementReset.Entitlement,
			Feature:     e.Payload.EntitlementReset.Feature,
			Subject:     e.Payload.EntitlementReset.Subject,
			Customer:    customer,
		},
	}, nil
}

func FromEventAsInvoiceCreatedPayload(e notification.Event) (api.NotificationEventInvoiceCreatedPayload, error) {
	if e.Payload.Invoice == nil {
		return api.NotificationEventInvoiceCreatedPayload{}, fmt.Errorf("invoice is nil")
	}

	data, err := billinghttp.MapEventInvoiceToAPI(*e.Payload.Invoice)
	if err != nil {
		return api.NotificationEventInvoiceCreatedPayload{}, fmt.Errorf("failed to map event invoice to API: %w", err)
	}

	return api.NotificationEventInvoiceCreatedPayload{
		Id:        e.ID,
		Timestamp: e.CreatedAt,
		Data:      data,
		Type:      api.NotificationEventInvoiceCreatedPayloadTypeInvoiceCreated,
	}, nil
}

func FromEventAsInvoiceUpdatedPayload(e notification.Event) (api.NotificationEventInvoiceUpdatedPayload, error) {
	if e.Payload.Invoice == nil {
		return api.NotificationEventInvoiceUpdatedPayload{}, fmt.Errorf("invoice is nil")
	}

	data, err := billinghttp.MapEventInvoiceToAPI(*e.Payload.Invoice)
	if err != nil {
		return api.NotificationEventInvoiceUpdatedPayload{}, fmt.Errorf("failed to map event invoice to API: %w", err)
	}

	return api.NotificationEventInvoiceUpdatedPayload{
		Id:        e.ID,
		Timestamp: e.CreatedAt,
		Data:      data,
		Type:      api.NotificationEventInvoiceUpdatedPayloadTypeInvoiceUpdated,
	}, nil
}
