package clickhouse

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMeterCacheMarkerMaintenanceQueries(t *testing.T) {
	parse := func(s string) time.Time {
		ts, err := time.Parse(time.RFC3339, s)
		require.NoError(t, err)
		return ts
	}

	healBound := 20 * time.Minute
	backfilledAt := parse("2025-03-01T00:00:00Z")
	refreshStart := parse("2025-03-01T09:10:00Z")

	refreshed := meterCacheMarkerHealScope{
		BackfilledAt: backfilledAt,
		RefreshStart: lo.ToPtr(refreshStart),
	}

	// A view without an observed refresh (system.view_refreshes wiped by a ClickHouse
	// restart) can only heal through its backfill.
	unrefreshed := meterCacheMarkerHealScope{BackfilledAt: backfilledAt}

	t.Run("healed markers golden with mixed scopes", func(t *testing.T) {
		q := meterCacheHealedMarkersQuery{
			Database:  "openmeter",
			Namespace: "my_namespace",
			EventType: "event1",
			HealBound: healBound,
			Scopes:    []meterCacheMarkerHealScope{refreshed, unrefreshed},
		}

		// Deletion requires the marker healed by EVERY view of the (namespace, event
		// type): one still-unhealed view keeps the marker gating reads until that view
		// re-backfills.
		wantWhere := "namespace = ? AND event_type = ? " +
			"AND (created_at < ? OR (created_at > ? AND created_at < ?)) " +
			"AND created_at < ?"
		wantArgs := []interface{}{
			"my_namespace", "event1",
			backfilledAt, refreshStart.Add(-healBound), refreshStart,
			backfilledAt,
		}

		countSQL, countArgs := q.countSQL()
		assert.Equal(t, "SELECT count() FROM openmeter.om_meter_cache_invalidations WHERE "+wantWhere, countSQL)
		assert.Equal(t, wantArgs, countArgs)

		deleteSQL, deleteArgs := q.deleteSQL()
		assert.Equal(t, "DELETE FROM openmeter.om_meter_cache_invalidations WHERE "+wantWhere, deleteSQL)
		assert.Equal(t, wantArgs, deleteArgs)
	})

	t.Run("expired unhealed markers golden", func(t *testing.T) {
		sql, args := meterCacheExpiredUnhealedMarkersQuery{
			Database:  "openmeter",
			Namespace: "my_namespace",
			EventType: "event1",
			HealBound: healBound,
			Scope:     refreshed,
		}.toSQL()

		// The unhealed complement mirrors the reader's meterCacheMarkerOverlapQuery arms;
		// the trailing now64 cutoff keeps markers whose heal window is still open (a
		// coming refresh may yet heal them) from triggering a premature re-backfill.
		assert.Equal(t,
			"SELECT count() FROM openmeter.om_meter_cache_invalidations "+
				"WHERE namespace = ? AND event_type = ? "+
				"AND created_at >= ? AND (created_at >= ? OR created_at <= ?) "+
				"AND created_at <= now64(3) - INTERVAL 1200 SECOND",
			sql,
		)
		assert.Equal(t, []interface{}{
			"my_namespace", "event1",
			backfilledAt, refreshStart, refreshStart.Add(-healBound),
		}, args)
	})
}
