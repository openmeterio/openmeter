package currencyx

import (
	"errors"
	"fmt"

	"github.com/invopop/gobl/currency"

	"github.com/openmeterio/openmeter/pkg/models"
)

type CurrencyType string

const (
	CurrencyTypeFiat   CurrencyType = "fiat"
	CurrencyTypeCustom CurrencyType = "custom"
)

func (t CurrencyType) Validate() error {
	var errs []error

	switch t {
	case CurrencyTypeFiat, CurrencyTypeCustom:
	default:
		errs = append(errs, fmt.Errorf("invalid currency type: %s", t))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type Currency interface {
	CurrencyCode() Code
	CurrencyType() CurrencyType
	CurrencyPrecision() int32
	CurrencyRoundingMode() RoundingMode
}

// Code represents a fiat or custom currency code. Code values used directly as
// Currency values are treated as fiat currencies for backwards compatibility.
type Code currency.Code

func (c Code) String() string {
	return string(c)
}

func (c Code) CurrencyCode() Code {
	return c
}

func (c Code) CurrencyType() CurrencyType {
	return CurrencyTypeFiat
}

func (c Code) CurrencyPrecision() int32 {
	def := currency.Get(currency.Code(c))
	if def == nil {
		return 0
	}

	return int32(def.Subunits)
}

func (c Code) CurrencyRoundingMode() RoundingMode {
	return RoundingModeHalfAwayFromZero
}

type CustomCurrency struct {
	Code         Code
	Precision    int32
	RoundingMode RoundingMode
}

func NewCurrency(code Code, currencyType CurrencyType, precision int32) (Currency, error) {
	switch currencyType {
	case CurrencyTypeFiat:
		return NewFiatCurrency(code)
	case CurrencyTypeCustom:
		return NewCustomCurrency(code, precision)
	default:
		return nil, currencyType.Validate()
	}
}

func NewFiatCurrency(code Code) (Code, error) {
	def := currency.Get(currency.Code(code))
	if def == nil || def.ISONumeric == "" {
		return "", fmt.Errorf("fiat currency definition is required for %s", code)
	}

	if err := code.Validate(); err != nil {
		return "", err
	}

	return code, nil
}

func NewCustomCurrency(code Code, precision int32) (CustomCurrency, error) {
	return NewCustomCurrencyWithRounding(code, precision, RoundingModeBankers)
}

func NewCustomCurrencyWithRounding(code Code, precision int32, roundingMode RoundingMode) (CustomCurrency, error) {
	out := CustomCurrency{
		Code:         code,
		Precision:    precision,
		RoundingMode: roundingMode,
	}

	return out, out.Validate()
}

func (c CustomCurrency) CurrencyCode() Code {
	return c.Code
}

func (c CustomCurrency) CurrencyType() CurrencyType {
	return CurrencyTypeCustom
}

func (c CustomCurrency) CurrencyPrecision() int32 {
	return c.Precision
}

func (c CustomCurrency) CurrencyRoundingMode() RoundingMode {
	if c.RoundingMode == "" {
		return RoundingModeBankers
	}

	return c.RoundingMode
}
