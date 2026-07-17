package currencyx

import (
	"errors"
	"fmt"
	"strings"

	"github.com/invopop/gobl/currency"

	"github.com/openmeterio/openmeter/pkg/models"
)

var _ fmt.Stringer = (*Code)(nil)
var _ CurrencyIdentity = (*Code)(nil)

// Code represents a fiat or custom currency code.
type Code currency.Code

func (c Code) String() string {
	return string(c)
}

func (c Code) Validate() error {
	var errs []error

	if c == "" {
		errs = append(errs, errors.New("currency code is required"))
	}

	code := c.String()
	if len(code) != len(strings.TrimSpace(code)) {
		errs = append(errs, errors.New("currency code cannot contain leading or trailing spaces"))
	}

	if strings.Contains(code, "|") {
		errs = append(errs, errors.New("currency code cannot contain route delimiter"))
	}

	if !c.IsFiat() {
		if length := len(code); length < CustomCurrencyCodeMinLength || length > CustomCurrencyCodeMaxLength {
			errs = append(errs, fmt.Errorf("custom currency code must be between %d and %d characters", CustomCurrencyCodeMinLength, CustomCurrencyCodeMaxLength))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// IsFiat reports whether the code identifies an ISO 4217 currency.
func (c Code) IsFiat() bool {
	definition := currency.Get(currency.Code(c))

	return definition != nil && definition.ISONumeric != ""
}

func (c Code) IsCustom() bool {
	return !c.IsFiat()
}

func (c Code) Type() CurrencyType {
	if c.IsFiat() {
		return CurrencyTypeFiat
	}

	return CurrencyTypeCustom
}

func (c Code) GetCode() Code {
	return c
}

func (c Code) Equal(other CurrencyIdentity) bool {
	if other == nil || c.Type() != other.Type() {
		return false
	}

	if c.IsCustom() {
		if _, managed := other.(ManagedCurrency); managed {
			return false
		}
	}

	return c == other.GetCode()
}
