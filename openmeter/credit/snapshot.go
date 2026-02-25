package credit

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/credit/engine"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	credittrace "github.com/openmeterio/openmeter/openmeter/credit/trace"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/models"
)

type snapshotParams struct {
	// Meter information to determine aggregation type
	meter meter.Meter
	// All grants used at engine.Run
	grants []grant.Grant
	// Owner of the snapshot
	owner models.NamespacedID
	// Snapshot is saved if the segment is not after this time & the start of the current usage period (at time of snapshot)
	notAfter time.Time
}

// It is assumed that there are no snapshots persisted during the length of the history (as engine.Run starts with a snapshot that should be the last valid snapshot)
func (m *connector) snapshotEngineResult(ctx context.Context, snapParams snapshotParams, runRes engine.RunResult) error {
	ctx, span := m.Tracer.Start(ctx, "credit.snapshotEngineResult", credittrace.WithOwner(snapParams.owner))
	defer span.End()

	// Skip snapshotting for LATEST type entitlements as the values fluctuate and snapshots can't be used
	if snapParams.meter.Aggregation == meter.MeterAggregationLatest {
		m.Logger.Debug("skipping snapshot for LATEST aggregation type entitlement", "owner", snapParams.owner, "meter", snapParams.meter.Key)
		return nil
	}

	segs := runRes.History.Segments()

	// i >= 1 because:
	// The first segment starts with the last valid snapshot and we don't want to create another snapshot for that same time
	for i := len(segs) - 1; i >= 1; i-- {
		seg := segs[i]

		// We can save a segment if its not after the current period start (this way backfilling, granting, resetting, etc... will work for the current UsagePeriod)
		if !seg.From.After(snapParams.notAfter) {
			snap, err := runRes.History.GetSnapshotAtStartOfSegment(i)
			if err != nil {
				return fmt.Errorf("failed to get snapshot at start of segment: %w", err)
			}

			if _, err := m.saveSnapshot(ctx, snapParams, snap); err != nil {
				return fmt.Errorf("failed to save snapshot: %w", err)
			}

			break
		}
	}

	return nil
}

func (m *connector) saveSnapshot(ctx context.Context, params snapshotParams, snap balance.Snapshot) (balance.Snapshot, error) {
	ctx, span := m.Tracer.Start(ctx, "credit.saveSnapshot", credittrace.WithOwner(params.owner))
	defer span.End()

	// Let's validate the timestamp
	if !snap.At.Truncate(m.Granularity).Equal(snap.At) {
		return snap, fmt.Errorf("snapshot timestamp %s is not aligned to granularity %s", snap.At, m.Granularity)
	}

	if err := m.removeInactiveGrantsFromSnapshotAt(&snap, params.grants, snap.At); err != nil {
		return snap, fmt.Errorf("failed to remove inactive grants from snapshot: %w", err)
	}

	if err := m.BalanceSnapshotService.Save(ctx, params.owner, []balance.Snapshot{snap}); err != nil {
		return snap, fmt.Errorf("failed to save snapshot: %w", err)
	}

	m.Logger.DebugContext(ctx, "saved snapshot", "snapshot", snap, "owner", params.owner)

	return snap, nil
}

// Fills in the snapshot's GrantBalanceMap with the provided grants so the Engine can use them.
func (m *connector) populateBalanceSnapshotWithMissingGrantsActiveAt(snapshot *balance.Snapshot, grants []grant.Grant, at time.Time) {
	for _, grant := range grants {
		if _, ok := snapshot.Balances[grant.ID]; !ok {
			if grant.ActiveAt(at) {
				snapshot.Balances.Set(grant.ID, grant.Amount)
			} else {
				snapshot.Balances.Set(grant.ID, 0.0)
			}
		}
	}
}

// Removes grants that are not active at the given time from the snapshot.
func (m *connector) removeInactiveGrantsFromSnapshotAt(snapshot *balance.Snapshot, grants []grant.Grant, at time.Time) error {
	grantMap := make(map[string]grant.Grant)
	for _, grant := range grants {
		grantMap[grant.ID] = grant
	}

	filtered := balance.Map{}
	for grantID, grantBalance := range snapshot.Balances {
		grant, ok := grantMap[grantID]
		if !ok {
			return fmt.Errorf("grant %s not found when removing inactive grants", grantID)
		}

		if grant.ActiveAt(at) {
			filtered.Set(grantID, grantBalance)
		}
	}

	snapshot.Balances = filtered

	return nil
}
