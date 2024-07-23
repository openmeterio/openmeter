package notification

import "github.com/openmeterio/openmeter/internal/notification"

type (
	ConnectorConfig = notification.ConnectorConfig
	Connector       = notification.Connector
)

func NewConnector(config ConnectorConfig) (Connector, error) {
	return notification.NewConnector(config)
}
