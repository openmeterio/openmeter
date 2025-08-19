package adapter

import (
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/meterevent"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

func New(
	streamingConnector streaming.Connector,
	customerService customer.Service,
	meterService meter.Service,
) meterevent.Service {
	return &adapter{
		streamingConnector: streamingConnector,
		customerService:    customerService,
		meterService:       meterService,
	}
}

var _ meterevent.Service = (*adapter)(nil)

type adapter struct {
	streamingConnector streaming.Connector
	customerService    customer.Service
	meterService       meter.Service
}
