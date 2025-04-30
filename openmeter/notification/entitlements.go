package notification

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/api"
)

const (
	EventTypeBalanceThreshold EventType = "entitlements.balance.threshold"
	EventTypeEntitlementReset EventType = "entitlements.reset"
)

type EntitlementValuePayloadBase struct {
	Entitlement api.EntitlementMetered `json:"entitlement"`
	Feature     api.Feature            `json:"feature"`
	Subject     api.Subject            `json:"subject"`
	Value       api.EntitlementValue   `json:"value"`
}

type BalanceThresholdPayload struct {
	EntitlementValuePayloadBase

	Threshold api.NotificationRuleBalanceThresholdValue `json:"threshold"`
}

// Validate returns an error if the balance threshold payload is invalid.
func (b BalanceThresholdPayload) Validate() error {
	return nil
}

const (
	BalanceThresholdTypeNumber  = api.NotificationRuleBalanceThresholdValueTypeNumber
	BalanceThresholdTypePercent = api.NotificationRuleBalanceThresholdValueTypePercent
)

type (
	FeatureMeta      = api.FeatureMeta
	BalanceThreshold = api.NotificationRuleBalanceThresholdValue
)

// BalanceThresholdRuleConfig defines the configuration specific to rule.
type BalanceThresholdRuleConfig struct {
	// Features store the list of features the rule is associated with.
	Features []string `json:"features"`
	// Thresholds store the list of thresholds used to trigger a new notification event if the balance exceeds one of the thresholds.
	Thresholds []BalanceThreshold `json:"thresholds"`
}

// Validate returns an error if the balance threshold configuration is invalid.
func (b BalanceThresholdRuleConfig) Validate(ctx context.Context, service Service, namespace string) error {
	if len(b.Thresholds) == 0 {
		return fmt.Errorf("must provide at least one threshold")
	}

	for _, threshold := range b.Thresholds {
		switch threshold.Type {
		case BalanceThresholdTypeNumber, BalanceThresholdTypePercent:
			if threshold.Value <= 0 {
				return ValidationError{
					Err: fmt.Errorf("invalid threshold with type %s: value must be greater than 0: %.2f",
						threshold.Type,
						threshold.Value,
					),
				}
			}
		default:
			return fmt.Errorf("unknown balance threshold type: %s", threshold.Type)
		}
	}

	if err := validateFeatures(ctx, service, namespace, b.Features); err != nil {
		return err
	}

	return nil
}

func validateFeatures(ctx context.Context, service Service, namespace string, features []string) error {
	if len(features) > 0 {
		dbFeatures, err := service.ListFeature(ctx, namespace, features...)
		if err != nil {
			return err
		}

		if len(features) != len(dbFeatures) {
			featureIdOrKeys := make(map[string]struct{}, len(features))
			for _, feature := range dbFeatures {
				featureIdOrKeys[feature.ID] = struct{}{}
				featureIdOrKeys[feature.Key] = struct{}{}
			}

			missingFeatures := make([]string, 0)
			for _, featureIdOrKey := range features {
				if _, ok := featureIdOrKeys[featureIdOrKey]; !ok {
					missingFeatures = append(missingFeatures, featureIdOrKey)
				}
			}

			return ValidationError{
				Err: fmt.Errorf("non-existing features: %v", missingFeatures),
			}
		}
	}

	return nil
}

// EntitlementReset support

type EntitlementResetPayload EntitlementValuePayloadBase

func (e EntitlementResetPayload) Validate() error {
	return nil
}

type EntitlementResetRuleConfig struct {
	Features []string `json:"features"`
}

func (e EntitlementResetRuleConfig) Validate(ctx context.Context, service Service, namespace string) error {
	if err := validateFeatures(ctx, service, namespace, e.Features); err != nil {
		return err
	}

	return nil
}
