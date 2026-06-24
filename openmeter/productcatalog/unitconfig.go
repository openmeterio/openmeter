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

// IsNone reports whether the mode applies no rounding. The empty string is the
// zero value of the field and is treated as the default, "none".
func (r UnitConfigRoundingMode) IsNone() bool {
	return r == "" || r == UnitConfigRoundingModeNone
}

func (r UnitConfigRoundingMode) Validate() error {
	// The empty string is the field's zero value and means the default ("none").
	if r == "" {
		return nil
	}

	if !slices.Contains(r.Values(), string(r)) {
		return fmt.Errorf("invalid unit config rounding mode: %s", r)
	}

	return nil
}

// UnitConfig transforms a raw metered quantity into a billing quantity before
// pricing and entitlement evaluation. It is a self-contained domain DTO with no
// dependency on the persistence layer — persistence maps to/from it.
//
// Rounding and Precision are value types with sensible defaults (no rounding,
// zero decimal places); only DisplayUnit is optional and pointer-typed.
type UnitConfig struct {
	// Operation is the arithmetic conversion to apply: multiply or divide.
	Operation UnitConfigOperation `json:"operation"`

	// ConversionFactor is the positive non-zero factor used by Operation. Full
	// precision is retained; the factor is never capped.
	ConversionFactor decimal.Decimal `json:"conversionFactor"`

	// Rounding controls how the converted quantity is rounded for invoicing.
	// Defaults to "none". Entitlement balance checks always use the unrounded
	// (converted) value, never the rounded one.
	Rounding UnitConfigRoundingMode `json:"rounding,omitempty"`

	// Precision is the number of decimal places retained when rounding. It is
	// only meaningful when Rounding is set to a value other than "none"; it is
	// ignored otherwise. Defaults to 0 (whole units).
	Precision int `json:"precision,omitempty"`

	// DisplayUnit is a human-readable label for the converted unit shown on
	// invoices and the customer portal (e.g. "GB", "hours").
	DisplayUnit *string `json:"displayUnit,omitempty"`
}

func (c *UnitConfig) Validate() error {
	if c == nil {
		return nil
	}

	var errs []error

	if err := c.Operation.Validate(); err != nil {
		errs = append(errs, err)
	}

	if c.ConversionFactor.Sign() <= 0 {
		errs = append(errs, errors.New("conversion_factor must be greater than zero"))
	}

	if err := c.Rounding.Validate(); err != nil {
		errs = append(errs, err)
	}

	// Precision is ignored when no rounding is applied, so only enforce it when
	// rounding is active.
	if !c.Rounding.IsNone() && c.Precision < 0 {
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

	// Treat the empty zero value and the explicit "none" as the same mode.
	if c.Rounding.IsNone() != v.Rounding.IsNone() {
		return false
	}

	if !c.Rounding.IsNone() && c.Rounding != v.Rounding {
		return false
	}

	if c.Precision != v.Precision {
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
//     as the line quantity at billing time. Equal to converted when no rounding
//     is set.
//
// When c is nil, both return values equal raw. Callers must have already run
// Validate so the operation and rounding enums are known values; an
// unrecognized operation or rounding mode falls back to identity rather than
// panicking mid-billing.
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

	places := int32(c.Precision)

	switch c.Rounding {
	case UnitConfigRoundingModeCeiling:
		invoiced = converted.RoundCeil(places)
	case UnitConfigRoundingModeFloor:
		invoiced = converted.RoundFloor(places)
	case UnitConfigRoundingModeHalfUp:
		// Round rounds half away from zero (2.5 → 3, −2.5 → −3).
		invoiced = converted.Round(places)
	case UnitConfigRoundingModeNone, "":
		// No rounding: invoiced stays equal to converted.
	default:
		// Unknown mode: identity. Validate rejects it before billing.
	}

	return converted, invoiced
}

func (c UnitConfig) Clone() UnitConfig {
	out := UnitConfig{
		Operation:        c.Operation,
		ConversionFactor: c.ConversionFactor.Copy(),
		Rounding:         c.Rounding,
		Precision:        c.Precision,
	}

	if c.DisplayUnit != nil {
		out.DisplayUnit = lo.ToPtr(*c.DisplayUnit)
	}

	return out
}
