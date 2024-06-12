package credit

import (
	"context"
)

// Generic Connector to interface with Credit Capabilities
type Connector interface {
	// TODO: do we need management APIs separate from entitlements?
	// if so then credits in general can be persisted and managed separately.
	// if not then we can just use entitlements for everything.

	// Grant Management
	CreateGrant(ctx context.Context, grant Grant) (Grant, error)
	VoidGrant(ctx context.Context, grantID NamespacedGrantID) error // TODO: do we need this even, maybe call it DeleteGrant?
	ListGrants(ctx context.Context, params ListGrantsParams) ([]Grant, error)
	GetGrant(ctx context.Context, grantID NamespacedGrantID) (Grant, error)

	// // Balance & Usage
	// GetGrantUsageHistory(ctx context.Context, grantID GrantID, params BalanceHistoryParams) ([]EntitlementBalanceHistoryWindow, error)
}

type GrantOrderBy string

const (
	GrantOrderByCreatedAt   GrantOrderBy = "created_at"
	GrantOrderByUpdatedAt   GrantOrderBy = "updated_at"
	GrantOrderByExpiresAt   GrantOrderBy = "expires_at"
	GrantOrderByEffectiveAt GrantOrderBy = "effective_at"
)

type ListGrantsParams struct {
	Namespace      string
	OwnerID        GrantOwner
	IncludeDeleted bool
	Offset         int
	Limit          int
	OrderBy        GrantOrderBy
}
