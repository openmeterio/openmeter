package service

import (
	"errors"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
)

type service struct {
	adapter        charges.Adapter
	billingService billing.Service
}

type Config struct {
	Adapter        charges.Adapter
	BillingService billing.Service
}

func (c Config) Validate() error {
	var errs []error

	if c.Adapter == nil {
		errs = append(errs, errors.New("adapter cannot be null"))
	}

	if c.BillingService == nil {
		errs = append(errs, errors.New("billing service cannot be null"))
	}

	return errors.Join(errs...)
}

func New(config Config) (*service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	svc := &service{
		adapter:        config.Adapter,
		billingService: config.BillingService,
	}

	standardInvoiceEventHandler := &standardInvoiceEventHandler{
		chargesService: svc,
	}

	config.BillingService.RegisterStandardInvoiceHooks(standardInvoiceEventHandler)

	return svc, nil
}

var _ charges.Service = (*service)(nil)
