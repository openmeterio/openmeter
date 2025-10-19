package service

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/cost"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

type service struct {
	Config
}

type Config struct {
	BillingService     billing.Service
	CustomerService    customer.Service
	FeatureService     feature.FeatureConnector
	MeterService       meter.Service
	StreamingConnector streaming.Connector
}

func (c Config) Validate() error {
	if c.BillingService == nil {
		return fmt.Errorf("billing service is required")
	}

	if c.CustomerService == nil {
		return fmt.Errorf("customer service is required")
	}

	if c.FeatureService == nil {
		return fmt.Errorf("feature service is required")
	}

	if c.MeterService == nil {
		return fmt.Errorf("meter repo is required")
	}

	if c.StreamingConnector == nil {
		return fmt.Errorf("streaming connector is required")
	}

	return nil
}

func New(in Config) (cost.Service, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	return &service{
		Config: in,
	}, nil
}
