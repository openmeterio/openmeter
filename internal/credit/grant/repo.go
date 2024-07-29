package grant

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/recurrence"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type GrantOrderBy string

const (
	GrantOrderByCreatedAt   GrantOrderBy = "created_at"
	GrantOrderByUpdatedAt   GrantOrderBy = "updated_at"
	GrantOrderByExpiresAt   GrantOrderBy = "expires_at"
	GrantOrderByEffectiveAt GrantOrderBy = "effective_at"
	GrantOrderByOwner       GrantOrderBy = "owner_id" // check
)

func (f GrantOrderBy) Values() []GrantOrderBy {
	return []GrantOrderBy{
		GrantOrderByCreatedAt,
		GrantOrderByUpdatedAt,
		GrantOrderByExpiresAt,
		GrantOrderByEffectiveAt,
		GrantOrderByOwner,
	}
}

func (f GrantOrderBy) StrValues() []string {
	return slicesx.Map(f.Values(), func(v GrantOrderBy) string {
		return string(v)
	})
}

type ListGrantsParams struct {
	Namespace        string
	OwnerID          *GrantOwner
	IncludeDeleted   bool
	SubjectKeys      []string
	FeatureIdsOrKeys []string
	Page             pagination.Page
	OrderBy          GrantOrderBy
	Order            sortx.Order
	// will be deprecated
	Limit int
	// will be deprecated
	Offset int
}

type GrantRepoCreateGrantInput struct {
	OwnerID          GrantOwner
	Namespace        string
	Amount           float64
	Priority         uint8
	EffectiveAt      time.Time
	Expiration       ExpirationPeriod
	ExpiresAt        time.Time
	Metadata         map[string]string
	ResetMaxRollover float64
	ResetMinRollover float64
	Recurrence       *recurrence.Recurrence
}

type GrantRepo interface {
	CreateGrant(ctx context.Context, grant GrantRepoCreateGrantInput) (*Grant, error)
	VoidGrant(ctx context.Context, grantID models.NamespacedID, at time.Time) error
	// For bw compatibility, if pagination is not provided we return a simple array
	ListGrants(ctx context.Context, params ListGrantsParams) (pagination.PagedResponse[Grant], error)
	// ListActiveGrantsBetween returns all grants that are active at any point between the given time range.
	ListActiveGrantsBetween(ctx context.Context, owner NamespacedGrantOwner, from, to time.Time) ([]Grant, error)
	GetGrant(ctx context.Context, grantID models.NamespacedID) (Grant, error)

	entutils.TxCreator
	entutils.TxUser[GrantRepo]
}
