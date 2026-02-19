package service

import (
	"errors"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/charges"
)

type service struct {
	adapter        charges.Adapter
	billingService billing.Service
	handler        charges.Handler
}

type Config struct {
	Adapter        charges.Adapter
	BillingService billing.Service
	Handler        charges.Handler
}

func (c Config) Validate() error {
	if c.Adapter == nil {
		return errors.New("adapter cannot be null")
	}

	if c.BillingService == nil {
		return errors.New("billing service cannot be null")
	}

	return nil
}

func New(config Config) (*service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	svc := &service{
		adapter:        config.Adapter,
		billingService: config.BillingService,
		handler:        config.Handler,
	}

	standardInvoiceEventHandler := &standardInvoiceEventHandler{
		chargesService: svc,
	}

	config.BillingService.RegisterStandardInvoiceHooks(standardInvoiceEventHandler)

	return svc, nil
}

var _ charges.Service = (*service)(nil)
