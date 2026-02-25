package credit

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	credittrace "github.com/openmeterio/openmeter/openmeter/credit/trace"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type GrantConnector interface {
	CreateGrant(ctx context.Context, owner models.NamespacedID, grant CreateGrantInput) (*grant.Grant, error)
	VoidGrant(ctx context.Context, grantID models.NamespacedID) error
}

var _ GrantConnector = &connector{}

type CreateGrantInput struct {
	Amount           float64
	Priority         uint8
	EffectiveAt      time.Time
	Expiration       *grant.ExpirationPeriod
	Metadata         map[string]string
	Annotations      models.Annotations
	ResetMaxRollover float64
	ResetMinRollover float64
	Recurrence       *timeutil.Recurrence
}

func (i CreateGrantInput) Validate() error {
	if i.Amount <= 0 {
		return ErrGrantAmountMustBePositive.WithAttr("amount", i.Amount)
	}
	if i.EffectiveAt.IsZero() {
		return ErrGrantEffectiveAtMustBeSet.WithAttr("effective_at", i.EffectiveAt)
	}
	return nil
}

func (m *connector) CreateGrant(ctx context.Context, ownerID models.NamespacedID, input CreateGrantInput) (*grant.Grant, error) {
	ctx, span := m.Tracer.Start(ctx, "credit.CreateGrant", credittrace.WithOwner(ownerID))
	defer span.End()

	if err := input.Validate(); err != nil {
		return nil, err
	}

	return transaction.Run(ctx, m.GrantRepo, func(ctx context.Context) (*grant.Grant, error) {
		tx, err := entutils.GetDriverFromContext(ctx)
		if err != nil {
			return nil, err
		}

		// All metering information is stored in windowSize chunks,
		// so we cannot do accurate calculations unless we follow that same windowing.
		owner, err := m.OwnerConnector.DescribeOwner(ctx, ownerID)
		if err != nil {
			return nil, fmt.Errorf("failed to describe owner: %w", err)
		}
		granularity := time.Minute
		input.EffectiveAt = input.EffectiveAt.Truncate(granularity)
		if input.Recurrence != nil {
			input.Recurrence.Anchor = input.Recurrence.Anchor.Truncate(granularity)
		}
		periodStart, err := m.OwnerConnector.GetUsagePeriodStartAt(ctx, ownerID, clock.Now())
		if err != nil {
			return nil, err
		}

		if input.EffectiveAt.Before(periodStart) {
			return nil, models.NewGenericValidationError(fmt.Errorf("grant effective date %s is before the current usage period %s", input.EffectiveAt, periodStart))
		}

		err = m.OwnerConnector.LockOwnerForTx(ctx, ownerID)
		if err != nil {
			return nil, err
		}
		repoInp := grant.RepoCreateInput{
			OwnerID:          ownerID.ID,
			Namespace:        ownerID.Namespace,
			Amount:           input.Amount,
			Priority:         input.Priority,
			EffectiveAt:      input.EffectiveAt,
			Expiration:       input.Expiration,
			Metadata:         input.Metadata,
			Annotations:      input.Annotations,
			ResetMaxRollover: input.ResetMaxRollover,
			ResetMinRollover: input.ResetMinRollover,
			Recurrence:       input.Recurrence,
		}

		if input.Expiration != nil {
			repoInp.ExpiresAt = lo.ToPtr(input.Expiration.GetExpiration(input.EffectiveAt))
			repoInp.Expiration = input.Expiration
		}

		g, err := m.GrantRepo.WithTx(ctx, tx).CreateGrant(ctx, repoInp)
		if err != nil {
			return nil, err
		}

		// invalidate snapshots
		err = m.BalanceSnapshotService.InvalidateAfter(ctx, ownerID, g.EffectiveAt)
		if err != nil {
			return nil, fmt.Errorf("failed to invalidate snapshots after %s: %w", g.EffectiveAt, err)
		}

		event := grant.NewCreatedEventV2FromGrant(*g, owner.StreamingCustomer)

		if err := m.Publisher.Publish(ctx, event); err != nil {
			return nil, err
		}

		return g, err
	})
}

func (m *connector) VoidGrant(ctx context.Context, grantID models.NamespacedID) error {
	ctx, span := m.Tracer.Start(ctx, "credit.VoidGrant", credittrace.WithOwner(grantID))
	defer span.End()

	// can we void grants that have been used?
	g, err := m.GrantRepo.GetGrant(ctx, grantID)
	if err != nil {
		return err
	}

	if g.VoidedAt != nil {
		return models.NewGenericValidationError(fmt.Errorf("grant already voided"))
	}

	ownerID := models.NamespacedID{Namespace: grantID.Namespace, ID: g.OwnerID}

	_, err = transaction.Run(ctx, m.GrantRepo, func(ctx context.Context) (*interface{}, error) {
		tx, err := entutils.GetDriverFromContext(ctx)
		if err != nil {
			return nil, err
		}

		owner, err := m.OwnerConnector.DescribeOwner(ctx, ownerID)
		if err != nil {
			return nil, err
		}

		err = m.OwnerConnector.LockOwnerForTx(ctx, ownerID)
		if err != nil {
			return nil, err
		}

		now := clock.Now().Truncate(m.Granularity)
		err = m.GrantRepo.WithTx(ctx, tx).VoidGrant(ctx, grantID, now)
		if err != nil {
			return nil, err
		}

		// invalidate snapshots
		err = m.BalanceSnapshotService.InvalidateAfter(ctx, ownerID, now)
		if err != nil {
			return nil, fmt.Errorf("failed to invalidate snapshots after %s: %w", g.EffectiveAt, err)
		}

		return nil, m.Publisher.Publish(ctx, grant.NewVoidedEventV2FromGrant(g, owner.StreamingCustomer, now))
	})
	return err
}

type GrantNotFoundError struct {
	GrantID string
}

func (e *GrantNotFoundError) Error() string {
	return fmt.Sprintf("grant not found: %s", e.GrantID)
}
