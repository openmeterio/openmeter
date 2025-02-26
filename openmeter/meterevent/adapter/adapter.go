package adapter

import (
	"github.com/openmeterio/openmeter/openmeter/meterevent"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

func New(
	streamingConnector streaming.Connector,
) meterevent.Service {
	return &adapter{
		streamingConnector: streamingConnector,
	}
}

var _ meterevent.Service = (*adapter)(nil)

type adapter struct {
	streamingConnector streaming.Connector
}
