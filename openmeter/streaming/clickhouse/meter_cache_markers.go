package clickhouse

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/samber/lo"
)

// meterCacheMarkerHealScope is one deployed view's heal evidence against invalidation
// markers of its (namespace, event type): the ClickHouse-clock instant its full-history
// backfill started, and the start of its latest observed successful refresh (nil while
// system.view_refreshes reports none, e.g. right after a ClickHouse restart).
type meterCacheMarkerHealScope struct {
	BackfilledAt time.Time
	RefreshStart *time.Time
}

// healedExpr renders the predicate under which a marker is healed for this view, with the
// same two arms the reader's meterCacheMarkerOverlapQuery uses: covered by the backfill
// (created_at < BackfilledAt) or covered by the latest refresh's stored_at lookback (G1:
// started after the marker, within the heal bound of it).
func (s meterCacheMarkerHealScope) healedExpr(healBound time.Duration) (string, []interface{}) {
	if s.RefreshStart == nil {
		return "created_at < ?", []interface{}{s.BackfilledAt}
	}

	return "(created_at < ? OR (created_at > ? AND created_at < ?))",
		[]interface{}{s.BackfilledAt, s.RefreshStart.Add(-healBound), *s.RefreshStart}
}

// unhealedExpr renders the complement of healedExpr. The refresh arm is present only when
// a refresh was observed; without one, only the backfill can heal.
func (s meterCacheMarkerHealScope) unhealedExpr(healBound time.Duration) (string, []interface{}) {
	if s.RefreshStart == nil {
		return "created_at >= ?", []interface{}{s.BackfilledAt}
	}

	return "created_at >= ? AND (created_at >= ? OR created_at <= ?)",
		[]interface{}{s.BackfilledAt, *s.RefreshStart, s.RefreshStart.Add(-healBound)}
}

// meterCacheHealedMarkersQuery selects or deletes the invalidation markers of one
// (namespace, event type) pair that every deployed stamped view of the pair has healed.
// Deleting such markers is what keeps marker healing from regressing: the reader's heal
// rule can only consult the latest refresh, so a marker healed within its window would
// flip back to unhealed once refreshes advance past created_at + heal bound, forcing the
// marked range live until the marker's TTL.
type meterCacheHealedMarkersQuery struct {
	Database  string
	Namespace string
	EventType string
	HealBound time.Duration
	Scopes    []meterCacheMarkerHealScope
}

func (q meterCacheHealedMarkersQuery) where() (string, []interface{}) {
	conds := []string{"namespace = ?", "event_type = ?"}
	args := []interface{}{q.Namespace, q.EventType}

	for _, scope := range q.Scopes {
		cond, condArgs := scope.healedExpr(q.HealBound)
		conds = append(conds, cond)
		args = append(args, condArgs...)
	}

	return strings.Join(conds, " AND "), args
}

func (q meterCacheHealedMarkersQuery) countSQL() (string, []interface{}) {
	where, args := q.where()

	return fmt.Sprintf("SELECT count() FROM %s WHERE %s", getTableName(q.Database, meterCacheInvalidationsTableName), where), args
}

func (q meterCacheHealedMarkersQuery) deleteSQL() (string, []interface{}) {
	where, args := q.where()

	return fmt.Sprintf("DELETE FROM %s WHERE %s", getTableName(q.Database, meterCacheInvalidationsTableName), where), args
}

// meterCacheExpiredUnhealedMarkersQuery counts one view's markers that are unhealed and
// whose heal window has already expired: no refresh starting from now on can satisfy
// refreshStart − created_at < healBound anymore, so the only way the marked buckets ever
// converge is a re-backfill. The expiry cutoff is evaluated on the ClickHouse clock
// (now64) against the server-stamped created_at, keeping app clock skew out of the
// decision.
type meterCacheExpiredUnhealedMarkersQuery struct {
	Database  string
	Namespace string
	EventType string
	HealBound time.Duration
	Scope     meterCacheMarkerHealScope
}

func (q meterCacheExpiredUnhealedMarkersQuery) toSQL() (string, []interface{}) {
	unhealed, args := q.Scope.unhealedExpr(q.HealBound)

	sql := fmt.Sprintf(
		"SELECT count() FROM %s WHERE namespace = ? AND event_type = ? AND %s AND created_at <= now64(3) - INTERVAL %d SECOND",
		getTableName(q.Database, meterCacheInvalidationsTableName),
		unhealed,
		int64(q.HealBound/time.Second),
	)

	return sql, append([]interface{}{q.Namespace, q.EventType}, args...)
}

// ReconcileMeterCacheMarkers is the reconciler's per-pass invalidation marker
// maintenance. It deletes markers that every deployed stamped view of their (namespace,
// event type) has healed, and returns the names of views holding expired-unhealed markers
// — markers no future refresh can heal — which only a re-backfill (viewActionEnsure) can
// converge.
//
// Views without a successful refresh are never reported for repair: their reads are
// already refused as stale, and once refreshes resume the next pass re-evaluates against
// real heal evidence. Repairs are self-throttling: a completed re-backfill advances the
// view's BackfilledAt past the offending markers' created_at, so they count as healed on
// the following pass and get deleted instead of re-reported.
//
// Marker groups without any deployed stamped view are left alone: no read consults the
// cache for them (the gate refuses on view_missing/backfill_unstamped), a future view's
// backfill starts after the markers and heals them, and the 7 day TTL bounds the table.
func (c *Connector) ReconcileMeterCacheMarkers(ctx context.Context, views []MeterCacheView) ([]string, error) {
	if !c.config.Cache.Enabled {
		return nil, errors.New("meter cache is disabled")
	}

	type markerGroup struct {
		namespace string
		eventType string
	}

	viewsByGroup := map[markerGroup][]MeterCacheView{}

	for _, view := range views {
		if !view.MetadataOK || view.BackfilledAt == nil {
			continue
		}

		key := markerGroup{namespace: view.Namespace, eventType: view.EventType}
		viewsByGroup[key] = append(viewsByGroup[key], view)
	}

	if len(viewsByGroup) == 0 {
		return nil, nil
	}

	rows, err := c.config.ClickHouse.Query(ctx,
		"SELECT DISTINCT namespace, event_type FROM "+getTableName(c.config.Database, meterCacheInvalidationsTableName))
	if err != nil {
		return nil, fmt.Errorf("list meter cache marker groups: %w", err)
	}

	defer rows.Close()

	var groups []markerGroup

	for rows.Next() {
		var group markerGroup

		if err := rows.Scan(&group.namespace, &group.eventType); err != nil {
			return nil, fmt.Errorf("scan meter cache marker group: %w", err)
		}

		groups = append(groups, group)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list meter cache marker groups: %w", err)
	}

	healBound := meterCacheHealBound(c.config.Cache.MinimumUsageAge, c.config.Cache.RefreshInterval)

	var (
		needRepair []string
		errs       []error
	)

	for _, group := range groups {
		matching := viewsByGroup[group]
		if len(matching) == 0 {
			continue
		}

		scopes := make([]meterCacheMarkerHealScope, 0, len(matching))

		for _, view := range matching {
			scope := meterCacheMarkerHealScope{BackfilledAt: *view.BackfilledAt}

			if view.LastSuccessTime != nil && view.LastSuccessDurationMS != nil {
				scope.RefreshStart = lo.ToPtr(view.LastSuccessTime.Add(-time.Duration(*view.LastSuccessDurationMS) * time.Millisecond))
			}

			scopes = append(scopes, scope)
		}

		healed := meterCacheHealedMarkersQuery{
			Database:  c.config.Database,
			Namespace: group.namespace,
			EventType: group.eventType,
			HealBound: healBound,
			Scopes:    scopes,
		}

		countSQL, countArgs := healed.countSQL()

		// The count probe guards the delete so the steady state (no healed markers, or no
		// markers at all once this pass deleted them) pays one indexed SELECT instead of a
		// delete mutation.
		var healedCount uint64
		if err := c.config.ClickHouse.QueryRow(ctx, countSQL, countArgs...).Scan(&healedCount); err != nil {
			errs = append(errs, fmt.Errorf("count healed markers for %s/%s: %w", group.namespace, group.eventType, err))

			continue
		}

		if healedCount > 0 {
			deleteSQL, deleteArgs := healed.deleteSQL()
			if err := c.config.ClickHouse.Exec(ctx, deleteSQL, deleteArgs...); err != nil {
				errs = append(errs, fmt.Errorf("delete healed markers for %s/%s: %w", group.namespace, group.eventType, err))
			}
		}

		for i, view := range matching {
			if scopes[i].RefreshStart == nil {
				continue
			}

			expiredSQL, expiredArgs := meterCacheExpiredUnhealedMarkersQuery{
				Database:  c.config.Database,
				Namespace: group.namespace,
				EventType: group.eventType,
				HealBound: healBound,
				Scope:     scopes[i],
			}.toSQL()

			var expiredCount uint64
			if err := c.config.ClickHouse.QueryRow(ctx, expiredSQL, expiredArgs...).Scan(&expiredCount); err != nil {
				errs = append(errs, fmt.Errorf("count expired unhealed markers for view %s: %w", view.Name, err))

				continue
			}

			if expiredCount > 0 {
				needRepair = append(needRepair, view.Name)
			}
		}
	}

	slices.Sort(needRepair)

	return slices.Compact(needRepair), errors.Join(errs...)
}
