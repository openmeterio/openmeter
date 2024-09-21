// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package credit

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	eventmodels "github.com/openmeterio/openmeter/openmeter/event/models"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/recurrence"
)

type GrantConnector interface {
	CreateGrant(ctx context.Context, owner grant.NamespacedOwner, grant CreateGrantInput) (*grant.Grant, error)
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

func (m *connector) CreateGrant(ctx context.Context, owner grant.NamespacedOwner, input CreateGrantInput) (*grant.Grant, error) {
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
		g, err := m.grantRepo.WithTx(ctx, tx).CreateGrant(ctx, grant.RepoCreateInput{
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

		event := grant.CreatedEvent{
			Grant:     *g,
			Namespace: eventmodels.NamespaceID{ID: owner.Namespace},
			Subject:   eventmodels.SubjectKeyAndID{Key: subjectKey},
		}

		if err := m.publisher.Publish(ctx, event); err != nil {
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

	owner := grant.NamespacedOwner{Namespace: grantID.Namespace, ID: g.OwnerID}

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

		return nil, m.publisher.Publish(ctx, grant.VoidedEvent{
			Grant:     g,
			Namespace: eventmodels.NamespaceID{ID: owner.Namespace},
			Subject:   eventmodels.SubjectKeyAndID{Key: subjectKey},
		})
	})
	return err
}

type GrantNotFoundError struct {
	GrantID string
}

func (e *GrantNotFoundError) Error() string {
	return fmt.Sprintf("grant not found: %s", e.GrantID)
}
