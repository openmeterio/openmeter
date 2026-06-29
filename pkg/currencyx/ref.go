package currencyx

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/pkg/models"
)

type CurrencyRef struct {
	CurrencyCode *Code   `json:"currency_code"`
	CurrencyID   *string `json:"currency_id"`
}

func (r CurrencyRef) Validate() error {
	var errs []error

	if r.CurrencyCode == nil && r.CurrencyID == nil {
		errs = append(errs, errors.New("either currency code or currency ID must be provided"))
	}

	if err := r.CurrencyCode.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("invalid currency code: %s", r.CurrencyCode))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
