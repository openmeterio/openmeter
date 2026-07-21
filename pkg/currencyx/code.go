package currencyx

import (
	"errors"
	"fmt"
	"strings"

	"github.com/invopop/gobl/currency"

	"github.com/openmeterio/openmeter/pkg/models"
)

var (
	_ fmt.Stringer         = (*Code)(nil)
	_ models.Validator     = (*Code)(nil)
	_ models.Equaler[Code] = (*Code)(nil)
)

// Code represents a fiat or custom currency code. Code values used directly as
// Currency values are treated as fiat currencies for backwards compatibility.
type Code currency.Code

const (
	CustomCurrencyCodeMinLength = 4
	CustomCurrencyCodeMaxLength = 24
)

func (c Code) String() string {
	return string(c)
}

func (c Code) Equal(other Code) bool {
	return c == other
}

func (c Code) Type() CurrencyType {
	if len(c) == 3 {
		return CurrencyTypeFiat
	}

	return CurrencyTypeCustom
}

func (c Code) IsFiat() bool {
	return c.Type() == CurrencyTypeFiat
}

func (c Code) IsCustom() bool {
	return c.Type() == CurrencyTypeCustom
}

func (c Code) Validate() error {
	var errs []error

	if c == "" {
		errs = append(errs, errors.New("currency code is required"))

		return models.NewNillableGenericValidationError(errors.Join(errs...))
	}

	if len(c) == 3 {
		if err := validateFiatCurrencyCode(c); err != nil {
			errs = append(errs, err)
		}
	} else {
		if err := validateCustomCurrencyCode(c); err != nil {
			errs = append(errs, err)
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func validateFiatCurrencyCode(code Code) error {
	if len(code) != 3 {
		return fmt.Errorf("invalid fiat currency code: %s", code)
	}

	definition := currency.Get(currency.Code(code))
	if definition == nil || definition.ISONumeric == "" {
		return fmt.Errorf("invalid fiat currency code: %s", code)
	}

	return nil
}

func validateCustomCurrencyCode(code Code) error {
	var errs []error

	if code == "" {
		errs = append(errs, errors.New("currency code is required"))
	}

	codeString := code.String()
	if len(codeString) != len(strings.TrimSpace(codeString)) {
		errs = append(errs, fmt.Errorf("invalid currency code: cannot contain leading or trailing spaces: %s", code))
	}

	if strings.Contains(codeString, "|") {
		errs = append(errs, fmt.Errorf("invalid currency code: cannot contain route delimiter: %s", code))
	}

	if fiatDefinition := currency.Get(currency.Code(code)); fiatDefinition != nil {
		errs = append(errs, fmt.Errorf("currency code %s is a fiat currency", code))
	}

	if codeLength := len(codeString); codeLength < CustomCurrencyCodeMinLength || codeLength > CustomCurrencyCodeMaxLength {
		errs = append(errs, fmt.Errorf("invalid currency code: it must be between %d and %d characters", CustomCurrencyCodeMinLength, CustomCurrencyCodeMaxLength))
	}

	return errors.Join(errs...)
}
