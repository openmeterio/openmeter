package currencyx

import (
	"errors"
	"fmt"
	"strings"

	"github.com/invopop/gobl/currency"

	"github.com/openmeterio/openmeter/pkg/models"
)

const (
	MinCodeLength = 3
	MaxCodeLength = 24
	MaxPrecision  = 12

	PostgresCodeSchemaType = "varchar(24)"
)

func (c Code) Validate() error {
	if c == "" {
		return errors.New("currency code is required")
	}

	return currency.Code(c).Validate()
}

func (c Code) ValidateFormat() error {
	value := c.String()
	if value == "" {
		return errors.New("currency code is required")
	}

	if strings.TrimSpace(value) != value {
		return errors.New("currency code must not contain leading or trailing whitespace")
	}

	if len(value) < MinCodeLength {
		return fmt.Errorf("currency code must be at least %d characters", MinCodeLength)
	}

	if len(value) > MaxCodeLength {
		return fmt.Errorf("currency code must be at most %d characters", MaxCodeLength)
	}

	if strings.Contains(value, "|") {
		return errors.New("currency code cannot contain route delimiter")
	}

	return nil
}

func (c Code) ValidateCustom() error {
	if err := c.ValidateFormat(); err != nil {
		return err
	}

	if c.IsKnownFiat() {
		return fmt.Errorf("custom currency code %s conflicts with fiat currency code", c)
	}

	return nil
}

func (c Code) IsKnownFiat() bool {
	return c.Validate() == nil && currency.Get(currency.Code(c)) != nil
}

func (c CustomCurrency) Validate() error {
	var errs []error

	if err := c.Code.ValidateCustom(); err != nil {
		errs = append(errs, err)
	}

	if err := validatePrecision(c.Precision); err != nil {
		errs = append(errs, err)
	}

	if err := c.CurrencyRoundingMode().Validate(); err != nil {
		errs = append(errs, err)
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func validatePrecision(precision int32) error {
	if precision < 0 {
		return errors.New("currency precision must be non-negative")
	}

	if precision > MaxPrecision {
		return fmt.Errorf("currency precision must be at most %d", MaxPrecision)
	}

	return nil
}
