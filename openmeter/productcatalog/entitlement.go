package productcatalog

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/hasher"
	"github.com/openmeterio/openmeter/pkg/models"
)

type entitlementTemplater interface {
	json.Marshaler
	json.Unmarshaler
	models.Validator
	models.Equaler[*EntitlementTemplate]

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

func (e *EntitlementTemplate) Equal(v *EntitlementTemplate) bool {
	if e == nil && v == nil {
		return true
	}

	if e == nil || v == nil {
		return false
	}

	if e.t != v.t {
		return false
	}

	switch e.t {
	case entitlement.EntitlementTypeMetered:
		return e.metered.Equal(v.metered)
	case entitlement.EntitlementTypeStatic:
		return e.static.Equal(v.static)
	case entitlement.EntitlementTypeBoolean:
		return e.boolean.Equal(v.boolean)
	default:
		return false
	}
}

func (e *EntitlementTemplate) MarshalJSON() ([]byte, error) {
	var b []byte
	var err error
	var serde interface{}

	switch e.t {
	case entitlement.EntitlementTypeMetered:
		serde = struct {
			Type entitlement.EntitlementType `json:"type"`
			*MeteredEntitlementTemplate
		}{
			Type:                       entitlement.EntitlementTypeMetered,
			MeteredEntitlementTemplate: e.metered,
		}
	case entitlement.EntitlementTypeStatic:
		serde = struct {
			Type entitlement.EntitlementType `json:"type"`
			*StaticEntitlementTemplate
		}{
			Type:                      entitlement.EntitlementTypeStatic,
			StaticEntitlementTemplate: e.static,
		}
	case entitlement.EntitlementTypeBoolean:
		serde = struct {
			Type entitlement.EntitlementType `json:"type"`
			*BooleanEntitlementTemplate
		}{
			Type:                       entitlement.EntitlementTypeBoolean,
			BooleanEntitlementTemplate: e.boolean,
		}
	default:
		return nil, fmt.Errorf("invalid Entitlement type: %s", e.t)
	}

	b, err = json.Marshal(serde)
	if err != nil {
		return nil, fmt.Errorf("failed to JSON serialize EntitlementTemplate: %w", err)
	}

	return b, nil
}

func (e *EntitlementTemplate) UnmarshalJSON(bytes []byte) error {
	serde := struct {
		Type entitlement.EntitlementType `json:"type"`
	}{}

	if err := json.Unmarshal(bytes, &serde); err != nil {
		return fmt.Errorf("failed to JSON deserialize EntitlementTemplate type: %w", err)
	}

	switch serde.Type {
	case entitlement.EntitlementTypeMetered:
		v := &MeteredEntitlementTemplate{}
		if err := json.Unmarshal(bytes, v); err != nil {
			return fmt.Errorf("failed to JSON deserialize EntitlementTemplate: %w", err)
		}

		e.metered = v
		e.t = entitlement.EntitlementTypeMetered
	case entitlement.EntitlementTypeStatic:
		v := &StaticEntitlementTemplate{}
		if err := json.Unmarshal(bytes, v); err != nil {
			return fmt.Errorf("failed to JSON deserialize EntitlementTemplate: %w", err)
		}

		e.static = v
		e.t = entitlement.EntitlementTypeStatic
	case entitlement.EntitlementTypeBoolean:
		v := &BooleanEntitlementTemplate{}
		if err := json.Unmarshal(bytes, v); err != nil {
			return fmt.Errorf("failed to JSON deserialize EntitlementTemplate: %w", err)
		}

		e.boolean = v
		e.t = entitlement.EntitlementTypeBoolean
	default:
		return fmt.Errorf("invalid EntitlementTemplate type: %s", serde.Type)
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
		return fmt.Errorf("invalid entitlement template type: %q", e.t)
	}
}

func (e *EntitlementTemplate) Type() entitlement.EntitlementType {
	return e.t
}

func (e *EntitlementTemplate) AsMetered() (MeteredEntitlementTemplate, error) {
	switch e.t {
	case entitlement.EntitlementTypeMetered:
		if e.metered == nil {
			return MeteredEntitlementTemplate{}, errors.New("invalid metered entitlement template: not initialized")
		}

		return *e.metered, nil
	case entitlement.EntitlementTypeBoolean, entitlement.EntitlementTypeStatic:
		return MeteredEntitlementTemplate{}, fmt.Errorf("invalid entitlement template: type mismatch: %s", e.t)
	default:
		return MeteredEntitlementTemplate{}, errors.New("invalid entitlement template: not initialized")
	}
}

func (e *EntitlementTemplate) AsStatic() (StaticEntitlementTemplate, error) {
	switch e.t {
	case entitlement.EntitlementTypeStatic:
		if e.static == nil {
			return StaticEntitlementTemplate{}, errors.New("invalid static entitlement template: not initialized")
		}

		return *e.static, nil
	case entitlement.EntitlementTypeBoolean, entitlement.EntitlementTypeMetered:
		return StaticEntitlementTemplate{}, fmt.Errorf("invalid entitlement template: type mismatch: %s", e.t)
	default:
		return StaticEntitlementTemplate{}, errors.New("invalid entitlement template: not initialized")
	}
}

func (e *EntitlementTemplate) AsBoolean() (BooleanEntitlementTemplate, error) {
	switch e.t {
	case entitlement.EntitlementTypeBoolean:
		if e.boolean == nil {
			return BooleanEntitlementTemplate{}, errors.New("invalid boolean entitlement template: not initialized")
		}

		return *e.boolean, nil
	case entitlement.EntitlementTypeStatic, entitlement.EntitlementTypeMetered:
		return BooleanEntitlementTemplate{}, fmt.Errorf("invalid entitlement template: type mismatch: %s", e.t)
	default:
		return BooleanEntitlementTemplate{}, errors.New("invalid entitlement template: not initialized")
	}
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

func NewEntitlementTemplateFrom[T MeteredEntitlementTemplate | StaticEntitlementTemplate | BooleanEntitlementTemplate](c T) *EntitlementTemplate {
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

	return r
}

var (
	_ models.Validator                            = (*MeteredEntitlementTemplate)(nil)
	_ models.Equaler[*MeteredEntitlementTemplate] = (*MeteredEntitlementTemplate)(nil)
)

type MeteredEntitlementTemplate struct {
	// Metadata a set of key/value pairs describing metadata for the RateCard.
	Metadata models.Metadata `json:"metadata,omitempty"`

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
	UsagePeriod datetime.ISODuration `json:"usagePeriod,omitempty"`
}

func (t *MeteredEntitlementTemplate) Equal(v *MeteredEntitlementTemplate) bool {
	if t == nil && v == nil {
		return true
	}

	if t == nil || v == nil {
		return false
	}

	if !t.Metadata.Equal(v.Metadata) {
		return false
	}

	if !t.IsSoftLimit && v.IsSoftLimit {
		return false
	}

	if (t.IssueAfterReset != nil && v.IssueAfterReset == nil) || (t.IssueAfterReset == nil && v.IssueAfterReset != nil) {
		return false
	}

	if lo.FromPtr(t.IssueAfterReset) != lo.FromPtr(v.IssueAfterReset) {
		return false
	}

	if (t.IssueAfterReset != nil && v.IssueAfterReset == nil) ||
		(t.IssueAfterReset == nil && v.IssueAfterReset != nil) {
		return false
	}

	if lo.FromPtr(t.IssueAfterReset) != lo.FromPtr(v.IssueAfterReset) {
		return false
	}

	if (t.IssueAfterResetPriority != nil && v.IssueAfterResetPriority == nil) ||
		(t.IssueAfterResetPriority == nil && v.IssueAfterResetPriority != nil) {
		return false
	}

	if lo.FromPtr(t.IssueAfterResetPriority) != lo.FromPtr(v.IssueAfterResetPriority) {
		return false
	}

	if (t.PreserveOverageAtReset != nil && v.PreserveOverageAtReset == nil) ||
		(t.PreserveOverageAtReset == nil && v.PreserveOverageAtReset != nil) {
		return false
	}

	if lo.FromPtr(t.PreserveOverageAtReset) != lo.FromPtr(v.PreserveOverageAtReset) {
		return false
	}

	return t.UsagePeriod.ISOString() == v.UsagePeriod.ISOString()
}

func (t *MeteredEntitlementTemplate) Validate() error {
	var errs []error

	if t.IssueAfterResetPriority != nil && t.IssueAfterReset == nil {
		errs = append(errs, ErrEntitlementTemplateInvalidIssueAfterResetWithPriority)
	}

	if t.UsagePeriod.Sign() != 1 {
		errs = append(errs, ErrEntitlementTemplateNegativeUsagePeriod)
	}

	hour := datetime.NewISODuration(0, 0, 0, 0, 1, 0, 0)
	if diff, err := t.UsagePeriod.Subtract(hour); err == nil && diff.Sign() == -1 {
		errs = append(errs, ErrEntitlementTemplateUsagePeriodLessThenAnHour)
	}

	if err := errors.Join(errs...); err != nil {
		return models.NewGenericValidationError(
			models.ErrorWithFieldPrefix(
				models.NewFieldSelectorGroup(models.NewFieldSelector("entitlementTemplate")),
				err),
		)
	}

	return nil
}

var (
	_ models.Validator                           = (*StaticEntitlementTemplate)(nil)
	_ models.Equaler[*StaticEntitlementTemplate] = (*StaticEntitlementTemplate)(nil)
)

type StaticEntitlementTemplate struct {
	// Metadata a set of key/value pairs describing metadata for the RateCard.
	Metadata models.Metadata `json:"metadata,omitempty"`

	// Config stores a JSON parsable configuration for the entitlement.Entitlement.
	// This value is also returned when checking entitlement access, and
	// it is useful for configuring fine-grained access settings to the feature implemented in customers own system.
	Config json.RawMessage `json:"config,omitempty"`
}

func (t *StaticEntitlementTemplate) Equal(v *StaticEntitlementTemplate) bool {
	if t == nil && v == nil {
		return true
	}

	if t == nil || v == nil {
		return false
	}

	if !t.Metadata.Equal(v.Metadata) {
		return false
	}

	return hasher.NewHash(t.Config) == hasher.NewHash(v.Config)
}

func (t *StaticEntitlementTemplate) Validate() error {
	if len(t.Config) > 0 {
		if ok := json.Valid(t.Config); !ok {
			return models.NewGenericValidationError(ErrEntitlementTemplateInvalidJSONConfig)
		}
	}

	return nil
}

var (
	_ models.Validator                            = (*BooleanEntitlementTemplate)(nil)
	_ models.Equaler[*BooleanEntitlementTemplate] = (*BooleanEntitlementTemplate)(nil)
)

type BooleanEntitlementTemplate struct {
	// Metadata a set of key/value pairs describing metadata for the RateCard.
	Metadata models.Metadata `json:"metadata,omitempty"`
}

func (t *BooleanEntitlementTemplate) Equal(v *BooleanEntitlementTemplate) bool {
	if t == nil && v == nil {
		return true
	}

	if t == nil || v == nil {
		return false
	}

	return t.Metadata.Equal(v.Metadata)
}

func (t *BooleanEntitlementTemplate) Validate() error {
	return nil
}
