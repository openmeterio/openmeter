package governance

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/governance"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	QueryGovernanceAccess() QueryGovernanceAccessHandler
}

type handler struct {
	resolveNamespace  func(ctx context.Context) (string, error)
	governanceService governance.Service
	options           []httptransport.HandlerOption
}

func New(
	resolveNamespace func(ctx context.Context) (string, error),
	governanceService governance.Service,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		resolveNamespace:  resolveNamespace,
		governanceService: governanceService,
		options:           options,
	}
}
