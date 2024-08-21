package notification

import (
	"errors"
	"fmt"

	"golang.org/x/net/context"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type (
	ChannelMeta = api.NotificationChannelMeta
	FeatureMeta = api.FeatureMeta

	BalanceThreshold = api.NotificationRuleBalanceThresholdValue
)

const (
	BalanceThresholdTypeNumber  = api.NUMBER
	BalanceThresholdTypePercent = api.PERCENT
)

type Rule struct {
	models.NamespacedModel
	models.ManagedModel

	// ID is the unique identifier for Rule.
	ID string `json:"id"`
	// Type of the notification Rule (e.g. entitlements.balance.threshold)
	Type RuleType `json:"type"`
	// Name of is the user provided name of the Rule.
	Name string `json:"name"`
	// Disabled defines whether the Rule is disabled or not.
	Disabled bool `json:"disabled"`
	// Config stores the actual Rule configuration specific to the Type.
	Config RuleConfig `json:"config"`
	// Channels stores the list of channels the Rule send notification Events to.
	Channels []Channel `json:"channels"`
}

func (r Rule) AsNotificationRule() (api.NotificationRule, error) {
	var rule api.NotificationRule
	var err error

	switch r.Type {
	case RuleTypeBalanceThreshold:
		err = rule.FromNotificationRuleBalanceThreshold(r.AsNotificationRuleBalanceThreshold())
		if err != nil {
			return rule, ValidationError{
				Err: err,
			}
		}
	default:
		return rule, ValidationError{
			Err: fmt.Errorf("invalid rule type: %s", r.Type),
		}
	}

	return rule, nil
}

func (r Rule) AsNotificationRuleBalanceThreshold() api.NotificationRuleBalanceThreshold {
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
		Disabled:  r.Disabled,
		Features: convert.SafeDeRef(&r.Config.BalanceThreshold.Features, func(featureIDs []string) *[]FeatureMeta {
			var features []FeatureMeta
			for _, id := range featureIDs {
				features = append(features, FeatureMeta{
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
		Type:       api.NotificationEventType(r.Type),
		UpdatedAt:  r.UpdatedAt,
		DeletedAt:  r.DeletedAt,
	}
}

func (r Rule) Validate(ctx context.Context, service Service) error {
	if r.Namespace == "" {
		return ValidationError{
			Err: errors.New("namespace is required"),
		}
	}

	if r.ID == "" {
		return ValidationError{
			Err: errors.New("id is required"),
		}
	}

	if r.Name == "" {
		return ValidationError{
			Err: errors.New("name is required"),
		}
	}

	if err := r.Type.Validate(); err != nil {
		return err
	}

	if err := r.Config.Validate(ctx, service, r.Namespace); err != nil {
		return err
	}

	return nil
}

func (r Rule) HasEnabledChannels() bool {
	for _, channel := range r.Channels {
		if !channel.Disabled {
			return true
		}
	}

	return false
}

const (
	RuleTypeBalanceThreshold = RuleType(api.EntitlementsBalanceThreshold)
)

type RuleType api.NotificationEventType

func (t RuleType) Validate() error {
	switch t {
	case RuleTypeBalanceThreshold:
		return nil
	default:
		return fmt.Errorf("unknown notification rule type %q", t)
	}
}

func (t RuleType) Values() []string {
	return []string{
		string(RuleTypeBalanceThreshold),
	}
}

type RuleConfigMeta struct {
	Type RuleType `json:"type"`
}

func (m RuleConfigMeta) Validate() error {
	return m.Type.Validate()
}

// RuleConfig is a union type capturing configuration parameters for all type of rules.
type RuleConfig struct {
	RuleConfigMeta

	// Balance Threshold
	BalanceThreshold BalanceThresholdRuleConfig `json:"balanceThreshold"`
}

// Validate invokes channel type specific validator and returns an error if channel configuration is invalid.
func (c RuleConfig) Validate(ctx context.Context, service Service, namespace string) error {
	switch c.Type {
	case RuleTypeBalanceThreshold:
		return c.BalanceThreshold.Validate(ctx, service, namespace)
	default:
		return fmt.Errorf("unknown rule type: %s", c.Type)
	}
}

// BalanceThresholdRuleConfig defines the configuration specific to channel with webhook type.
type BalanceThresholdRuleConfig struct {
	// Features stores the list of features the rule is associated with.
	Features []string `json:"features"`
	// Thresholds stores the list of thresholds used to trigger a new notification event if the balance exceeds one of the thresholds.
	Thresholds []BalanceThreshold `json:"thresholds"`
}

// Validate returns an error if balance threshold configuration is invalid.
func (b BalanceThresholdRuleConfig) Validate(ctx context.Context, service Service, namespace string) error {
	if len(b.Thresholds) == 0 {
		return fmt.Errorf("must provide at least one threshold")
	}

	for _, threshold := range b.Thresholds {
		switch threshold.Type {
		case BalanceThresholdTypeNumber:
			if threshold.Value <= 0 {
				return ValidationError{
					Err: fmt.Errorf("invalid threshold with type %s: value must be greater than 0: %f",
						threshold.Type,
						threshold.Value,
					),
				}
			}
		case BalanceThresholdTypePercent:
			if threshold.Value <= 0 || threshold.Value > 100 {
				return ValidationError{
					Err: fmt.Errorf("invalid threshold with type %s: value must be between 0 anad 100: %f",
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

type RuleOrderBy string

const (
	RuleOrderByID        = api.ListNotificationRulesParamsOrderById
	RuleOrderByType      = api.ListNotificationRulesParamsOrderByType
	RuleOrderByCreatedAt = api.ListNotificationRulesParamsOrderByCreatedAt
	RuleOrderByUpdatedAt = api.ListNotificationRulesParamsOrderByUpdatedAt
)

var _ validator = (*ListRulesInput)(nil)

type ListRulesInput struct {
	pagination.Page

	Namespaces      []string
	Rules           []string
	IncludeDisabled bool
	Types           []RuleType

	OrderBy api.ListNotificationRulesParamsOrderBy
	Order   sortx.Order
}

func (i ListRulesInput) Validate(_ context.Context, _ Service) error {
	return nil
}

type ListRulesResult = pagination.PagedResponse[Rule]

var _ validator = (*CreateRuleInput)(nil)

type CreateRuleInput struct {
	models.NamespacedModel

	// Type defines the Rule type (e.g. entitlements.balance.threshold)
	Type RuleType
	// Name stores the user defined name of the Rule.
	Name string
	// Disabled defines whether the Rule is disabled or not. Deleted Rules are always disabled.
	Disabled bool
	// Config stores the Rule Type specific configuration.
	Config RuleConfig
	// Channels defines the list of Channels the Rule needs to send Events.
	Channels []string
}

func (i CreateRuleInput) Validate(ctx context.Context, service Service) error {
	if i.Namespace == "" {
		return ValidationError{
			Err: errors.New("namespace is required"),
		}
	}

	if err := i.Type.Validate(); err != nil {
		return err
	}

	if i.Name == "" {
		return ValidationError{
			Err: errors.New("channel name is required"),
		}
	}

	if err := i.Config.Validate(ctx, service, i.Namespace); err != nil {
		return err
	}

	return nil
}

func (i CreateRuleInput) FromNotificationRuleBalanceThresholdCreateRequest(r api.NotificationRuleBalanceThresholdCreateRequest) CreateRuleInput {
	return CreateRuleInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: i.Namespace,
		},
		Name:     r.Name,
		Type:     RuleType(r.Type),
		Disabled: defaultx.WithDefault(r.Disabled, DefaultDisabled),
		Config: RuleConfig{
			RuleConfigMeta: RuleConfigMeta{
				Type: RuleType(r.Type),
			},
			BalanceThreshold: BalanceThresholdRuleConfig{
				Features:   defaultx.WithDefault(r.Features, nil),
				Thresholds: r.Thresholds,
			},
		},
		Channels: r.Channels,
	}
}

var _ validator = (*UpdateRuleInput)(nil)

type UpdateRuleInput struct {
	models.NamespacedModel

	// Type defines the Rule type (e.g. entitlements.balance.threshold)
	Type RuleType
	// Name stores the user defined name of the Rule.
	Name string
	// Disabled defines whether the Rule is disabled or not. Deleted Rules are always disabled.
	Disabled bool
	// Config stores the Rule Type specific configuration.
	Config RuleConfig
	// Channels defines the list of Channels the Rule needs to send Events.
	Channels []string

	// ID is the unique identifier for Rule.
	ID string
}

func (i UpdateRuleInput) Validate(ctx context.Context, service Service) error {
	if i.Namespace == "" {
		return ValidationError{
			Err: errors.New("namespace is required"),
		}
	}

	if err := i.Type.Validate(); err != nil {
		return err
	}

	if i.Name == "" {
		return ValidationError{
			Err: errors.New("rule name is required"),
		}
	}

	if err := i.Config.Validate(ctx, service, i.Namespace); err != nil {
		return err
	}

	if i.ID == "" {
		return ValidationError{
			Err: errors.New("rule id is required"),
		}
	}

	return nil
}

func (i UpdateRuleInput) FromNotificationRuleBalanceThresholdCreateRequest(r api.NotificationRuleBalanceThresholdCreateRequest) UpdateRuleInput {
	return UpdateRuleInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: i.Namespace,
		},
		Name:     r.Name,
		Type:     RuleType(r.Type),
		Disabled: defaultx.WithDefault(r.Disabled, DefaultDisabled),
		Config: RuleConfig{
			RuleConfigMeta: RuleConfigMeta{
				Type: RuleType(r.Type),
			},
			BalanceThreshold: BalanceThresholdRuleConfig{
				Features:   defaultx.WithDefault(r.Features, nil),
				Thresholds: r.Thresholds,
			},
		},
		Channels: r.Channels,
		ID:       i.ID,
	}
}

var _ validator = (*GetRuleInput)(nil)

type GetRuleInput models.NamespacedID

func (i GetRuleInput) Validate(_ context.Context, _ Service) error {
	if i.Namespace == "" {
		return ValidationError{
			Err: errors.New("namespace is required"),
		}
	}

	if i.ID == "" {
		return ValidationError{
			Err: errors.New("rule id is required"),
		}
	}

	return nil
}

var _ validator = (*DeleteRuleInput)(nil)

type DeleteRuleInput models.NamespacedID

func (i DeleteRuleInput) Validate(_ context.Context, _ Service) error {
	if i.Namespace == "" {
		return ValidationError{
			Err: errors.New("namespace is required"),
		}
	}

	if i.ID == "" {
		return ValidationError{
			Err: errors.New("rule id is required"),
		}
	}

	return nil
}
