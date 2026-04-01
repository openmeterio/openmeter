package resolvers

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/namespace"
)

type businessAccountProvisioner interface {
	EnsureBusinessAccounts(ctx context.Context, namespace string) (ledger.BusinessAccounts, error)
}

type namespaceHandler struct {
	provisioner businessAccountProvisioner
}

var _ namespace.Handler = (*namespaceHandler)(nil)

func NewNamespaceHandler(provisioner businessAccountProvisioner) namespace.Handler {
	return &namespaceHandler{
		provisioner: provisioner,
	}
}

func (h *namespaceHandler) CreateNamespace(ctx context.Context, name string) error {
	_, err := h.provisioner.EnsureBusinessAccounts(ctx, name)
	return err
}

func (h *namespaceHandler) DeleteNamespace(ctx context.Context, _ string) error {
	return nil
}
