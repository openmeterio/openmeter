package debug

import (
	"github.com/openmeterio/openmeter/internal/debug"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

func NewDebugConnector(
	streaming streaming.Connector,
) DebugConnector {
	return debug.NewDebugConnector(streaming)
}
