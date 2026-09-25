package rules

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	v1api "github.com/openmeterio/openmeter/api"
	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/handlers/notification/events"
	"github.com/openmeterio/openmeter/api/v3/labels"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/models"
)

// FromAPIRuleSortField maps a v3 API sort field name to the domain OrderBy. The
// allow-list and default ("id") mirror the adapter's own fallback ordering
// (openmeter/notification/adapter/rule.go).
func FromAPIRuleSortField(ctx context.Context, field string) (notification.OrderBy, error) {
	switch field {
	case "id":
		return notification.OrderByID, nil
	case "type":
		return notification.OrderByType, nil
	case "created_at":
		return notification.OrderByCreatedAt, nil
	case "updated_at":
		return notification.OrderByUpdatedAt, nil
	default:
		return "", apierrors.NewUnsupportedSortFieldError(ctx, field, "id", "type", "created_at", "updated_at")
	}
}

// ToDomainBalanceThresholdType maps a v3 threshold type to the value stored on the
// rule, which is the v1 API model. The deprecated v1 aliases `NUMBER` and `PERCENT`
// are not part of the v3 enum, so they are rejected on write even though reads
// still normalize them (events.ToAPIBalanceThresholdType).
func ToDomainBalanceThresholdType(v api.NotificationBalanceThresholdType) (v1api.NotificationRuleBalanceThresholdValueType, error) {
	switch v {
	case api.NotificationBalanceThresholdTypeBalanceValue:
		return v1api.NotificationRuleBalanceThresholdValueTypeBalanceValue, nil
	case api.NotificationBalanceThresholdTypeUsagePercentage:
		return v1api.NotificationRuleBalanceThresholdValueTypeUsagePercentage, nil
	case api.NotificationBalanceThresholdTypeUsageValue:
		return v1api.NotificationRuleBalanceThresholdValueTypeUsageValue, nil
	default:
		return "", models.NewGenericValidationError(fmt.Errorf("invalid notification balance threshold type: %s", v))
	}
}

func toDomainBalanceThresholds(thresholds []api.NotificationBalanceThreshold) ([]notification.BalanceThreshold, error) {
	result := make([]notification.BalanceThreshold, 0, len(thresholds))

	for _, threshold := range thresholds {
		thresholdType, err := ToDomainBalanceThresholdType(threshold.Type)
		if err != nil {
			return nil, err
		}

		result = append(result, notification.BalanceThreshold{
			Type:  thresholdType,
			Value: threshold.Value,
		})
	}

	return result, nil
}

func toAPIBalanceThresholds(thresholds []notification.BalanceThreshold) ([]api.NotificationBalanceThreshold, error) {
	result := make([]api.NotificationBalanceThreshold, 0, len(thresholds))

	for _, threshold := range thresholds {
		thresholdType, err := events.ToAPIBalanceThresholdType(threshold.Type)
		if err != nil {
			return nil, err
		}

		result = append(result, api.NotificationBalanceThreshold{
			Type:  thresholdType,
			Value: threshold.Value,
		})
	}

	return result, nil
}

// ruleRequest is the type-independent content of a NotificationRuleRequest. Create and
// update share the same request body, so both inputs are built from it.
type ruleRequest struct {
	Type        notification.EventType
	Name        string
	Disabled    bool
	Channels    []string
	Config      notification.RuleConfig
	Metadata    models.Metadata
	Annotations models.Annotations
}

// FromAPICreateRuleRequest maps a v3 request body to the domain create input.
func FromAPICreateRuleRequest(ns string, body api.NotificationRuleRequest) (notification.CreateRuleInput, error) {
	r, err := fromAPIRuleRequest(body)
	if err != nil {
		return notification.CreateRuleInput{}, err
	}

	return notification.CreateRuleInput{
		NamespacedModel: models.NamespacedModel{Namespace: ns},
		Type:            r.Type,
		Name:            r.Name,
		Disabled:        r.Disabled,
		Config:          r.Config,
		Channels:        r.Channels,
		Metadata:        r.Metadata,
		Annotations:     r.Annotations,
	}, nil
}

// FromAPIUpdateRuleRequest maps a v3 request body to the domain update input. Updates
// are full replacements: an omitted disabled, labels, or features resets the field.
// The service rejects a type that differs from the stored rule.
func FromAPIUpdateRuleRequest(ns string, id string, body api.NotificationRuleRequest) (notification.UpdateRuleInput, error) {
	r, err := fromAPIRuleRequest(body)
	if err != nil {
		return notification.UpdateRuleInput{}, err
	}

	return notification.UpdateRuleInput{
		NamespacedID: models.NamespacedID{Namespace: ns, ID: id},
		Type:         r.Type,
		Name:         r.Name,
		Disabled:     r.Disabled,
		Config:       r.Config,
		Channels:     r.Channels,
		Metadata:     r.Metadata,
		Annotations:  r.Annotations,
	}, nil
}

func fromAPIRuleRequest(body api.NotificationRuleRequest) (ruleRequest, error) {
	discriminator, err := body.Discriminator()
	if err != nil {
		return ruleRequest{}, models.NewGenericValidationError(fmt.Errorf("invalid notification rule type: %w", err))
	}

	ruleType := notification.EventType(discriminator)

	switch ruleType {
	case notification.EventTypeBalanceThreshold:
		v, err := body.AsNotificationRuleBalanceThresholdRequest()
		if err != nil {
			return ruleRequest{}, models.NewGenericValidationError(fmt.Errorf("invalid balance threshold rule: %w", err))
		}

		thresholds, err := toDomainBalanceThresholds(v.Thresholds)
		if err != nil {
			return ruleRequest{}, err
		}

		return newRuleRequest(ruleType, v.Name, v.Disabled, v.ChannelIds, v.Labels, notification.RuleConfig{
			RuleConfigMeta: notification.RuleConfigMeta{Type: ruleType},
			BalanceThreshold: &notification.BalanceThresholdRuleConfig{
				Features:   lo.FromPtr(v.Features),
				Thresholds: thresholds,
			},
		})
	case notification.EventTypeEntitlementReset:
		v, err := body.AsNotificationRuleEntitlementResetRequest()
		if err != nil {
			return ruleRequest{}, models.NewGenericValidationError(fmt.Errorf("invalid entitlement reset rule: %w", err))
		}

		return newRuleRequest(ruleType, v.Name, v.Disabled, v.ChannelIds, v.Labels, notification.RuleConfig{
			RuleConfigMeta: notification.RuleConfigMeta{Type: ruleType},
			EntitlementReset: &notification.EntitlementResetRuleConfig{
				Features: lo.FromPtr(v.Features),
			},
		})
	case notification.EventTypeInvoiceCreated:
		v, err := body.AsNotificationRuleInvoiceCreatedRequest()
		if err != nil {
			return ruleRequest{}, models.NewGenericValidationError(fmt.Errorf("invalid invoice created rule: %w", err))
		}

		return newRuleRequest(ruleType, v.Name, v.Disabled, v.ChannelIds, v.Labels, notification.RuleConfig{
			RuleConfigMeta: notification.RuleConfigMeta{Type: ruleType},
			Invoice:        &notification.InvoiceRuleConfig{},
		})
	case notification.EventTypeInvoiceUpdated:
		v, err := body.AsNotificationRuleInvoiceUpdatedRequest()
		if err != nil {
			return ruleRequest{}, models.NewGenericValidationError(fmt.Errorf("invalid invoice updated rule: %w", err))
		}

		return newRuleRequest(ruleType, v.Name, v.Disabled, v.ChannelIds, v.Labels, notification.RuleConfig{
			RuleConfigMeta: notification.RuleConfigMeta{Type: ruleType},
			Invoice:        &notification.InvoiceRuleConfig{},
		})
	default:
		return ruleRequest{}, models.NewGenericValidationError(fmt.Errorf("invalid notification rule type: %s", discriminator))
	}
}

func newRuleRequest(ruleType notification.EventType, name string, disabled *bool, channelIDs []api.ULID, apiLabels *api.Labels, config notification.RuleConfig) (ruleRequest, error) {
	ma, err := labels.ToMetadataAnnotations(apiLabels)
	if err != nil {
		return ruleRequest{}, err
	}

	return ruleRequest{
		Type:        ruleType,
		Name:        name,
		Disabled:    lo.FromPtr(disabled),
		Channels:    channelIDs,
		Config:      config,
		Metadata:    ma.Metadata,
		Annotations: ma.Annotations,
	}, nil
}

// ToAPIRule maps a domain Rule to its v3 API representation. Only the rule's active
// channels are loaded on the domain object, so channel_ids omits channels that have
// been disabled or deleted since the assignment.
func ToAPIRule(r notification.Rule) (api.NotificationRule, error) {
	var rule api.NotificationRule

	channelIDs := lo.Map(r.Channels, func(channel notification.Channel, _ int) api.ULID {
		return channel.ID
	})
	ruleLabels := labels.FromMetadataAnnotations(r.Metadata, r.Annotations)

	switch r.Type {
	case notification.EventTypeBalanceThreshold:
		if r.Config.BalanceThreshold == nil {
			return rule, fmt.Errorf("missing balance threshold config on notification rule %s", r.ID)
		}

		thresholds, err := toAPIBalanceThresholds(r.Config.BalanceThreshold.Thresholds)
		if err != nil {
			return rule, err
		}

		return rule, rule.FromNotificationRuleBalanceThreshold(api.NotificationRuleBalanceThreshold{
			Id:         r.ID,
			Name:       r.Name,
			Type:       api.NotificationRuleBalanceThresholdTypeEntitlementsBalanceThreshold,
			Disabled:   lo.ToPtr(r.Disabled),
			ChannelIds: channelIDs,
			Thresholds: thresholds,
			Features:   toAPIFeatures(r.Config.BalanceThreshold.Features),
			Labels:     ruleLabels,
			CreatedAt:  r.CreatedAt,
			UpdatedAt:  r.UpdatedAt,
			DeletedAt:  r.DeletedAt,
		})
	case notification.EventTypeEntitlementReset:
		if r.Config.EntitlementReset == nil {
			return rule, fmt.Errorf("missing entitlement reset config on notification rule %s", r.ID)
		}

		return rule, rule.FromNotificationRuleEntitlementReset(api.NotificationRuleEntitlementReset{
			Id:         r.ID,
			Name:       r.Name,
			Type:       api.NotificationRuleEntitlementResetTypeEntitlementsReset,
			Disabled:   lo.ToPtr(r.Disabled),
			ChannelIds: channelIDs,
			Features:   toAPIFeatures(r.Config.EntitlementReset.Features),
			Labels:     ruleLabels,
			CreatedAt:  r.CreatedAt,
			UpdatedAt:  r.UpdatedAt,
			DeletedAt:  r.DeletedAt,
		})
	case notification.EventTypeInvoiceCreated:
		return rule, rule.FromNotificationRuleInvoiceCreated(api.NotificationRuleInvoiceCreated{
			Id:         r.ID,
			Name:       r.Name,
			Type:       api.NotificationRuleInvoiceCreatedTypeInvoiceCreated,
			Disabled:   lo.ToPtr(r.Disabled),
			ChannelIds: channelIDs,
			Labels:     ruleLabels,
			CreatedAt:  r.CreatedAt,
			UpdatedAt:  r.UpdatedAt,
			DeletedAt:  r.DeletedAt,
		})
	case notification.EventTypeInvoiceUpdated:
		return rule, rule.FromNotificationRuleInvoiceUpdated(api.NotificationRuleInvoiceUpdated{
			Id:         r.ID,
			Name:       r.Name,
			Type:       api.NotificationRuleInvoiceUpdatedTypeInvoiceUpdated,
			Disabled:   lo.ToPtr(r.Disabled),
			ChannelIds: channelIDs,
			Labels:     ruleLabels,
			CreatedAt:  r.CreatedAt,
			UpdatedAt:  r.UpdatedAt,
			DeletedAt:  r.DeletedAt,
		})
	default:
		return rule, fmt.Errorf("invalid notification rule type: %s", r.Type)
	}
}

func toAPIFeatures(features []string) *[]string {
	if len(features) == 0 {
		return nil
	}

	return &features
}
