package meterexportservice

import (
	"errors"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/meterexport"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

type Config struct {
	// Configuration
	EventSourceGroup string

	// Dependencies
	StreamingConnector streaming.Connector
	MeterService       meter.Service
}

func (c Config) validate() error {
	var errs []error

	if c.StreamingConnector == nil {
		errs = append(errs, errors.New("streaming connector is required"))
	}

	if c.MeterService == nil {
		errs = append(errs, errors.New("meter service is required"))
	}

	return errors.Join(errs...)
}

type service struct {
	Config
}

func New(config Config) (*service, error) {
	if err := config.validate(); err != nil {
		return nil, err
	}

	return &service{
		Config: config,
	}, nil
}

var _ meterexport.Service = (*service)(nil)
