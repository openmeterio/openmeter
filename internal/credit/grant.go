package credit

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/internal/credit/grant"
	eventmodels "github.com/openmeterio/openmeter/internal/event/models"
	"github.com/openmeterio/openmeter/internal/event/spec"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/recurrence"
)

type GrantConnector interface {
	CreateGrant(ctx context.Context, owner grant.NamespacedGrantOwner, grant CreateGrantInput) (*grant.Grant, error)
	VoidGrant(ctx context.Context, grantID models.NamespacedID) error
}

var _ GrantConnector = &connector{}

type CreateGrantInput struct {
	Amount           float64
	Priority         uint8
	EffectiveAt      time.Time
	Expiration       grant.ExpirationPeriod
	Metadata         map[string]string
	ResetMaxRollover float64
	ResetMinRollover float64
	Recurrence       *recurrence.Recurrence
}

func (m *connector) CreateGrant(ctx context.Context, owner grant.NamespacedGrantOwner, input CreateGrantInput) (*grant.Grant, error) {
	doInTx := func(ctx context.Context, tx *entutils.TxDriver) (*grant.Grant, error) {
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
		g, err := m.grantRepo.WithTx(ctx, tx).CreateGrant(ctx, grant.GrantRepoCreateGrantInput{
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
		err = m.balanceSnapshotRepo.WithTx(ctx, tx).InvalidateAfter(ctx, owner, g.EffectiveAt)
		if err != nil {
			return nil, fmt.Errorf("failed to invalidate snapshots after %s: %w", g.EffectiveAt, err)
		}

		// publish event
		subjectKey, err := m.ownerConnector.GetOwnerSubjectKey(ctx, owner)
		if err != nil {
			return nil, err
		}

		event, err := spec.NewCloudEvent(
			spec.EventSpec{
				Source:  spec.ComposeResourcePath(owner.Namespace, spec.EntityEntitlement, string(owner.ID), spec.EntityGrant, g.ID),
				Subject: spec.ComposeResourcePath(owner.Namespace, spec.EntitySubjectKey, subjectKey),
			},
			grant.GrantCreatedEvent{
				Grant:     *g,
				Namespace: eventmodels.NamespaceID{ID: owner.Namespace},
				Subject:   eventmodels.SubjectKeyAndID{Key: subjectKey},
			},
		)
		if err != nil {
			return nil, err
		}

		if err := m.publisher.Publish(event); err != nil {
			return nil, err
		}

		return g, err
	}

	if ctxTx, err := entutils.GetTxDriver(ctx); err == nil {
		// we're already in a tx
		return doInTx(ctx, ctxTx)
	} else {
		return entutils.StartAndRunTx(ctx, m.grantRepo, doInTx)
	}
}

func (m *connector) VoidGrant(ctx context.Context, grantID models.NamespacedID) error {
	// can we void grants that have been used?
	g, err := m.grantRepo.GetGrant(ctx, grantID)
	if err != nil {
		return err
	}

	if g.VoidedAt != nil {
		return &models.GenericUserError{Message: "grant already voided"}
	}

	owner := grant.NamespacedGrantOwner{Namespace: grantID.Namespace, ID: g.OwnerID}

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

		// invalidate snapshots
		err = m.balanceSnapshotRepo.WithTx(ctx, tx).InvalidateAfter(ctx, owner, now)
		if err != nil {
			return nil, fmt.Errorf("failed to invalidate snapshots after %s: %w", g.EffectiveAt, err)
		}

		// publish an event
		subjectKey, err := m.ownerConnector.GetOwnerSubjectKey(ctx, owner)
		if err != nil {
			return nil, err
		}

		event, err := spec.NewCloudEvent(
			spec.EventSpec{
				Source:  spec.ComposeResourcePath(grantID.Namespace, spec.EntityEntitlement, string(owner.ID), spec.EntityGrant, grantID.ID),
				Subject: spec.ComposeResourcePath(grantID.Namespace, spec.EntitySubjectKey, subjectKey),
			},
			grant.GrantVoidedEvent{
				Grant:     g,
				Namespace: eventmodels.NamespaceID{ID: owner.Namespace},
				Subject:   eventmodels.SubjectKeyAndID{Key: subjectKey},
			},
		)
		if err != nil {
			return nil, err
		}

		return nil, m.publisher.Publish(event)
	})
	return err
}

func (m *connector) ListGrants(ctx context.Context, params grant.ListGrantsParams) (pagination.PagedResponse[grant.Grant], error) {
	if !params.Page.IsZero() {
		if err := params.Page.Validate(); err != nil {
			return pagination.PagedResponse[grant.Grant]{}, err
		}
	}
	return m.grantRepo.ListGrants(ctx, params)
}

func (m *connector) ListActiveGrantsBetween(ctx context.Context, owner grant.NamespacedGrantOwner, from, to time.Time) ([]grant.Grant, error) {
	return m.grantRepo.ListActiveGrantsBetween(ctx, owner, from, to)
}

func (m *connector) GetGrant(ctx context.Context, grantID models.NamespacedID) (grant.Grant, error) {
	return m.grantRepo.GetGrant(ctx, grantID)
}

type GrantNotFoundError struct {
	GrantID string
}

func (e *GrantNotFoundError) Error() string {
	return fmt.Sprintf("grant not found: %s", e.GrantID)
}
