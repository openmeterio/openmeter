package creditdriver

import (
	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/credit/creditdriver"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler = creditdriver.Handlers

func New(
	creditConnector credit.Connector,
	meterRepository meter.Repository,
	namespaceDecoder namespacedriver.NamespaceDecoder,
	options ...httptransport.HandlerOption,
) Handler {
	return creditdriver.New(
		creditConnector,
		meterRepository,
		namespaceDecoder,
		options...,
	)
}
