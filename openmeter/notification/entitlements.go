package notification

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"

	apiv1models "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/pkg/models"
)

const (
	EventTypeBalanceThreshold EventType = "entitlements.balance.threshold"
	EventTypeEntitlementReset EventType = "entitlements.reset"
)

var (
	_ models.Validator                                    = (*EntitlementValuePayloadBase)(nil)
	_ models.CustomValidator[EntitlementValuePayloadBase] = (*EntitlementValuePayloadBase)(nil)
)

type EntitlementValuePayloadBase struct {
	Entitlement apiv1models.EntitlementMetered `json:"entitlement"`
	Feature     apiv1models.Feature            `json:"feature"`
	Subject     apiv1models.Subject            `json:"subject"`
	Value       apiv1models.EntitlementValue   `json:"value"`
	Customer    apiv1models.Customer           `json:"customer"`
}

func (e EntitlementValuePayloadBase) ValidateWith(validators ...models.ValidatorFunc[EntitlementValuePayloadBase]) error {
	return models.Validate(e, validators...)
}

func (e EntitlementValuePayloadBase) Validate() error {
	return nil
}

var (
	_ models.Validator                                = (*BalanceThresholdPayload)(nil)
	_ models.CustomValidator[BalanceThresholdPayload] = (*BalanceThresholdPayload)(nil)
)

type BalanceThresholdPayload struct {
	EntitlementValuePayloadBase

	Threshold apiv1models.NotificationRuleBalanceThresholdValue `json:"threshold"`
}

func (b BalanceThresholdPayload) ValidateWith(validators ...models.ValidatorFunc[BalanceThresholdPayload]) error {
	return models.Validate(b, validators...)
}

// Validate returns an error if the balance threshold payload is invalid.
func (b BalanceThresholdPayload) Validate() error {
	return nil
}

const (
	BalanceThresholdTypeNumber          = apiv1models.NotificationRuleBalanceThresholdValueTypeNumber
	BalanceThresholdTypePercent         = apiv1models.NotificationRuleBalanceThresholdValueTypePercent
	BalanceThresholdTypeUsagePercentage = apiv1models.NotificationRuleBalanceThresholdValueTypeUsagePercentage
	BalanceThresholdTypeBalanceValue    = apiv1models.NotificationRuleBalanceThresholdValueTypeBalanceValue
	BalanceThresholdTypeUsageValue      = apiv1models.NotificationRuleBalanceThresholdValueTypeUsageValue
)

type (
	FeatureMeta      = apiv1models.FeatureMeta
	BalanceThreshold = apiv1models.NotificationRuleBalanceThresholdValue
)

var (
	_ models.Validator                                   = (*BalanceThresholdRuleConfig)(nil)
	_ models.CustomValidator[BalanceThresholdRuleConfig] = (*BalanceThresholdRuleConfig)(nil)
)

// BalanceThresholdRuleConfig defines the configuration specific to rule.
type BalanceThresholdRuleConfig struct {
	// Features store the list of features the rule is associated with.
	Features []string `json:"features"`
	// Thresholds store the list of thresholds used to trigger a new notification event if the balance exceeds one of the thresholds.
	Thresholds []BalanceThreshold `json:"thresholds"`
}

func (b BalanceThresholdRuleConfig) ValidateWith(validators ...models.ValidatorFunc[BalanceThresholdRuleConfig]) error {
	return models.Validate(b, validators...)
}

// Validate returns an error if the balance threshold configuration is invalid.
func (b BalanceThresholdRuleConfig) Validate() error {
	var errs []error

	if len(b.Features) != len(lo.Uniq(b.Features)) {
		errs = append(errs, errors.New("duplicated features"))
	}

	if len(b.Thresholds) == 0 {
		errs = append(errs, errors.New("must provide at least one threshold"))
	}

	for _, threshold := range b.Thresholds {
		switch threshold.Type {
		case BalanceThresholdTypeNumber, BalanceThresholdTypePercent:
			fallthrough
		case BalanceThresholdTypeUsageValue, BalanceThresholdTypeUsagePercentage:
			if threshold.Value <= 0 {
				errs = append(errs, fmt.Errorf("invalid threshold with type %s: value must be greater than 0: %.2f",
					threshold.Type, threshold.Value))
			}
		case BalanceThresholdTypeBalanceValue:
			if threshold.Value < 0 {
				errs = append(errs, fmt.Errorf("invalid threshold with type %s: value must be greater than or equal to 0: %.2f",
					threshold.Type, threshold.Value))
			}
		default:
			errs = append(errs, fmt.Errorf("unknown balance threshold type: %s", threshold.Type))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func ValidateRuleConfigWithFeatures(ctx context.Context, service FeatureService, namespace string) models.ValidatorFunc[RuleConfig] {
	return func(r RuleConfig) error {
		var featuresIDOrKeys []string

		switch r.Type {
		case EventTypeBalanceThreshold:
			if r.BalanceThreshold == nil {
				return models.NewGenericValidationError(errors.New("missing balance threshold rule config"))
			}

			featuresIDOrKeys = r.BalanceThreshold.Features
		case EventTypeEntitlementReset:
			if r.EntitlementReset == nil {
				return models.NewGenericValidationError(errors.New("missing entitlement reset rule config"))
			}

			featuresIDOrKeys = r.EntitlementReset.Features
		default:
			return nil
		}

		featuresIDOrKeys = lo.Uniq(featuresIDOrKeys)

		if len(featuresIDOrKeys) > 0 {
			features, err := service.ListFeature(ctx, namespace, featuresIDOrKeys...)
			if err != nil {
				return fmt.Errorf("failed to list features: %w", err)
			}

			// Collect all feature IDs and keys returned by the API.
			featureIDAndKeys := make(map[string]struct{}, len(features))

			for _, feature := range features {
				featureIDAndKeys[feature.ID] = struct{}{}
				featureIDAndKeys[feature.Key] = struct{}{}
			}

			// Collect all feature IDs and keys that are available in the rule config but are missing from the API response.
			missingFeatures := make([]string, 0)

			for _, featureIdOrKey := range featuresIDOrKeys {
				if _, ok := featureIDAndKeys[featureIdOrKey]; !ok {
					missingFeatures = append(missingFeatures, featureIdOrKey)
				}
			}

			if len(missingFeatures) > 0 {
				return models.NewGenericValidationError(fmt.Errorf("non-existing features: %v", missingFeatures))
			}
		}

		return nil
	}
}

// EntitlementReset support

var (
	_ models.Validator                                = (*EntitlementResetPayload)(nil)
	_ models.CustomValidator[EntitlementResetPayload] = (*EntitlementResetPayload)(nil)
)

type EntitlementResetPayload EntitlementValuePayloadBase

func (e EntitlementResetPayload) ValidateWith(validators ...models.ValidatorFunc[EntitlementResetPayload]) error {
	return models.Validate(e, validators...)
}

func (e EntitlementResetPayload) Validate() error {
	return nil
}

var (
	_ models.Validator                                   = (*EntitlementResetRuleConfig)(nil)
	_ models.CustomValidator[EntitlementResetRuleConfig] = (*EntitlementResetRuleConfig)(nil)
)

type EntitlementResetRuleConfig struct {
	Features []string `json:"features"`
}

func (e EntitlementResetRuleConfig) ValidateWith(validators ...models.ValidatorFunc[EntitlementResetRuleConfig]) error {
	return models.Validate(e, validators...)
}

func (e EntitlementResetRuleConfig) Validate() error {
	var errs []error

	if len(e.Features) != len(lo.Uniq(e.Features)) {
		errs = append(errs, errors.New("duplicated features"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
