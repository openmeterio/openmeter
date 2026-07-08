package clickhouse

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/samber/lo"

	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
)

// ErrMeterCacheUnsupported marks capability-probe failures that no amount of retrying can
// fix on this ClickHouse deployment (no refreshable view support, missing SYSTEM REFRESH
// VIEW grant). The lifecycle reconciler distinguishes it from transient probe errors: an
// unsupported deployment disables reconciliation for the process lifetime, while transient
// failures are retried on the next tick.
var ErrMeterCacheUnsupported = errors.New("meter cache is unsupported by this ClickHouse deployment")

// MeterCacheView is one deployed cache MV as the lifecycle reconciler sees it: the object
// name discovered in system.tables, the parsed comment metadata, and the refresh health
// reported by system.view_refreshes.
type MeterCacheView struct {
	Name string

	// MetadataOK is false when the comment did not parse as valid cache MV metadata; the
	// reconciler must treat such a view as foreign or corrupt and recreate it rather than
	// trust any of the metadata fields below.
	MetadataOK bool
	// Namespace is the exact namespace recorded at creation; the view name only carries an
	// 8-hex fold of it, so name-colliding namespaces are told apart by this field alone.
	Namespace string
	MeterKey  string
	EventType string
	// MeterHash and DDLHash are the formatted (16 hex char) hashes recorded at creation.
	MeterHash string
	DDLHash   string
	// BackfilledAt is nil while the one-time backfill has not completed (G3): either it is
	// still running, or the actor performing it died in between. Readers refuse such views
	// and the reconciler re-runs backfill + stamp. Its value is the ClickHouse-clock
	// instant the backfill started, which doubles as a marker heal bound.
	BackfilledAt *time.Time
	// CoveredAt is the durable refresh-coverage watermark the reconciler advances while
	// refreshes stay continuous; nil until first advanced. See meterCacheMVMetadata.
	CoveredAt *time.Time

	// LastSuccessTime is nil until the view's first successful refresh since ClickHouse
	// startup (system.view_refreshes is per-server, in-memory state).
	LastSuccessTime *time.Time
	// LastSuccessDurationMS is the matching refresh duration; refreshStart (the instant
	// heal comparisons must anchor on) is LastSuccessTime − LastSuccessDurationMS.
	LastSuccessDurationMS *uint64
	Exception             string
}

// MeterCacheDesiredView is the cache view one meter definition maps to under the current
// cache configuration: the deterministic MV name plus the formatted hashes the deployed
// view's metadata must match to be considered converged.
type MeterCacheDesiredView struct {
	Name      string
	MeterHash string
	DDLHash   string
}

func (c *Connector) meterCacheMV(namespace string, m meterpkg.Meter) createMeterCacheMV {
	return createMeterCacheMV{
		Database:        c.config.Database,
		EventsTableName: c.config.EventsTableName,
		Namespace:       namespace,
		Meter:           m,
		Grain:           c.config.Cache.WindowSize,
		RefreshInterval: c.config.Cache.RefreshInterval,
		MinimumUsageAge: c.config.Cache.MinimumUsageAge,
	}
}

// DesiredMeterCacheView maps a meter definition to its desired cache view. A non-nil error
// means the meter cannot be cached at all under the current configuration (reserved
// group-by aliases, G9); callers must skip the meter and let it be served live, never
// treat the error as fatal for other meters.
func (c *Connector) DesiredMeterCacheView(namespace string, m meterpkg.Meter) (MeterCacheDesiredView, error) {
	if !c.config.Cache.Enabled {
		return MeterCacheDesiredView{}, errors.New("meter cache is disabled")
	}

	// LATEST is never cacheable (see meterCacheStaticReject): rejecting it here, before any
	// MV metadata is generated, keeps it out of the reconciler's desired set entirely, so a
	// LATEST meter never gets an MV created and any pre-existing LATEST MV from an earlier
	// deploy is dropped as undesired by the reconciler's diff.
	if m.Aggregation == meterpkg.MeterAggregationLatest {
		return MeterCacheDesiredView{}, errors.New("meter cache does not support the LATEST aggregation: always served live")
	}

	mv := c.meterCacheMV(namespace, m)

	metadata, err := mv.metadata()
	if err != nil {
		return MeterCacheDesiredView{}, err
	}

	return MeterCacheDesiredView{
		Name:      mv.name(),
		MeterHash: metadata.MeterHash,
		DDLHash:   metadata.DDLHash,
	}, nil
}

// ListActualViews returns every deployed meter cache MV in the database, joined with its
// refresh health. Views with unparseable comment metadata are returned with MetadataOK
// false instead of being filtered out: the reconciler owns the om_meter_cache_mv_ name
// prefix and must see foreign or corrupt views to recreate them.
func (c *Connector) ListActualViews(ctx context.Context) ([]MeterCacheView, error) {
	if !c.config.Cache.Enabled {
		return nil, errors.New("meter cache is disabled")
	}

	rows, err := c.config.ClickHouse.Query(ctx,
		"SELECT t.name, t.comment, r.last_success_time, r.last_success_duration_ms, r.exception "+
			"FROM system.tables AS t "+
			"LEFT JOIN system.view_refreshes AS r ON r.database = t.database AND r.view = t.name "+
			"WHERE t.database = ? AND t.engine = 'MaterializedView' AND startsWith(t.name, ?)",
		c.config.Database, meterCacheMVNamePrefix,
	)
	if err != nil {
		return nil, fmt.Errorf("list meter cache views: %w", err)
	}

	defer rows.Close()

	var views []MeterCacheView

	for rows.Next() {
		var (
			view    MeterCacheView
			comment string
		)

		if err := rows.Scan(&view.Name, &comment, &view.LastSuccessTime, &view.LastSuccessDurationMS, &view.Exception); err != nil {
			return nil, fmt.Errorf("scan meter cache view: %w", err)
		}

		if metadata, err := parseMeterCacheMVMetadata(comment); err == nil {
			view.MetadataOK = true
			view.Namespace = metadata.Namespace
			view.MeterKey = metadata.MeterKey
			view.EventType = metadata.EventType
			view.MeterHash = metadata.MeterHash
			view.DDLHash = metadata.DDLHash
			view.BackfilledAt = metadata.BackfilledAt
			view.CoveredAt = metadata.CoveredAt
		}

		views = append(views, view)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list meter cache views: %w", err)
	}

	return views, nil
}

// EnsureMeterCache converges one meter's cache view to fully deployed: CREATE the MV (IF
// NOT EXISTS, so repairing a half-deployed view and losing a create race are both benign),
// backfill full settled history in month chunks, then stamp backfilled_at. The stamp is
// deliberately last: readers refuse unstamped views (G3), so an actor dying anywhere in
// this sequence leaves a visibly unfinished view the next reconciliation pass re-runs this
// exact method on. Re-running over an already-deployed view is safe end to end because
// backfill rows resolve against refresh rows by newest-wins.
//
// The stamped value is the ClickHouse-clock instant read before the backfill starts
// scanning, not the app-clock completion time: marker healing compares it against marker
// created_at (also ClickHouse clock), and a marker written before the backfill started is
// provably covered by its full-history scan, while a marker written during the backfill
// may describe events a chunk already passed over and must stay unhealed by it.
func (c *Connector) EnsureMeterCache(ctx context.Context, namespace string, m meterpkg.Meter) error {
	if !c.config.Cache.Enabled {
		return errors.New("meter cache is disabled")
	}

	mv := c.meterCacheMV(namespace, m)

	createSQL, err := mv.toSQL()
	if err != nil {
		return fmt.Errorf("generate meter cache mv: %w", err)
	}

	if err := c.config.ClickHouse.Exec(ctx, createSQL); err != nil {
		return fmt.Errorf("create meter cache mv: %w", err)
	}

	var backfillStartedAt time.Time
	if err := c.config.ClickHouse.QueryRow(ctx, "SELECT now64(3)").Scan(&backfillStartedAt); err != nil {
		return fmt.Errorf("query backfill start time: %w", err)
	}

	if err := c.backfillMeterCache(ctx, namespace, m); err != nil {
		return fmt.Errorf("backfill meter cache: %w", err)
	}

	metadata, err := mv.metadata()
	if err != nil {
		return fmt.Errorf("generate meter cache mv metadata: %w", err)
	}

	// Truncating down keeps the heal bound conservative: a sub-second-older marker is
	// treated as not covered by the backfill rather than the other way around.
	metadata.BackfilledAt = lo.ToPtr(backfillStartedAt.UTC().Truncate(time.Second))

	comment, err := metadata.marshal()
	if err != nil {
		return fmt.Errorf("marshal meter cache mv metadata: %w", err)
	}

	stampSQL := fmt.Sprintf("ALTER TABLE %s MODIFY COMMENT %s", getTableName(c.config.Database, mv.name()), sqlStringLiteral(comment))
	if err := c.config.ClickHouse.Exec(ctx, stampSQL); err != nil {
		return fmt.Errorf("stamp meter cache mv backfill: %w", err)
	}

	return nil
}

// StampMeterCacheCoverage rewrites a deployed view's comment metadata with an advanced
// coverage watermark (covered_at), preserving every other recorded field. The reconciler
// calls it while refreshes stay continuous so that after a ClickHouse restart — which
// wipes system.view_refreshes — the durable watermark still reveals how long refreshes had
// been absent, letting the outage-repair rule fire instead of trusting the fresh-looking
// post-restart refresh state.
func (c *Connector) StampMeterCacheCoverage(ctx context.Context, view MeterCacheView, coveredAt time.Time) error {
	if !c.config.Cache.Enabled {
		return errors.New("meter cache is disabled")
	}

	if !view.MetadataOK || view.BackfilledAt == nil {
		return fmt.Errorf("refusing to stamp coverage on %q: view metadata is not converged", view.Name)
	}

	metadata := meterCacheMVMetadata{
		Namespace:    view.Namespace,
		MeterKey:     view.MeterKey,
		EventType:    view.EventType,
		MeterHash:    view.MeterHash,
		DDLHash:      view.DDLHash,
		BackfilledAt: view.BackfilledAt,
		CoveredAt:    lo.ToPtr(coveredAt.UTC().Truncate(time.Second)),
	}

	comment, err := metadata.marshal()
	if err != nil {
		return fmt.Errorf("marshal meter cache mv metadata: %w", err)
	}

	stampSQL := fmt.Sprintf("ALTER TABLE %s MODIFY COMMENT %s", getTableName(c.config.Database, view.Name), sqlStringLiteral(comment))
	if err := c.config.ClickHouse.Exec(ctx, stampSQL); err != nil {
		return fmt.Errorf("stamp meter cache mv coverage: %w", err)
	}

	return nil
}

// backfillMeterCache inserts the meter's full settled history into om_meter_cache, chunked
// on UTC month boundaries so each INSERT scans a bounded slice of the toYYYYMM-partitioned
// events table instead of all history at once.
func (c *Connector) backfillMeterCache(ctx context.Context, namespace string, m meterpkg.Meter) error {
	// The chunk range starts at the earliest event of the meter's event type; scanning for
	// it costs one aggregation over the type's rows, paid only on (re-)backfill.
	var (
		minTime    time.Time
		eventCount uint64
	)

	minTimeSQL := fmt.Sprintf("SELECT min(time), count() FROM %s WHERE namespace = ? AND type = ?", getTableName(c.config.Database, c.config.EventsTableName))
	if err := c.config.ClickHouse.QueryRow(ctx, minTimeSQL, namespace, m.EventType).Scan(&minTime, &eventCount); err != nil {
		return fmt.Errorf("query earliest event time: %w", err)
	}

	// No events at all: there is no settled history to backfill, and stamping right away is
	// correct because scheduled refreshes cover everything that arrives from here on.
	if eventCount == 0 {
		return nil
	}

	from := minTime.UTC()
	if m.EventFrom != nil && m.EventFrom.After(from) {
		from = m.EventFrom.UTC()
	}

	chunks := backfillMonthChunks(from, time.Now().UTC())

	for i, chunk := range chunks {
		backfill := meterCacheBackfill{
			Database:        c.config.Database,
			EventsTableName: c.config.EventsTableName,
			Namespace:       namespace,
			Meter:           m,
			Grain:           c.config.Cache.WindowSize,
			MinimumUsageAge: c.config.Cache.MinimumUsageAge,
			From:            &chunk.From,
		}

		// The final chunk is deliberately unbounded above: its real upper limit is the
		// ClickHouse-evaluated settled bound, and capping it at the app clock's now could
		// fall below that bound under clock skew, silently skipping freshly settled buckets.
		if i < len(chunks)-1 {
			backfill.To = &chunk.To
		}

		backfillSQL, err := backfill.toSQL()
		if err != nil {
			return err
		}

		if err := c.config.ClickHouse.Exec(ctx, backfillSQL); err != nil {
			return fmt.Errorf("backfill chunk [%s, %s): %w", chunk.From, chunk.To, err)
		}
	}

	return nil
}

// DropMeterCache drops one deployed cache MV. Dropping the view never touches its rows in
// om_meter_cache: they are unreachable for reads the moment no meter resolves to their
// meter_hash, and the reconciler's orphan-row GC removes them.
func (c *Connector) DropMeterCache(ctx context.Context, viewName string) error {
	if !c.config.Cache.Enabled {
		return errors.New("meter cache is disabled")
	}

	// The reconciler only owns objects under the cache MV name prefix; refusing anything
	// else keeps a buggy or confused caller from dropping unrelated database objects.
	if !strings.HasPrefix(viewName, meterCacheMVNamePrefix) {
		return fmt.Errorf("refusing to drop %q: not a meter cache view name", viewName)
	}

	if err := c.config.ClickHouse.Exec(ctx, "DROP VIEW IF EXISTS "+getTableName(c.config.Database, viewName)); err != nil {
		return fmt.Errorf("drop meter cache view %s: %w", viewName, err)
	}

	return nil
}

// DeleteMeterCacheOrphanRows deletes cached rows whose meter_hash no meter resolves to
// anymore (deleted meters, pre-change shapes). Orphan rows are correctness-neutral —
// every read filters on the current meter_hash (G8) — so this is storage hygiene that runs
// every reconciliation pass, guarded by a cheap existence probe so the steady state pays
// one indexed SELECT instead of a delete mutation.
//
// keepMeterHashes carries formatted (16 hex char) hashes as recorded in MV metadata. The
// keep set is hash-only on purpose: meters of different namespaces can share a shape hash,
// and rows must survive as long as any meter anywhere still resolves to their hash.
func (c *Connector) DeleteMeterCacheOrphanRows(ctx context.Context, keepMeterHashes []string) error {
	if !c.config.Cache.Enabled {
		return errors.New("meter cache is disabled")
	}

	hashes := make([]string, 0, len(keepMeterHashes))

	for _, formatted := range keepMeterHashes {
		hash, err := parseCacheHash(formatted)
		if err != nil {
			return fmt.Errorf("keep meter hash %q: %w", formatted, err)
		}

		hashes = append(hashes, strconv.FormatUint(hash, 10))
	}

	// An empty keep set means no meter is cacheable at all: every cached row is an orphan.
	where := "true"
	if len(hashes) > 0 {
		where = fmt.Sprintf("meter_hash NOT IN (%s)", strings.Join(hashes, ", "))
	}

	table := getTableName(c.config.Database, meterCacheTableName)

	rows, err := c.config.ClickHouse.Query(ctx, fmt.Sprintf("SELECT 1 FROM %s WHERE %s LIMIT 1", table, where))
	if err != nil {
		return fmt.Errorf("probe orphan meter cache rows: %w", err)
	}

	orphansExist := rows.Next()

	if err := rows.Err(); err != nil {
		rows.Close()

		return fmt.Errorf("probe orphan meter cache rows: %w", err)
	}

	rows.Close()

	if !orphansExist {
		return nil
	}

	if err := c.config.ClickHouse.Exec(ctx, fmt.Sprintf("DELETE FROM %s WHERE %s", table, where)); err != nil {
		return fmt.Errorf("delete orphan meter cache rows: %w", err)
	}

	return nil
}

// ProbeMeterCacheCapabilities checks whether this ClickHouse deployment can run the meter
// cache: refreshable materialized view state must be readable (system.view_refreshes) and
// the connecting user must hold the SYSTEM REFRESH VIEW grant the invalidator's
// best-effort refresh triggers need. It returns the server version for operator logs.
//
// Errors wrapping ErrMeterCacheUnsupported are permanent for the deployment and callers
// should disable cache reconciliation; any other error is transient (connectivity) and
// worth retrying.
func (c *Connector) ProbeMeterCacheCapabilities(ctx context.Context) (string, error) {
	if !c.config.Cache.Enabled {
		return "", errors.New("meter cache is disabled")
	}

	var version string
	if err := c.config.ClickHouse.QueryRow(ctx, "SELECT version()").Scan(&version); err != nil {
		return "", fmt.Errorf("clickhouse version query: %w", err)
	}

	var refreshableViews uint64
	if err := c.config.ClickHouse.QueryRow(ctx, "SELECT count() FROM system.view_refreshes WHERE database = ?", c.config.Database).Scan(&refreshableViews); err != nil {
		// Code 60 (UNKNOWN_TABLE) means this server has no refreshable view bookkeeping at
		// all — the feature the whole cache design schedules recomputation with.
		if strings.Contains(err.Error(), "code: 60") {
			return "", fmt.Errorf("%w: system.view_refreshes is not readable: %v", ErrMeterCacheUnsupported, err)
		}

		return "", fmt.Errorf("system.view_refreshes query: %w", err)
	}

	// Grant probe against a deliberately nonexistent view: ClickHouse checks access control
	// before resolving the target, so ACCESS_DENIED (code 497) proves the missing SYSTEM
	// REFRESH VIEW grant while an unknown-table error proves the statement passed it. The
	// probe name can never collide with a real cache MV because generated names are all-hex
	// after the prefix.
	err := c.config.ClickHouse.Exec(ctx, "SYSTEM REFRESH VIEW "+getTableName(c.config.Database, meterCacheMVNamePrefix+"grant_probe"))
	if err != nil && strings.Contains(err.Error(), "code: 497") {
		return "", fmt.Errorf("%w: missing SYSTEM REFRESH VIEW grant: %v", ErrMeterCacheUnsupported, err)
	}

	return version, nil
}

// MeterCacheRepairAge is the maximum age of a view's last successful refresh before the
// lifecycle reconciler must treat its cache content as gapped and re-backfill: the slack
// portion of the dirty window (dirty window minus the freshness horizon). Buckets settle
// continuously, and a refresh only provably covers buckets that settled within this slack
// of it (via the stored_at lookback and the newly-settled strip, both sized from it); once
// refreshes have been absent longer, buckets settled early in the outage may never be
// recomputed by any future refresh, which silent gap only a re-backfill closes.
func (c *Connector) MeterCacheRepairAge() time.Duration {
	return meterCacheDirtyWindow(c.config.Cache.MinimumUsageAge, c.config.Cache.RefreshInterval) - c.config.Cache.MinimumUsageAge
}
