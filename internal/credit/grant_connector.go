package credit

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/recurrence"
)

type CreateGrantInput struct {
	Amount           float64
	Priority         uint8
	EffectiveAt      time.Time
	Expiration       ExpirationPeriod
	Metadata         map[string]string
	ResetMaxRollover float64
	ResetMinRollover float64
	Recurrence       *recurrence.Recurrence
}

type GrantConnector interface {
	CreateGrant(ctx context.Context, owner NamespacedGrantOwner, grant CreateGrantInput) (*Grant, error)
	VoidGrant(ctx context.Context, grantID models.NamespacedID) error
	ListGrants(ctx context.Context, params ListGrantsParams) ([]Grant, error)
	ListActiveGrantsBetween(ctx context.Context, owner NamespacedGrantOwner, from, to time.Time) ([]Grant, error)
	GetGrant(ctx context.Context, grantID models.NamespacedID) (Grant, error)
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
	ListGrants(ctx context.Context, params ListGrantsParams) ([]Grant, error)
	// ListActiveGrantsBetween returns all grants that are active at any point between the given time range.
	ListActiveGrantsBetween(ctx context.Context, owner NamespacedGrantOwner, from, to time.Time) ([]Grant, error)
	GetGrant(ctx context.Context, grantID models.NamespacedID) (Grant, error)

	entutils.TxCreator
	entutils.TxUser[GrantRepo]
}

type grantConnector struct {
	ownerConnector           OwnerConnector
	grantRepo                GrantRepo
	balanceSnapshotConnector BalanceSnapshotRepo
	granularity              time.Duration
}

func NewGrantConnector(
	ownerConnector OwnerConnector,
	grantRepo GrantRepo,
	balanceSnapshotConnector BalanceSnapshotRepo,
	granularity time.Duration,
) GrantConnector {
	return &grantConnector{
		ownerConnector:           ownerConnector,
		grantRepo:                grantRepo,
		balanceSnapshotConnector: balanceSnapshotConnector,
		granularity:              granularity,
	}
}

func (m *grantConnector) CreateGrant(ctx context.Context, owner NamespacedGrantOwner, input CreateGrantInput) (*Grant, error) {
	doInTx := func(ctx context.Context, tx *entutils.TxDriver) (*Grant, error) {
		// All metering information is stored in windowSize chunks,
		// so we cannot do accurate calculations unless we follow that same windowing.
		meter, err := m.ownerConnector.GetMeter(ctx, owner)
		if err != nil {
			return nil, err
		}
		granularity := meter.WindowSize.Duration()
		input.EffectiveAt = input.EffectiveAt.Truncate(granularity)
		if input.Recurrence != nil {
			input.Recurrence.Anchor = input.Recurrence.Anchor.Truncate(granularity)
		}
		periodStart, err := m.ownerConnector.GetUsagePeriodStartAt(ctx, owner, clock.Now())
		if err != nil {
			return nil, err
		}

		if input.EffectiveAt.Before(periodStart) {
			return nil, &models.GenericUserError{Message: "grant effective date is before the current usage period"}
		}

		err = m.ownerConnector.LockOwnerForTx(ctx, tx, owner)
		if err != nil {
			return nil, err
		}
		grant, err := m.grantRepo.WithTx(ctx, tx).CreateGrant(ctx, GrantRepoCreateGrantInput{
			OwnerID:          owner.ID,
			Namespace:        owner.Namespace,
			Amount:           input.Amount,
			Priority:         input.Priority,
			EffectiveAt:      input.EffectiveAt,
			Expiration:       input.Expiration,
			ExpiresAt:        input.Expiration.GetExpiration(input.EffectiveAt),
			Metadata:         input.Metadata,
			ResetMaxRollover: input.ResetMaxRollover,
			ResetMinRollover: input.ResetMinRollover,
			Recurrence:       input.Recurrence,
		})
		if err != nil {
			return nil, err
		}

		// invalidate snapshots
		err = m.balanceSnapshotConnector.WithTx(ctx, tx).InvalidateAfter(ctx, owner, grant.EffectiveAt)
		if err != nil {
			return nil, fmt.Errorf("failed to invalidate snapshots after %s: %w", grant.EffectiveAt, err)
		}

		return grant, err
	}

	if ctxTx, err := entutils.GetTxDriver(ctx); err == nil {
		// we're already in a tx
		return doInTx(ctx, ctxTx)
	} else {
		return entutils.StartAndRunTx(ctx, m.grantRepo, doInTx)
	}
}

func (m *grantConnector) VoidGrant(ctx context.Context, grantID models.NamespacedID) error {
	// can we void grants that have been used?
	grant, err := m.grantRepo.GetGrant(ctx, grantID)
	if err != nil {
		return err
	}

	if grant.VoidedAt != nil {
		return &models.GenericUserError{Message: "grant already voided"}
	}

	owner := NamespacedGrantOwner{Namespace: grantID.Namespace, ID: grant.OwnerID}

	_, err = entutils.StartAndRunTx(ctx, m.grantRepo, func(ctx context.Context, tx *entutils.TxDriver) (*interface{}, error) {
		err := m.ownerConnector.LockOwnerForTx(ctx, tx, owner)
		if err != nil {
			return nil, err
		}
		now := clock.Now().Truncate(m.granularity)
		err = m.grantRepo.WithTx(ctx, tx).VoidGrant(ctx, grantID, now)
		if err != nil {
			return nil, err
		}
		err = m.balanceSnapshotConnector.WithTx(ctx, tx).InvalidateAfter(ctx, owner, now)
		return nil, err
	})
	return err
}

func (m *grantConnector) ListGrants(ctx context.Context, params ListGrantsParams) ([]Grant, error) {
	return m.grantRepo.ListGrants(ctx, params)
}

func (m *grantConnector) ListActiveGrantsBetween(ctx context.Context, owner NamespacedGrantOwner, from, to time.Time) ([]Grant, error) {
	return m.grantRepo.ListActiveGrantsBetween(ctx, owner, from, to)
}

func (m *grantConnector) GetGrant(ctx context.Context, grantID models.NamespacedID) (Grant, error) {
	return m.grantRepo.GetGrant(ctx, grantID)
}

type GrantNotFoundError struct {
	GrantID string
}

func (e *GrantNotFoundError) Error() string {
	return fmt.Sprintf("grant not found: %s", e.GrantID)
}
