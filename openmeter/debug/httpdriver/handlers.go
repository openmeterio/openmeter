package httpdriver

import (
	"github.com/openmeterio/openmeter/internal/debug"
	"github.com/openmeterio/openmeter/internal/debug/httpdriver"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type DebugHandler = httpdriver.DebugHandler
type GetMetricsHandler = httpdriver.GetMetricsHandler

func NewDebugHandler(
	namespaceDecoder namespacedriver.NamespaceDecoder,
	connector debug.DebugConnector,
	options ...httptransport.HandlerOption,
) DebugHandler {
	return httpdriver.NewDebugHandler(namespaceDecoder, connector, options...)
}
