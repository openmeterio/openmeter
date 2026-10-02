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
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/models"
)

// The allow-list and default ("id") mirror the adapter's own fallback ordering
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

// The stored value is the v1 API model. Its deprecated aliases `NUMBER` and `PERCENT`
// are not part of the v3 enum, so they are rejected on write even though reads still
// normalize them (events.ToAPIBalanceThresholdType).
func ToDomainBalanceThresholdType(v api.BillingNotificationBalanceThresholdType) (v1api.NotificationRuleBalanceThresholdValueType, error) {
	switch v {
	case api.BillingNotificationBalanceThresholdTypeBalanceValue:
		return v1api.NotificationRuleBalanceThresholdValueTypeBalanceValue, nil
	case api.BillingNotificationBalanceThresholdTypeUsagePercentage:
		return v1api.NotificationRuleBalanceThresholdValueTypeUsagePercentage, nil
	case api.BillingNotificationBalanceThresholdTypeUsageValue:
		return v1api.NotificationRuleBalanceThresholdValueTypeUsageValue, nil
	default:
		return "", models.NewGenericValidationError(fmt.Errorf("invalid notification balance threshold type: %s", v))
	}
}

func toDomainBalanceThresholds(thresholds []api.BillingNotificationBalanceThreshold) ([]notification.BalanceThreshold, error) {
	return lo.MapErr(thresholds, func(threshold api.BillingNotificationBalanceThreshold, _ int) (notification.BalanceThreshold, error) {
		thresholdType, err := ToDomainBalanceThresholdType(threshold.Type)

		return notification.BalanceThreshold{Type: thresholdType, Value: threshold.Value}, err
	})
}

func toAPIBalanceThresholds(thresholds []notification.BalanceThreshold) ([]api.BillingNotificationBalanceThreshold, error) {
	return lo.MapErr(thresholds, func(threshold notification.BalanceThreshold, _ int) (api.BillingNotificationBalanceThreshold, error) {
		thresholdType, err := events.ToAPIBalanceThresholdType(threshold.Type)

		return api.BillingNotificationBalanceThreshold{Type: thresholdType, Value: threshold.Value}, err
	})
}

// Create and update share the same request body, so both inputs are built from it.
type ruleRequest struct {
	Type        notification.EventType
	Name        string
	Disabled    bool
	Channels    []string
	Config      notification.RuleConfig
	Metadata    models.Metadata
	Annotations models.Annotations
}

func FromAPICreateRuleRequest(ns string, body api.BillingNotificationRuleRequest) (notification.CreateRuleInput, error) {
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

// Updates are full replacements: an omitted disabled, labels, or features resets the
// field. The service rejects a type that differs from the stored rule.
func FromAPIUpdateRuleRequest(ns string, id string, body api.BillingNotificationRuleRequest) (notification.UpdateRuleInput, error) {
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

func fromAPIRuleRequest(body api.BillingNotificationRuleRequest) (ruleRequest, error) {
	discriminator, err := body.Discriminator()
	if err != nil {
		return ruleRequest{}, models.NewGenericValidationError(fmt.Errorf("invalid notification rule type: %w", err))
	}

	ruleType := notification.EventType(discriminator)

	switch ruleType {
	case notification.EventTypeBalanceThreshold:
		v, err := body.AsBillingNotificationRuleBalanceThresholdRequest()
		if err != nil {
			return ruleRequest{}, models.NewGenericValidationError(fmt.Errorf("invalid balance threshold rule: %w", err))
		}

		thresholds, err := toDomainBalanceThresholds(v.Thresholds)
		if err != nil {
			return ruleRequest{}, err
		}

		return newRuleRequest(ruleType, v.Name, v.Disabled, v.Channels, v.Labels, notification.RuleConfig{
			RuleConfigMeta: notification.RuleConfigMeta{Type: ruleType},
			BalanceThreshold: &notification.BalanceThresholdRuleConfig{
				Features:   fromAPIFeatures(v.Features),
				Thresholds: thresholds,
			},
		})
	case notification.EventTypeEntitlementReset:
		v, err := body.AsBillingNotificationRuleEntitlementResetRequest()
		if err != nil {
			return ruleRequest{}, models.NewGenericValidationError(fmt.Errorf("invalid entitlement reset rule: %w", err))
		}

		return newRuleRequest(ruleType, v.Name, v.Disabled, v.Channels, v.Labels, notification.RuleConfig{
			RuleConfigMeta: notification.RuleConfigMeta{Type: ruleType},
			EntitlementReset: &notification.EntitlementResetRuleConfig{
				Features: fromAPIFeatures(v.Features),
			},
		})
	case notification.EventTypeInvoiceCreated:
		v, err := body.AsBillingNotificationRuleInvoiceCreatedRequest()
		if err != nil {
			return ruleRequest{}, models.NewGenericValidationError(fmt.Errorf("invalid invoice created rule: %w", err))
		}

		return newRuleRequest(ruleType, v.Name, v.Disabled, v.Channels, v.Labels, notification.RuleConfig{
			RuleConfigMeta: notification.RuleConfigMeta{Type: ruleType},
			Invoice:        &notification.InvoiceRuleConfig{},
		})
	case notification.EventTypeInvoiceUpdated:
		v, err := body.AsBillingNotificationRuleInvoiceUpdatedRequest()
		if err != nil {
			return ruleRequest{}, models.NewGenericValidationError(fmt.Errorf("invalid invoice updated rule: %w", err))
		}

		return newRuleRequest(ruleType, v.Name, v.Disabled, v.Channels, v.Labels, notification.RuleConfig{
			RuleConfigMeta: notification.RuleConfigMeta{Type: ruleType},
			Invoice:        &notification.InvoiceRuleConfig{},
		})
	default:
		return ruleRequest{}, models.NewGenericValidationError(fmt.Errorf("invalid notification rule type: %s", discriminator))
	}
}

func newRuleRequest(ruleType notification.EventType, name string, disabled *bool, channels []api.NotificationChannelReference, apiLabels *api.Labels, config notification.RuleConfig) (ruleRequest, error) {
	ma, err := labels.ToMetadataAnnotations(apiLabels)
	if err != nil {
		return ruleRequest{}, err
	}

	return ruleRequest{
		Type:        ruleType,
		Name:        name,
		Disabled:    lo.FromPtr(disabled),
		Channels:    lo.Map(channels, func(ref api.NotificationChannelReference, _ int) string { return ref.Id }),
		Config:      config,
		Metadata:    ma.Metadata,
		Annotations: ma.Annotations,
	}, nil
}

// Only the rule's active channels are loaded on the domain object, so channels omits
// those disabled or deleted since the assignment; features were resolved by the service.
func ToAPIRule(r notification.RuleView) (api.BillingNotificationRule, error) {
	var rule api.BillingNotificationRule

	channels := lo.Map(r.Channels, func(channel notification.Channel, _ int) api.NotificationChannelReference {
		return api.NotificationChannelReference{Id: channel.ID}
	})
	features := toAPIFeatures(r.Features)
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

		return rule, rule.FromBillingNotificationRuleBalanceThreshold(api.BillingNotificationRuleBalanceThreshold{
			Id:         r.ID,
			Name:       r.Name,
			Type:       api.BillingNotificationRuleBalanceThresholdTypeEntitlementsBalanceThreshold,
			Disabled:   lo.ToPtr(r.Disabled),
			Channels:   channels,
			Thresholds: thresholds,
			Features:   features,
			Labels:     ruleLabels,
			CreatedAt:  r.CreatedAt,
			UpdatedAt:  r.UpdatedAt,
			DeletedAt:  r.DeletedAt,
		})
	case notification.EventTypeEntitlementReset:
		if r.Config.EntitlementReset == nil {
			return rule, fmt.Errorf("missing entitlement reset config on notification rule %s", r.ID)
		}

		return rule, rule.FromBillingNotificationRuleEntitlementReset(api.BillingNotificationRuleEntitlementReset{
			Id:        r.ID,
			Name:      r.Name,
			Type:      api.BillingNotificationRuleEntitlementResetTypeEntitlementsReset,
			Disabled:  lo.ToPtr(r.Disabled),
			Channels:  channels,
			Features:  features,
			Labels:    ruleLabels,
			CreatedAt: r.CreatedAt,
			UpdatedAt: r.UpdatedAt,
			DeletedAt: r.DeletedAt,
		})
	case notification.EventTypeInvoiceCreated:
		return rule, rule.FromBillingNotificationRuleInvoiceCreated(api.BillingNotificationRuleInvoiceCreated{
			Id:        r.ID,
			Name:      r.Name,
			Type:      api.BillingNotificationRuleInvoiceCreatedTypeInvoiceCreated,
			Disabled:  lo.ToPtr(r.Disabled),
			Channels:  channels,
			Labels:    ruleLabels,
			CreatedAt: r.CreatedAt,
			UpdatedAt: r.UpdatedAt,
			DeletedAt: r.DeletedAt,
		})
	case notification.EventTypeInvoiceUpdated:
		return rule, rule.FromBillingNotificationRuleInvoiceUpdated(api.BillingNotificationRuleInvoiceUpdated{
			Id:        r.ID,
			Name:      r.Name,
			Type:      api.BillingNotificationRuleInvoiceUpdatedTypeInvoiceUpdated,
			Disabled:  lo.ToPtr(r.Disabled),
			Channels:  channels,
			Labels:    ruleLabels,
			CreatedAt: r.CreatedAt,
			UpdatedAt: r.UpdatedAt,
			DeletedAt: r.DeletedAt,
		})
	default:
		return rule, fmt.Errorf("invalid notification rule type: %s", r.Type)
	}
}

func fromAPIFeatures(refs *[]api.FeatureReference) []string {
	if refs == nil {
		return nil
	}

	return lo.Map(*refs, func(ref api.FeatureReference, _ int) string { return ref.Id })
}

func toAPIFeatures(features []feature.Feature) *[]api.FeatureReference {
	if len(features) == 0 {
		return nil
	}

	return lo.ToPtr(lo.Map(features, func(f feature.Feature, _ int) api.FeatureReference {
		return api.FeatureReference{Id: f.ID}
	}))
}
