package grants

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

const testNamespace = "ns"

type fakeService struct {
	listGrants func(context.Context, entitlement.ListNamespaceGrantsInput) (pagination.Result[grant.Grant], error)
	voidGrant  func(context.Context, entitlement.VoidGrantInput) error
}

func (f fakeService) ListNamespaceGrants(ctx context.Context, input entitlement.ListNamespaceGrantsInput) (pagination.Result[grant.Grant], error) {
	return f.listGrants(ctx, input)
}

func (f fakeService) VoidGrant(ctx context.Context, input entitlement.VoidGrantInput) error {
	return f.voidGrant(ctx, input)
}

func newTestHandler(svc fakeService) Handler {
	return New(func(context.Context) (string, error) { return testNamespace, nil }, svc)
}
