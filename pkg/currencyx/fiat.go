package currencyx

import (
	"errors"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"

	"github.com/openmeterio/openmeter/pkg/models"
)

type RoundingMode string

const (
	RoundingModeHalfAwayFromZero RoundingMode = "half_away_from_zero"
	RoundingModeBankers          RoundingMode = "bankers"
)

func (m RoundingMode) Validate() error {
	var errs []error

	switch m {
	case RoundingModeHalfAwayFromZero, RoundingModeBankers:
		return nil
	default:
		errs = append(errs, fmt.Errorf("invalid rounding mode: %s", m))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// Calculator provides a currency calculator object. This allows callers to
// resolve fiat definitions once and then apply currency precision consistently.
func (c Code) Calculator() (Calculator, error) {
	return NewCalculator(c)
}

type Calculator struct {
	currency     Code
	def          *currency.Def
	currencyType CurrencyType
	precision    int32
	roundingMode RoundingMode
}

func NewCalculator(cur Currency) (Calculator, error) {
	if cur == nil {
		return Calculator{}, errors.New("currency is required")
	}

	code := cur.CurrencyCode()
	currencyType := cur.CurrencyType()
	precision := cur.CurrencyPrecision()
	roundingMode := cur.CurrencyRoundingMode()

	switch currencyType {
	case CurrencyTypeFiat:
		if err := code.Validate(); err != nil {
			return Calculator{}, err
		}

		def := currency.Get(currency.Code(code))
		if def == nil || def.ISONumeric == "" {
			return Calculator{}, fmt.Errorf("fiat currency definition is required for %s", code)
		}

		return Calculator{
			currency:     code,
			def:          def,
			currencyType: CurrencyTypeFiat,
			precision:    int32(def.Subunits),
			roundingMode: RoundingModeHalfAwayFromZero,
		}, nil
	case CurrencyTypeCustom:
		if err := code.ValidateCustom(); err != nil {
			return Calculator{}, err
		}

		if err := validatePrecision(precision); err != nil {
			return Calculator{}, err
		}

		if err := roundingMode.Validate(); err != nil {
			return Calculator{}, err
		}

		return Calculator{
			currency:     code,
			currencyType: CurrencyTypeCustom,
			precision:    precision,
			roundingMode: roundingMode,
		}, nil
	default:
		return Calculator{}, currencyType.Validate()
	}
}

func (c Calculator) CurrencyCode() Code {
	return c.currency
}

func (c Calculator) Definition() *currency.Def {
	return c.def
}

func (c Calculator) CurrencyType() CurrencyType {
	return c.currencyType
}

func (c Calculator) CurrencyPrecision() int32 {
	return c.precision
}

func (c Calculator) RoundingMode() RoundingMode {
	return c.roundingMode
}

func (c Calculator) RoundToPrecision(amount alpacadecimal.Decimal) alpacadecimal.Decimal {
	// TODO: For now we are skipping the smallestDenomination, as that is a reference to the coins
	// in circulation, but should not be an issue for online payments.
	switch c.roundingMode {
	case RoundingModeBankers:
		return amount.RoundBank(c.precision)
	default:
		return amount.Round(c.precision)
	}
}

func (c Calculator) RoundDown(amount alpacadecimal.Decimal) alpacadecimal.Decimal {
	return amount.RoundDown(c.precision)
}

func (c Calculator) Unit() alpacadecimal.Decimal {
	return alpacadecimal.NewFromInt(1).Shift(-c.precision)
}

func (c Calculator) Validate() error {
	var errs []error

	switch c.currencyType {
	case CurrencyTypeFiat:
		if err := c.currency.Validate(); err != nil {
			errs = append(errs, err)
		}
		if c.def == nil || c.def.ISONumeric == "" {
			errs = append(errs, fmt.Errorf("fiat currency definition is required for %s", c.currency))
		}
		if c.def != nil && c.precision != int32(c.def.Subunits) {
			errs = append(errs, errors.New("fiat currency precision must match currency definition"))
		}
	case CurrencyTypeCustom:
		if err := c.currency.ValidateCustom(); err != nil {
			errs = append(errs, err)
		}
		if err := validatePrecision(c.precision); err != nil {
			errs = append(errs, err)
		}
		if err := c.roundingMode.Validate(); err != nil {
			errs = append(errs, err)
		}
	default:
		if c.currency == "" {
			errs = append(errs, errors.New("currency code is required"))
		}
		errs = append(errs, c.currencyType.Validate())
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (c Calculator) IsRoundedToPrecision(amount alpacadecimal.Decimal) bool {
	return amount.Equal(c.RoundToPrecision(amount))
}
