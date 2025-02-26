package common

import (
	"github.com/google/wire"
	"github.com/openmeterio/openmeter/openmeter/meterevent"
	"github.com/openmeterio/openmeter/openmeter/meterevent/adapter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

var MeterEvent = wire.NewSet(
	NewMeterEventService,
)

func NewMeterEventService(
	streamingConnector streaming.Connector,
) meterevent.Service {
	return adapter.New(streamingConnector)
}
