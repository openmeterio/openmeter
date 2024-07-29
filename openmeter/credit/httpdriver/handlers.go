package httpdriver

import (
	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/credit/grant"
	"github.com/openmeterio/openmeter/internal/credit/httpdriver"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	GrantHandler      = httpdriver.GrantHandler
	ListGrantsHandler = httpdriver.ListGrantsHandler
	VoidGrantHandler  = httpdriver.VoidGrantHandler
)

func NewGrantHandler(
	namespaceDecoder namespacedriver.NamespaceDecoder,
	grantConnector credit.GrantConnector,
	grantRepo grant.GrantRepo,
	options ...httptransport.HandlerOption,
) GrantHandler {
	return httpdriver.NewGrantHandler(namespaceDecoder, grantConnector, grantRepo, options...)
}
