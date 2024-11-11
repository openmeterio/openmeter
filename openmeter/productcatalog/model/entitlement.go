package model

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/datex"
)

type entitlementTemplater interface {
	json.Marshaler
	json.Unmarshaler
	Validator

	Type() entitlement.EntitlementType
	AsMetered() (MeteredEntitlementTemplate, error)
	AsStatic() (StaticEntitlementTemplate, error)
	AsBoolean() (BooleanEntitlementTemplate, error)
	FromMetered(MeteredEntitlementTemplate)
	FromStatic(StaticEntitlementTemplate)
	FromBoolean(BooleanEntitlementTemplate)
}

var _ entitlementTemplater = (*EntitlementTemplate)(nil)

// EntitlementTemplate is the template used for instantiating entitlement.Entitlement for RateCard.
type EntitlementTemplate struct {
	t       entitlement.EntitlementType
	metered *MeteredEntitlementTemplate
	static  *StaticEntitlementTemplate
	boolean *BooleanEntitlementTemplate
}

func (e *EntitlementTemplate) MarshalJSON() ([]byte, error) {
	var b []byte
	var err error

	switch e.t {
	case entitlement.EntitlementTypeMetered:
		b, err = json.Marshal(e.metered)
		if err != nil {
			return nil, fmt.Errorf("failed to json marshal metered entitlement: %w", err)
		}
	case entitlement.EntitlementTypeStatic:
		b, err = json.Marshal(e.static)
		if err != nil {
			return nil, fmt.Errorf("failed to json marshal metered entitlement: %w", err)
		}
	case entitlement.EntitlementTypeBoolean:
		b, err = json.Marshal(e.boolean)
		if err != nil {
			return nil, fmt.Errorf("failed to json marshal metered entitlement: %w", err)
		}
	default:
		return nil, fmt.Errorf("invalid entitlement type: %s", e.t)
	}

	return b, nil
}

func (e *EntitlementTemplate) UnmarshalJSON(bytes []byte) error {
	meta := &EntitlementTemplateMeta{}

	if err := json.Unmarshal(bytes, meta); err != nil {
		return fmt.Errorf("failed to json unmarshal entitlement template type: %w", err)
	}

	switch meta.Type {
	case entitlement.EntitlementTypeMetered:
		v := &MeteredEntitlementTemplate{}
		if err := json.Unmarshal(bytes, v); err != nil {
			return fmt.Errorf("failed to json unmarshal metered entitlement template: %w", err)
		}

		e.metered = v
		e.t = entitlement.EntitlementTypeMetered
	case entitlement.EntitlementTypeStatic:
		v := &StaticEntitlementTemplate{}
		if err := json.Unmarshal(bytes, v); err != nil {
			return fmt.Errorf("failed to json unmarshal metered entitlement template: %w", err)
		}

		e.static = v
		e.t = entitlement.EntitlementTypeStatic
	case entitlement.EntitlementTypeBoolean:
		v := &BooleanEntitlementTemplate{}
		if err := json.Unmarshal(bytes, v); err != nil {
			return fmt.Errorf("failed to json unmarshal boolean entitlement template: %w", err)
		}

		e.boolean = v
		e.t = entitlement.EntitlementTypeBoolean
	default:
		return fmt.Errorf("invalid entitlement type: %s", meta.Type)
	}

	return nil
}

func (e *EntitlementTemplate) Validate() error {
	if e == nil {
		return nil
	}

	switch e.t {
	case entitlement.EntitlementTypeMetered:
		return e.metered.Validate()
	case entitlement.EntitlementTypeStatic:
		return e.static.Validate()
	case entitlement.EntitlementTypeBoolean:
		return e.boolean.Validate()
	default:
		return fmt.Errorf("invalid entitlement type: %q", e.t)
	}
}

func (e *EntitlementTemplate) Type() entitlement.EntitlementType {
	return e.t
}

func (e *EntitlementTemplate) AsMetered() (MeteredEntitlementTemplate, error) {
	if e.t == "" || e.metered == nil {
		return MeteredEntitlementTemplate{}, errors.New("invalid entitlement template: not initialized")
	}

	return *e.metered, nil
}

func (e *EntitlementTemplate) AsStatic() (StaticEntitlementTemplate, error) {
	if e.t == "" || e.static == nil {
		return StaticEntitlementTemplate{}, errors.New("invalid entitlement template: not initialized")
	}

	return *e.static, nil
}

func (e *EntitlementTemplate) AsBoolean() (BooleanEntitlementTemplate, error) {
	if e.t == "" || e.boolean == nil {
		return BooleanEntitlementTemplate{}, errors.New("invalid entitlement template: not initialized")
	}

	return *e.boolean, nil
}

func (e *EntitlementTemplate) FromMetered(t MeteredEntitlementTemplate) {
	e.metered = &t
	e.t = entitlement.EntitlementTypeMetered
}

func (e *EntitlementTemplate) FromStatic(t StaticEntitlementTemplate) {
	e.static = &t
	e.t = entitlement.EntitlementTypeStatic
}

func (e *EntitlementTemplate) FromBoolean(t BooleanEntitlementTemplate) {
	e.boolean = &t
	e.t = entitlement.EntitlementTypeBoolean
}

func NewEntitlementTemplateFrom[T MeteredEntitlementTemplate | StaticEntitlementTemplate | BooleanEntitlementTemplate](c T) EntitlementTemplate {
	r := &EntitlementTemplate{}

	switch any(c).(type) {
	case MeteredEntitlementTemplate:
		e := any(c).(MeteredEntitlementTemplate)
		r.FromMetered(e)
	case StaticEntitlementTemplate:
		e := any(c).(StaticEntitlementTemplate)
		r.FromStatic(e)
	case BooleanEntitlementTemplate:
		e := any(c).(BooleanEntitlementTemplate)
		r.FromBoolean(e)
	}

	return *r
}

type EntitlementTemplateMeta struct {
	// Type defines the type of the entitlement.Entitlement.
	Type entitlement.EntitlementType `json:"type"`
}

type MeteredEntitlementTemplate struct {
	EntitlementTemplateMeta

	// Metadata a set of key/value pairs describing metadata for the RateCard.
	Metadata map[string]string `json:"metadata,omitempty"`

	// IsSoftLimit set to `true` for allowing the subject to use the feature even if the entitlement is exhausted.
	IsSoftLimit bool `json:"isSoftLimit,omitempty"`

	// IssueAfterReset defines the amount to be automatically granted at entitlement.Entitlement creation or reset.
	IssueAfterReset *float64 `json:"issueAfterReset,omitempty"`

	// IssueAfterResetPriority defines the grant priority for the default grant.
	IssueAfterResetPriority *uint8 `json:"issueAfterResetPriority,omitempty"`

	// PreserveOverageAtReset defines whether the overage is preserved after reset.
	PreserveOverageAtReset *bool `json:"preserveOverageAtReset,omitempty"`

	// UsagePeriod defines the interval of the entitlement in ISO8601 format.
	// Defaults to the billing cadence of the rate card.
	// Example: "P1D12H"
	UsagePeriod datex.Period `json:"usagePeriod,omitempty"`
}

func (t MeteredEntitlementTemplate) Validate() error {
	if t.IssueAfterResetPriority != nil && t.IssueAfterReset == nil {
		return errors.New("IssueAfterReset is required for IssueAfterResetPriority")
	}

	return nil
}

type StaticEntitlementTemplate struct {
	EntitlementTemplateMeta

	// Metadata a set of key/value pairs describing metadata for the RateCard.
	Metadata map[string]string `json:"metadata,omitempty"`

	// Config stores a JSON parsable configuration for the entitlement.Entitlement.
	// This value is also returned when checking entitlement access, and
	// it is useful for configuring fine-grained access settings to the feature implemented in customers own system.
	Config json.RawMessage `json:"config,omitempty"`
}

func (t StaticEntitlementTemplate) Validate() error {
	if len(t.Config) > 0 {
		if ok := json.Valid(t.Config); !ok {
			return errors.New("invalid JSON in config")
		}
	}

	return nil
}

type BooleanEntitlementTemplate struct {
	EntitlementTemplateMeta

	// Metadata a set of key/value pairs describing metadata for the RateCard.
	Metadata map[string]string `json:"metadata,omitempty"`
}

func (t BooleanEntitlementTemplate) Validate() error {
	return nil
}
