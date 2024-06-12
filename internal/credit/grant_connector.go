package credit

import "context"

// Generic grant connector interface for accessing grants from persistence or network.
type GrantConnector interface {
	CreateGrant(ctx context.Context, grant Grant) (Grant, error)
	VoidGrant(ctx context.Context, grantID NamespacedGrantID) error
	ListGrants(ctx context.Context, params ListGrantsParams) ([]Grant, error)
	GetGrant(ctx context.Context, grantID NamespacedGrantID) (Grant, error)
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
