package notification

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type Rule struct {
	models.NamespacedModel
	models.ManagedModel

	// ID is the unique identifier for Rule.
	ID string `json:"id"`
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

type RuleConfigMeta struct {
	Type EventType `json:"type"`
}

func (m RuleConfigMeta) Validate() error {
	return m.Type.Validate()
}

// RuleConfig is a union type capturing configuration parameters for all type of rules.
type RuleConfig struct {
	RuleConfigMeta

	// Balance Threshold
	BalanceThreshold *BalanceThresholdRuleConfig `json:"balanceThreshold,omitempty"`
	EntitlementReset *EntitlementResetRuleConfig `json:"entitlementReset,omitempty"`

	// Invoice
	Invoice *InvoiceRuleConfig `json:"invoice,omitempty"`
}

// Validate invokes channel type specific validator and returns an error if channel configuration is invalid.
func (c RuleConfig) Validate(ctx context.Context, service Service, namespace string) error {
	switch c.Type {
	case EventTypeBalanceThreshold:
		if c.BalanceThreshold == nil {
			return ValidationError{
				Err: errors.New("missing balance threshold rule config"),
			}
		}

		return c.BalanceThreshold.Validate(ctx, service, namespace)
	case EventTypeEntitlementReset:
		if c.EntitlementReset == nil {
			return ValidationError{
				Err: errors.New("missing entitlement reset rule config"),
			}
		}

		return c.EntitlementReset.Validate(ctx, service, namespace)
	case EventTypeInvoiceCreated, EventTypeInvoiceUpdated:
		if c.Invoice == nil {
			return ValidationError{
				Err: errors.New("missing invoice rule config"),
			}
		}

		return c.Invoice.Validate(ctx, service, namespace)

	default:
		return fmt.Errorf("unknown rule type: %s", c.Type)
	}
}

var _ validator = (*ListRulesInput)(nil)

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

func (i ListRulesInput) Validate(_ context.Context, _ Service) error {
	return nil
}

type ListRulesResult = pagination.Result[Rule]

var _ validator = (*CreateRuleInput)(nil)

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
}

const MaxChannelsPerRule = 5

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

	if len(i.Channels) > MaxChannelsPerRule {
		return ValidationError{
			Err: fmt.Errorf("too many channels: %d > %d", len(i.Channels), MaxChannelsPerRule),
		}
	}

	return nil
}

var _ validator = (*UpdateRuleInput)(nil)

type UpdateRuleInput struct {
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

	if len(i.Channels) > MaxChannelsPerRule {
		return ValidationError{
			Err: fmt.Errorf("too many channels: %d > %d", len(i.Channels), MaxChannelsPerRule),
		}
	}

	return nil
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
