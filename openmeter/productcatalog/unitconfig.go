package productcatalog

import (
	"errors"
	"fmt"
	"math"
	"slices"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/models"
)

const (
	ConversionOperationDivide   ConversionOperation = "divide"
	ConversionOperationMultiply ConversionOperation = "multiply"
)

type ConversionOperation string

func (c ConversionOperation) Values() []string {
	return []string{
		string(ConversionOperationDivide),
		string(ConversionOperationMultiply),
	}
}

const (
	RoundingModeCeiling RoundingMode = "ceiling"
	RoundingModeFloor   RoundingMode = "floor"
	RoundingModeHalfUp  RoundingMode = "half_up"
	RoundingModeNone    RoundingMode = "none"
)

type RoundingMode string

func (r RoundingMode) Values() []string {
	return []string{
		string(RoundingModeCeiling),
		string(RoundingModeFloor),
		string(RoundingModeHalfUp),
		string(RoundingModeNone),
	}
}

// UnitConfig defines how to convert raw metered quantities into billing units.
// It is applied before pricing runs on usage-based rate cards.
type UnitConfig struct {
	// Operation defines whether to divide or multiply the metered quantity by the factor.
	Operation ConversionOperation `json:"operation"`

	// ConversionFactor is the factor applied to the metered quantity.
	// For example, divide by 1e9 to convert bytes to GB, or multiply by 1.2 for a 20% margin.
	ConversionFactor decimal.Decimal `json:"conversion_factor"`

	// Rounding defines how to round the converted quantity for invoicing.
	// Defaults to RoundingModeNone.
	Rounding RoundingMode `json:"rounding"`

	// Precision defines the number of decimal places for rounding.
	// Only used when Rounding is not RoundingModeNone.
	Precision int `json:"precision"`

	// DisplayUnit is a human-readable unit label shown on invoices (e.g., "GB", "hours", "M").
	DisplayUnit *string `json:"display_unit,omitempty"`
}

var _ models.Validator = (*UnitConfig)(nil)

func (u *UnitConfig) Validate() error {
	if u == nil {
		return nil
	}

	var errs []error

	if !slices.Contains(ConversionOperation("").Values(), string(u.Operation)) {
		errs = append(errs, fmt.Errorf("invalid conversion operation: %q", u.Operation))
	}

	if u.ConversionFactor.IsZero() {
		errs = append(errs, errors.New("conversion_factor must not be zero"))
	}

	if u.ConversionFactor.IsNegative() {
		errs = append(errs, errors.New("conversion_factor must be positive"))
	}

	if u.Rounding != "" && !slices.Contains(RoundingMode("").Values(), string(u.Rounding)) {
		errs = append(errs, fmt.Errorf("invalid rounding mode: %q", u.Rounding))
	}

	if u.Precision < 0 {
		errs = append(errs, errors.New("precision must not be negative"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (u *UnitConfig) Equal(v *UnitConfig) bool {
	if u == nil && v == nil {
		return true
	}

	if u == nil || v == nil {
		return false
	}

	if u.Operation != v.Operation {
		return false
	}

	if !u.ConversionFactor.Equal(v.ConversionFactor) {
		return false
	}

	if u.Rounding != v.Rounding {
		return false
	}

	if u.Precision != v.Precision {
		return false
	}

	if lo.FromPtr(u.DisplayUnit) != lo.FromPtr(v.DisplayUnit) {
		return false
	}

	return true
}

func (u *UnitConfig) Clone() UnitConfig {
	clone := UnitConfig{
		Operation:        u.Operation,
		ConversionFactor: u.ConversionFactor.Copy(),
		Rounding:         u.Rounding,
		Precision:        u.Precision,
	}

	if u.DisplayUnit != nil {
		du := *u.DisplayUnit
		clone.DisplayUnit = &du
	}

	return clone
}

// Convert applies the conversion operation and factor to the given quantity
// without rounding. Use this for entitlement balance checks where precision matters.
func (u *UnitConfig) Convert(qty decimal.Decimal) decimal.Decimal {
	if u == nil {
		return qty
	}

	switch u.Operation {
	case ConversionOperationDivide:
		return qty.Div(u.ConversionFactor)
	case ConversionOperationMultiply:
		return qty.Mul(u.ConversionFactor)
	default:
		return qty
	}
}

// ConvertAndRound applies the conversion operation, factor, and rounding to the given quantity.
// Use this for billing/invoicing quantities.
func (u *UnitConfig) ConvertAndRound(qty decimal.Decimal) decimal.Decimal {
	if u == nil {
		return qty
	}

	converted := u.Convert(qty)
	return u.Round(converted)
}

// Round applies the rounding mode and precision to the given value.
func (u *UnitConfig) Round(value decimal.Decimal) decimal.Decimal {
	if u == nil {
		return value
	}

	rounding := u.Rounding
	if rounding == "" {
		rounding = RoundingModeNone
	}

	switch rounding {
	case RoundingModeNone:
		return value
	case RoundingModeCeiling:
		return roundCeil(value, u.Precision)
	case RoundingModeFloor:
		return roundFloor(value, u.Precision)
	case RoundingModeHalfUp:
		return roundHalfUp(value, u.Precision)
	default:
		return value
	}
}

// roundCeil rounds up to the given number of decimal places.
func roundCeil(value decimal.Decimal, precision int) decimal.Decimal {
	scale := decimal.NewFromFloat(math.Pow10(precision))
	scaled := value.Mul(scale)

	// If already an integer at this scale, no rounding needed
	truncated := scaled.Truncate(0)
	if scaled.Equal(truncated) {
		return value
	}

	// Round up: truncate and add 1 if positive, just truncate if negative
	if value.IsPositive() {
		return truncated.Add(decimal.NewFromInt(1)).Div(scale)
	}

	return truncated.Div(scale)
}

// roundFloor rounds down to the given number of decimal places.
func roundFloor(value decimal.Decimal, precision int) decimal.Decimal {
	scale := decimal.NewFromFloat(math.Pow10(precision))
	scaled := value.Mul(scale)

	truncated := scaled.Truncate(0)
	if scaled.Equal(truncated) {
		return value
	}

	if value.IsNegative() {
		return truncated.Sub(decimal.NewFromInt(1)).Div(scale)
	}

	return truncated.Div(scale)
}

// roundHalfUp rounds to the nearest value, with ties going up.
func roundHalfUp(value decimal.Decimal, precision int) decimal.Decimal {
	scale := decimal.NewFromFloat(math.Pow10(precision))
	scaled := value.Mul(scale)

	half := decimal.NewFromFloat(0.5)
	if value.IsNegative() {
		half = half.Neg()
	}

	return scaled.Add(half).Truncate(0).Div(scale)
}
