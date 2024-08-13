package notification

import (
	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/pkg/models"
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

const (
	RuleTypeBalanceThreshold = RuleType(api.EntitlementsBalanceThreshold)
)

type RuleType api.NotificationEventType

func (t RuleType) Values() []string {
	return []string{
		string(RuleTypeBalanceThreshold),
	}
}

type RuleConfigMeta struct {
	Type RuleType `json:"type"`
}

// RuleConfig is a union type capturing configuration parameters for all type of rules.
type RuleConfig struct {
	RuleConfigMeta

	// Balance Threshold
	BalanceThreshold BalanceThresholdRuleConfig `json:"balanceThreshold"`
}

const (
	BalanceThresholdTypeNumber  = api.NUMBER
	BalanceThresholdTypePercent = api.PERCENT
)

type BalanceThreshold = api.NotificationRuleBalanceThresholdValue

// BalanceThresholdRuleConfig defines the configuration specific to channel with webhook type.
type BalanceThresholdRuleConfig struct {
	// Features stores the list of features the rule is associated with.
	Features []string `json:"features"`
	// Thresholds stores the list of thresholds used to trigger a new notification event if the balance exceeds one of the thresholds.
	Thresholds []BalanceThreshold `json:"thresholds"`
}
