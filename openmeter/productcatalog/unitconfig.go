package productcatalog

import (
	"errors"
	"fmt"
	"slices"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/models"
)

const (
	UnitConfigOperationMultiply UnitConfigOperation = "multiply"
	UnitConfigOperationDivide   UnitConfigOperation = "divide"
)

type UnitConfigOperation string

func (o UnitConfigOperation) Values() []string {
	return []string{
		string(UnitConfigOperationMultiply),
		string(UnitConfigOperationDivide),
	}
}

func (o UnitConfigOperation) Validate() error {
	if !slices.Contains(o.Values(), string(o)) {
		return fmt.Errorf("invalid unit config operation: %s", o)
	}

	return nil
}

const (
	UnitConfigRoundingModeNone    UnitConfigRoundingMode = "none"
	UnitConfigRoundingModeCeiling UnitConfigRoundingMode = "ceiling"
	UnitConfigRoundingModeFloor   UnitConfigRoundingMode = "floor"
	UnitConfigRoundingModeHalfUp  UnitConfigRoundingMode = "half_up"
)

type UnitConfigRoundingMode string

func (r UnitConfigRoundingMode) Values() []string {
	return []string{
		string(UnitConfigRoundingModeNone),
		string(UnitConfigRoundingModeCeiling),
		string(UnitConfigRoundingModeFloor),
		string(UnitConfigRoundingModeHalfUp),
	}
}

func (r UnitConfigRoundingMode) Validate() error {
	if !slices.Contains(r.Values(), string(r)) {
		return fmt.Errorf("invalid unit config rounding mode: %s", r)
	}

	return nil
}

// UnitConfig transforms a raw metered quantity into a billing quantity before
// pricing and entitlement evaluation. See unitconfig.md / UnitConfig.md for the
// conceptual model. The rating engine does not yet apply UnitConfig (Phase 3
// of the roadmap); persisted values currently round-trip but are inert.
type UnitConfig struct {
	// Operation is the arithmetic conversion to apply: multiply or divide.
	Operation UnitConfigOperation `json:"operation"`

	// ConversionFactor is the positive non-zero factor used by Operation.
	ConversionFactor decimal.Decimal `json:"conversion_factor"`

	// Rounding controls how the converted quantity is rounded for invoicing.
	// Entitlement balance checks always use the unrounded value.
	Rounding *UnitConfigRoundingMode `json:"rounding,omitempty"`

	// Precision is the decimal places retained after rounding. Only meaningful
	// when Rounding is set and not "none". Nil means round to whole numbers.
	Precision *int `json:"precision,omitempty"`

	// DisplayUnit is a human-readable label for the converted unit shown on
	// invoices and the customer portal (e.g. "GB", "hours").
	DisplayUnit *string `json:"display_unit,omitempty"`
}

func (c *UnitConfig) Validate() error {
	if c == nil {
		return nil
	}

	var errs []error

	if err := c.Operation.Validate(); err != nil {
		errs = append(errs, err)
	}

	if c.ConversionFactor.IsNegative() {
		errs = append(errs, errors.New("conversion_factor must not be negative"))
	}

	if c.ConversionFactor.IsZero() {
		errs = append(errs, errors.New("conversion_factor must not be zero"))
	}

	if c.Rounding != nil {
		if err := c.Rounding.Validate(); err != nil {
			errs = append(errs, err)
		}
	}

	if c.Precision != nil && *c.Precision < 0 {
		errs = append(errs, errors.New("precision must not be negative"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (c *UnitConfig) Equal(v *UnitConfig) bool {
	if c == nil && v == nil {
		return true
	}

	if c == nil || v == nil {
		return false
	}

	if c.Operation != v.Operation {
		return false
	}

	if !c.ConversionFactor.Equal(v.ConversionFactor) {
		return false
	}

	if lo.FromPtr(c.Rounding) != lo.FromPtr(v.Rounding) {
		return false
	}

	if lo.FromPtr(c.Precision) != lo.FromPtr(v.Precision) {
		return false
	}

	if lo.FromPtr(c.DisplayUnit) != lo.FromPtr(v.DisplayUnit) {
		return false
	}

	return true
}

// Apply transforms a raw metered quantity into the converted (precise) and
// invoiced (rounded) billing quantities.
//
//   - converted: raw with the operation × conversion_factor applied. Used for
//     entitlement balance checks, which always see the precise value.
//   - invoiced: converted with the configured rounding/precision applied. Used
//     as the line quantity at billing time. Equal to converted when no
//     rounding is set or rounding is "none".
//
// When c is nil, both return values equal raw. Callers must have already run
// Validate so the operation and rounding enums are known values; an
// unrecognized operation falls back to identity rather than panicking
// mid-billing.
func (c *UnitConfig) Apply(raw decimal.Decimal) (converted, invoiced decimal.Decimal) {
	if c == nil {
		return raw, raw
	}

	switch c.Operation {
	case UnitConfigOperationMultiply:
		converted = raw.Mul(c.ConversionFactor)
	case UnitConfigOperationDivide:
		converted = raw.Div(c.ConversionFactor)
	default:
		return raw, raw
	}

	invoiced = converted

	if c.Rounding == nil {
		return converted, invoiced
	}

	places := int32(0)
	if c.Precision != nil {
		places = int32(*c.Precision)
	}

	switch *c.Rounding {
	case UnitConfigRoundingModeCeiling:
		invoiced = converted.RoundCeil(places)
	case UnitConfigRoundingModeFloor:
		invoiced = converted.RoundFloor(places)
	case UnitConfigRoundingModeHalfUp:
		invoiced = converted.Round(places)
	case UnitConfigRoundingModeNone:
	}

	return converted, invoiced
}

func (c UnitConfig) Clone() UnitConfig {
	out := UnitConfig{
		Operation:        c.Operation,
		ConversionFactor: c.ConversionFactor.Copy(),
	}

	if c.Rounding != nil {
		out.Rounding = lo.ToPtr(*c.Rounding)
	}

	if c.Precision != nil {
		out.Precision = lo.ToPtr(*c.Precision)
	}

	if c.DisplayUnit != nil {
		out.DisplayUnit = lo.ToPtr(*c.DisplayUnit)
	}

	return out
}
