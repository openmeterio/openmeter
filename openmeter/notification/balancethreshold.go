package notification

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/api"
)

const (
	EventTypeBalanceThreshold EventType = "entitlements.balance.threshold"
)

type BalanceThresholdPayload struct {
	Entitlement api.EntitlementMetered                    `json:"entitlement"`
	Feature     api.Feature                               `json:"feature"`
	Subject     api.Subject                               `json:"subject"`
	Threshold   api.NotificationRuleBalanceThresholdValue `json:"threshold"`
	Value       api.EntitlementValue                      `json:"value"`
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

	if len(b.Features) > 0 {
		features, err := service.ListFeature(ctx, namespace, b.Features...)
		if err != nil {
			return err
		}

		if len(b.Features) != len(features) {
			featureIdOrKeys := make(map[string]struct{}, len(features))
			for _, feature := range features {
				featureIdOrKeys[feature.ID] = struct{}{}
				featureIdOrKeys[feature.Key] = struct{}{}
			}

			missingFeatures := make([]string, 0)
			for _, featureIdOrKey := range b.Features {
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
