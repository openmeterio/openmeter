package grant

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/sortx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type OrderBy string

const (
	OrderByCreatedAt   OrderBy = "created_at"
	OrderByUpdatedAt   OrderBy = "updated_at"
	OrderByExpiresAt   OrderBy = "expires_at"
	OrderByEffectiveAt OrderBy = "effective_at"
	OrderByOwner       OrderBy = "owner_id" // check
	OrderByDefault     OrderBy = OrderByCreatedAt
)

func (f OrderBy) Values() []OrderBy {
	return []OrderBy{
		OrderByCreatedAt,
		OrderByUpdatedAt,
		OrderByExpiresAt,
		OrderByEffectiveAt,
		OrderByOwner,
	}
}

func (f OrderBy) StrValues() []string {
	return slicesx.Map(f.Values(), func(v OrderBy) string {
		return string(v)
	})
}

type ListParams struct {
	Namespace        string
	OwnerID          *string
	IncludeDeleted   bool
	CustomerIDs      []string
	SubjectKeys      []string
	FeatureIdsOrKeys []string
	Page             pagination.Page
	OrderBy          OrderBy
	Order            sortx.Order
	// will be deprecated
	Limit int
	// will be deprecated
	Offset int
}

type RepoCreateInput struct {
	OwnerID          string
	Namespace        string
	Amount           float64
	Priority         uint8
	EffectiveAt      time.Time
	Expiration       *ExpirationPeriod
	ExpiresAt        *time.Time
	Metadata         map[string]string
	Annotations      models.Annotations
	ResetMaxRollover float64
	ResetMinRollover float64
	Recurrence       *timeutil.Recurrence
}

type Repo interface {
	CreateGrant(ctx context.Context, grant RepoCreateInput) (*Grant, error)
	VoidGrant(ctx context.Context, grantID models.NamespacedID, at time.Time) error
	// For bw compatibility, if pagination is not provided we return a simple array
	ListGrants(ctx context.Context, params ListParams) (pagination.Result[Grant], error)
	// ListActiveGrantsBetween returns all grants that are active at any point between the given time range.
	ListActiveGrantsBetween(ctx context.Context, owner models.NamespacedID, from, to time.Time) ([]Grant, error)
	GetGrant(ctx context.Context, grantID models.NamespacedID) (Grant, error)

	// Sets deleted_at timestamp
	DeleteOwnerGrants(ctx context.Context, ownerID models.NamespacedID) error

	entutils.TxCreator
	entutils.TxUser[Repo]
}
