package charges

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

type CreateChargeIntentInput struct {
	Intent
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

func (i CreateChargeIntentInput) Validate() error {
	var errs []error

	if err := i.Intent.Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.Name == "" {
		errs = append(errs, errors.New("name is required"))
	}

	return errors.Join(errs...)
}

type CreateChargeInput struct {
	Customer customer.CustomerID `json:"customer"`
	Currency currencyx.Code      `json:"currency"`

	Intents []CreateChargeIntentInput `json:"intents"`
}

func (i CreateChargeInput) Validate() error {
	var errs []error

	if err := i.Customer.Validate(); err != nil {
		errs = append(errs, err)
	}

	if err := i.Currency.Validate(); err != nil {
		errs = append(errs, err)
	}

	for idx, intent := range i.Intents {
		if err := intent.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("intent[%d]: %w", idx, err))
		}
	}

	return errors.Join(errs...)
}

func (i CreateChargeInput) WithCustomerAndCurrency() CreateChargeInput {
	for idx := range i.Intents {
		i.Intents[idx].Intent.CustomerID = i.Customer.ID
		i.Intents[idx].Intent.Currency = i.Currency
	}

	return i
}
