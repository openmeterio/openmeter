package creditpurchase

import (
	"errors"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
)

type Config struct {
	Adapter               charges.Adapter
	CreditPurchaseHandler charges.CreditPurchaseHandler
}

func (c Config) Validate() error {
	var errs []error

	if c.Adapter == nil {
		errs = append(errs, errors.New("adapter cannot be null"))
	}

	if c.CreditPurchaseHandler == nil {
		errs = append(errs, errors.New("credit purchase handler cannot be null"))
	}

	return errors.Join(errs...)
}

func New(config Config) (charges.CreditPurchaseService, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &service{
		adapter:               config.Adapter,
		creditPurchaseHandler: config.CreditPurchaseHandler,
	}, nil
}

type service struct {
	adapter               charges.Adapter
	creditPurchaseHandler charges.CreditPurchaseHandler
}
