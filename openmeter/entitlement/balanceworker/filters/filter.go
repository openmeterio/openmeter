package filters

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
)

type Filter interface {
	IsNamespaceInScope(ctx context.Context, namespace string) (bool, error)
	IsEntitlementInScope(ctx context.Context, entitlement entitlement.Entitlement) (bool, error)
}

type NamedFilter interface {
	Filter

	Name() string
}
