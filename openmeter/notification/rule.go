package notification

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

var (
	_ models.Validator             = (*Rule)(nil)
	_ models.CustomValidator[Rule] = (*Rule)(nil)
)

type Rule struct {
	models.NamespacedID
	models.ManagedModel
	models.Annotations
	models.Metadata

	// Type of the notification Rule (e.g. entitlements.balance.threshold)
	Type EventType `json:"type"`
	// Name of is the user provided name of the Rule.
	Name string `json:"name"`
	// Disabled defines whether the Rule is disabled or not.
	Disabled bool `json:"disabled"`
	// Config stores the actual Rule configuration specific to the Type.
	Config RuleConfig `json:"config"`
	// Channels stores the list of channels the Rule send notification Events to.
	Channels []Channel `json:"channels"`
}

func (r Rule) ValidateWith(validators ...models.ValidatorFunc[Rule]) error {
	return models.Validate(r, validators...)
}

func (r Rule) Validate() error {
	var errs []error

	if err := r.NamespacedID.Validate(); err != nil {
		errs = append(errs, err)
	}

	if r.Name == "" {
		errs = append(errs, errors.New("name is required"))
	}

	if err := r.Type.Validate(); err != nil {
		errs = append(errs, err)
	}

	if err := r.Config.Validate(); err != nil {
		errs = append(errs, err)
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (r Rule) HasEnabledChannels() bool {
	for _, channel := range r.Channels {
		if !channel.Disabled {
			return true
		}
	}

	return false
}

var (
	_ models.Validator                       = (*RuleConfigMeta)(nil)
	_ models.CustomValidator[RuleConfigMeta] = (*RuleConfigMeta)(nil)
)

type RuleConfigMeta struct {
	Type EventType `json:"type"`
}

func (m RuleConfigMeta) ValidateWith(validators ...models.ValidatorFunc[RuleConfigMeta]) error {
	return models.Validate(m, validators...)
}

func (m RuleConfigMeta) Validate() error {
	return m.Type.Validate()
}

var (
	_ models.Validator                   = (*RuleConfig)(nil)
	_ models.CustomValidator[RuleConfig] = (*RuleConfig)(nil)
)

// RuleConfig is a union type capturing configuration parameters for all type of rules.
type RuleConfig struct {
	RuleConfigMeta

	// Balance Threshold
	BalanceThreshold *BalanceThresholdRuleConfig `json:"balanceThreshold,omitempty"`
	EntitlementReset *EntitlementResetRuleConfig `json:"entitlementReset,omitempty"`

	// Invoice
	Invoice *InvoiceRuleConfig `json:"invoice,omitempty"`
}

func (c RuleConfig) ValidateWith(validators ...models.ValidatorFunc[RuleConfig]) error {
	return models.Validate(c, validators...)
}

// Validate invokes channel type specific validator and returns an error if channel configuration is invalid.
func (c RuleConfig) Validate() error {
	switch c.Type {
	case EventTypeBalanceThreshold:
		if c.BalanceThreshold == nil {
			return models.NewGenericValidationError(errors.New("missing balance threshold rule config"))
		}

		return c.BalanceThreshold.Validate()
	case EventTypeEntitlementReset:
		if c.EntitlementReset == nil {
			return models.NewGenericValidationError(errors.New("missing entitlement reset rule config"))
		}

		return c.EntitlementReset.Validate()
	case EventTypeInvoiceCreated, EventTypeInvoiceUpdated:
		if c.Invoice == nil {
			return models.NewGenericValidationError(errors.New("missing invoice rule config"))
		}

		return c.Invoice.Validate()
	default:
		return models.NewGenericValidationError(fmt.Errorf("unknown rule type: %s", c.Type))
	}
}

var (
	_ models.Validator                       = (*ListRulesInput)(nil)
	_ models.CustomValidator[ListRulesInput] = (*ListRulesInput)(nil)
)

type ListRulesInput struct {
	pagination.Page

	Namespaces      []string
	Rules           []string
	IncludeDisabled bool
	Types           []EventType
	Channels        []string

	OrderBy OrderBy
	Order   sortx.Order
}

func (i ListRulesInput) ValidateWith(validators ...models.ValidatorFunc[ListRulesInput]) error {
	return models.Validate(i, validators...)
}

func (i ListRulesInput) Validate() error {
	return nil
}

type ListRulesResult = pagination.Result[Rule]

var (
	_ models.Validator                        = (*CreateRuleInput)(nil)
	_ models.CustomValidator[CreateRuleInput] = (*CreateRuleInput)(nil)
)

type CreateRuleInput struct {
	models.NamespacedModel

	// Type defines the Rule type (e.g. entitlements.balance.threshold)
	Type EventType
	// Name stores the user defined name of the Rule.
	Name string
	// Disabled defines whether the Rule is disabled or not. Deleted Rules are always disabled.
	Disabled bool
	// Config stores the Rule Type specific configuration.
	Config RuleConfig
	// Channels defines the list of Channels the Rule needs to send Events.
	Channels []string
	// Metadata
	Metadata models.Metadata
	// Annotations
	Annotations models.Annotations
}

func (i CreateRuleInput) ValidateWith(validators ...models.ValidatorFunc[CreateRuleInput]) error {
	return models.Validate(i, validators...)
}

const MaxChannelsPerRule = 5

func (i CreateRuleInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if err := i.Type.Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.Name == "" {
		errs = append(errs, errors.New("rule name is required"))
	}

	if err := i.Config.Validate(); err != nil {
		errs = append(errs, err)
	}

	if len(i.Channels) == 0 {
		errs = append(errs, errors.New("at least one channel is required"))
	}

	if len(i.Channels) > MaxChannelsPerRule {
		errs = append(errs, fmt.Errorf("too many channels: %d > %d", len(i.Channels), MaxChannelsPerRule))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

var (
	_ models.Validator                        = (*UpdateRuleInput)(nil)
	_ models.CustomValidator[UpdateRuleInput] = (*UpdateRuleInput)(nil)
)

type UpdateRuleInput struct {
	models.NamespacedID

	// Type defines the Rule type (e.g. entitlements.balance.threshold)
	Type EventType
	// Name stores the user defined name of the Rule.
	Name string
	// Disabled defines whether the Rule is disabled or not. Deleted Rules are always disabled.
	Disabled bool
	// Config stores the Rule Type specific configuration.
	Config RuleConfig
	// Channels defines the list of Channels the Rule needs to send Events.
	Channels []string
	// Metadata
	Metadata models.Metadata
	// Annotations
	Annotations models.Annotations
}

func (i UpdateRuleInput) ValidateWith(validators ...models.ValidatorFunc[UpdateRuleInput]) error {
	return models.Validate(i, validators...)
}

func (i UpdateRuleInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.ID == "" {
		errs = append(errs, errors.New("id is required"))
	}

	if err := i.Type.Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.Name == "" {
		errs = append(errs, errors.New("rule name is required"))
	}

	if err := i.Config.Validate(); err != nil {
		errs = append(errs, err)
	}

	if len(i.Channels) == 0 {
		errs = append(errs, errors.New("at least one channel is required"))
	}

	if len(i.Channels) > MaxChannelsPerRule {
		errs = append(errs, fmt.Errorf("too many channels: %d > %d", len(i.Channels), MaxChannelsPerRule))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

var (
	_ models.Validator                     = (*GetRuleInput)(nil)
	_ models.CustomValidator[GetRuleInput] = (*GetRuleInput)(nil)
)

type GetRuleInput models.NamespacedID

func (i GetRuleInput) ValidateWith(validators ...models.ValidatorFunc[GetRuleInput]) error {
	return models.Validate(i, validators...)
}

func (i GetRuleInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.ID == "" {
		errs = append(errs, errors.New("id is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

var (
	_ models.Validator                        = (*DeleteRuleInput)(nil)
	_ models.CustomValidator[DeleteRuleInput] = (*DeleteRuleInput)(nil)
)

type DeleteRuleInput = GetRuleInput
