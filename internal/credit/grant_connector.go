package credit

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
)

type CreateGrantInput struct {
	Amount           float64
	Priority         uint8
	EffectiveAt      time.Time
	Expiration       ExpirationPeriod
	Metadata         map[string]string
	ResetMaxRollover float64
	Recurrence       *Recurrence
}

type GrantConnector interface {
	CreateGrant(ctx context.Context, owner NamespacedGrantOwner, grant CreateGrantInput) (Grant, error)
	VoidGrant(ctx context.Context, grantID NamespacedGrantID) error
	ListGrants(ctx context.Context, params ListGrantsParams) ([]Grant, error)
	ListActiveGrantsBetween(ctx context.Context, owner NamespacedGrantOwner, from, to time.Time) ([]Grant, error)
	GetGrant(ctx context.Context, grantID NamespacedGrantID) (Grant, error)
}

type GrantOrderBy string

const (
	GrantOrderByCreatedAt   GrantOrderBy = "created_at"
	GrantOrderByUpdatedAt   GrantOrderBy = "updated_at"
	GrantOrderByExpiresAt   GrantOrderBy = "expires_at"
	GrantOrderByEffectiveAt GrantOrderBy = "effective_at"
	GrantOrderByOwner       GrantOrderBy = "owner_id" // check
)

type ListGrantsParams struct {
	Namespace      string
	OwnerID        *GrantOwner
	IncludeDeleted bool
	Offset         int
	Limit          int
	OrderBy        GrantOrderBy
}

type DBCreateGrantInput struct {
	OwnerID          GrantOwner
	Namespace        string
	Amount           float64
	Priority         uint8
	EffectiveAt      time.Time
	Expiration       ExpirationPeriod
	ExpiresAt        time.Time
	Metadata         map[string]string
	ResetMaxRollover float64
	Recurrence       *Recurrence
}

// FIXME: separating these interfaces (connector & dbconnector) as is doesnt really make sense for grants
// Might be that credit operations in general are more tighlty linked than assumed here
type GrantDBConnector interface {
	CreateGrant(ctx context.Context, grant DBCreateGrantInput) (Grant, error)
	VoidGrant(ctx context.Context, grantID NamespacedGrantID) error
	ListGrants(ctx context.Context, params ListGrantsParams) ([]Grant, error)
	// ListActiveGrantsBetween returns all grants that are active at any point between the given time range.
	ListActiveGrantsBetween(ctx context.Context, owner NamespacedGrantOwner, from, to time.Time) ([]Grant, error)
	GetGrant(ctx context.Context, grantID NamespacedGrantID) (Grant, error)
}

type grantConnector struct {
	oc          OwnerConnector
	db          GrantDBConnector
	granularity time.Duration
}

func NewGrantConnector(oc OwnerConnector, db GrantDBConnector, granularity time.Duration) GrantConnector {
	return &grantConnector{oc: oc, db: db, granularity: granularity}
}

func (m *grantConnector) CreateGrant(ctx context.Context, owner NamespacedGrantOwner, input CreateGrantInput) (Grant, error) {
	periodStart, err := m.oc.GetCurrentUsagePeriodStartAt(ctx, owner, time.Now())
	if err != nil {
		return Grant{}, err
	}

	if input.EffectiveAt.Before(periodStart) {
		return Grant{}, &models.GenericUserError{Message: "grant effective date is before the current usage period"}
	}

	// All metering information is stored in windowSize chunks,
	// so we cannot do accurate calculations unless we follow that same windowing.
	// We don't want grants to retroactively apply, so they always take effect at the start of the
	// next window.
	//
	// TODO: validate against meter granularity not global config windowsize
	if truncated := input.EffectiveAt.Truncate(m.granularity); !truncated.Equal(input.EffectiveAt) {
		input.EffectiveAt = truncated.Add(m.granularity)
	}

	return m.db.CreateGrant(ctx, DBCreateGrantInput{
		OwnerID:          owner.ID,
		Namespace:        owner.Namespace,
		Amount:           input.Amount,
		Priority:         input.Priority,
		EffectiveAt:      input.EffectiveAt,
		Expiration:       input.Expiration,
		ExpiresAt:        input.Expiration.GetExpiration(input.EffectiveAt),
		Metadata:         input.Metadata,
		ResetMaxRollover: input.ResetMaxRollover,
		Recurrence:       input.Recurrence,
	})
}

func (m *grantConnector) VoidGrant(ctx context.Context, grantID NamespacedGrantID) error {
	// can we void grants that have been used?
	_, err := m.db.GetGrant(ctx, grantID)
	if err != nil {
		return err
	}

	return m.db.VoidGrant(ctx, grantID)
}

func (m *grantConnector) ListGrants(ctx context.Context, params ListGrantsParams) ([]Grant, error) {
	return m.db.ListGrants(ctx, params)
}

func (m *grantConnector) ListActiveGrantsBetween(ctx context.Context, owner NamespacedGrantOwner, from, to time.Time) ([]Grant, error) {
	return m.db.ListActiveGrantsBetween(ctx, owner, from, to)
}

func (m *grantConnector) GetGrant(ctx context.Context, grantID NamespacedGrantID) (Grant, error) {
	return m.db.GetGrant(ctx, grantID)
}

type GrantNotFoundError struct {
	GrantID GrantID
}

func (e *GrantNotFoundError) Error() string {
	return fmt.Sprintf("grant not found: %s", e.GrantID)
}
