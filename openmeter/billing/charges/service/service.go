package service

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/service/flatfee"
)

type service struct {
	adapter        charges.Adapter
	billingService billing.Service
	handlers       Handlers
	flatFeeService charges.FlatFeeService
}

type Handlers struct {
	FlatFee charges.FlatFeeHandler
}

func (h Handlers) Validate() error {
	var errs []error

	if h.FlatFee == nil {
		errs = append(errs, errors.New("flat fee handler cannot be null"))
	}

	return errors.Join(errs...)
}

type Config struct {
	Adapter        charges.Adapter
	BillingService billing.Service
	Handlers       Handlers
}

func (c Config) Validate() error {
	var errs []error

	if c.Adapter == nil {
		errs = append(errs, errors.New("adapter cannot be null"))
	}

	if c.BillingService == nil {
		errs = append(errs, errors.New("billing service cannot be null"))
	}

	if err := c.Handlers.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("handlers: %w", err))
	}

	return errors.Join(errs...)
}

func New(config Config) (*service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	flatFeeService, err := flatfee.New(flatfee.Config{
		Adapter:        config.Adapter,
		FlatFeeHandler: config.Handlers.FlatFee,
	})
	if err != nil {
		return nil, err
	}

	svc := &service{
		adapter:        config.Adapter,
		billingService: config.BillingService,
		handlers:       config.Handlers,
		flatFeeService: flatFeeService,
	}

	standardInvoiceEventHandler := &standardInvoiceEventHandler{
		chargesService: svc,
	}

	config.BillingService.RegisterStandardInvoiceHooks(standardInvoiceEventHandler)

	return svc, nil
}

var _ charges.Service = (*service)(nil)
