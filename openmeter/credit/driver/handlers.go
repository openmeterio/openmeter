package creditdriver

import (
	"github.com/openmeterio/openmeter/internal/credit"
	creditdriver "github.com/openmeterio/openmeter/internal/credit/driver"
	"github.com/openmeterio/openmeter/internal/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	GrantHandler      = creditdriver.GrantHandler
	ListGrantsHandler = creditdriver.ListGrantsHandler
	VoidGrantHandler  = creditdriver.VoidGrantHandler
)

func NewGrantHandler(
	namespaceDecoder namespacedriver.NamespaceDecoder,
	grantConnector credit.GrantConnector,
	grantRepo grant.Repo,
	options ...httptransport.HandlerOption,
) GrantHandler {
	return creditdriver.NewGrantHandler(namespaceDecoder, grantConnector, grantRepo, options...)
}
