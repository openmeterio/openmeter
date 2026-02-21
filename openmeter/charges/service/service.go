package service

import (
	"errors"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/charges"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

type service struct {
	adapter            charges.Adapter
	billingService     billing.Service
	handler            charges.Handler
	featureService     feature.FeatureConnector
	customerService    customer.Service
	streamingConnector streaming.Connector
}

type Config struct {
	Adapter            charges.Adapter
	BillingService     billing.Service
	Handler            charges.Handler
	FeatureService     feature.FeatureConnector
	CustomerService    customer.Service
	StreamingConnector streaming.Connector
}

func (c Config) Validate() error {
	if c.Adapter == nil {
		return errors.New("adapter cannot be null")
	}

	if c.BillingService == nil {
		return errors.New("billing service cannot be null")
	}

	if c.FeatureService == nil {
		return errors.New("feature service cannot be null")
	}

	if c.CustomerService == nil {
		return errors.New("customer service cannot be null")
	}

	if c.StreamingConnector == nil {
		return errors.New("streaming connector cannot be null")
	}

	return nil
}

func New(config Config) (*service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	svc := &service{
		adapter:            config.Adapter,
		billingService:     config.BillingService,
		handler:            config.Handler,
		featureService:     config.FeatureService,
		customerService:    config.CustomerService,
		streamingConnector: config.StreamingConnector,
	}

	standardInvoiceEventHandler := &standardInvoiceEventHandler{
		chargesService: svc,
	}

	config.BillingService.RegisterStandardInvoiceHooks(standardInvoiceEventHandler)

	return svc, nil
}

var _ charges.Service = (*service)(nil)
