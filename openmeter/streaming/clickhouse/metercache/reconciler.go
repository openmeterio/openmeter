// Package metercache runs the meter cache lifecycle reconciler: the periodic in-server
// service that converges the deployed set of per-meter refreshable materialized views onto
// the current meter definitions. ClickHouse owns refresh scheduling and recomputation; the
// reconciler owns everything with a lifecycle — creating views for new meters, swapping
// views whose meter shape changed, dropping views of deleted meters, repairing half-done
// or gapped deployments, and garbage collecting orphaned rollup rows.
package metercache

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"runtime/debug"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"cirello.io/pglock"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming/clickhouse"
	"github.com/openmeterio/openmeter/pkg/models"
)

// DefaultReconcileInterval paces reconciliation passes. One minute is deliberately much
// shorter than the cache refresh interval: meter CRUD is only observed by polling (the
// server consumes no Kafka in-process), so this bounds how long a new or changed meter
// reads live before its view exists.
const DefaultReconcileInterval = time.Minute

// reconcilerLeaderLockKey serializes reconciliation globally: DDL against the shared
// om_meter_cache target and full-history backfills must run from a single actor at a time.
const reconcilerLeaderLockKey = "streaming.meter_cache.reconcile_lock"

// Connector is the meter cache manager surface of the ClickHouse streaming connector the
// reconciler drives. It is an interface so reconciliation planning is testable without a
// ClickHouse connection; *clickhouse.Connector is the production implementation.
type Connector interface {
	DesiredMeterCacheView(namespace string, m meter.Meter) (clickhouse.MeterCacheDesiredView, error)
	ListActualViews(ctx context.Context) ([]clickhouse.MeterCacheView, error)
	EnsureMeterCache(ctx context.Context, namespace string, m meter.Meter) error
	DropMeterCache(ctx context.Context, viewName string) error
	DeleteMeterCacheOrphanRows(ctx context.Context, keepMeterHashes []string) error
	StampMeterCacheCoverage(ctx context.Context, view clickhouse.MeterCacheView, coveredAt time.Time) error
	ReconcileMeterCacheMarkers(ctx context.Context, views []clickhouse.MeterCacheView) ([]string, error)
	ProbeMeterCacheCapabilities(ctx context.Context) (string, error)
	MeterCacheRepairAge() time.Duration
}

var _ Connector = (*clickhouse.Connector)(nil)

type Config struct {
	// Enabled mirrors aggregation.cache.enabled. A disabled reconciler is still constructed
	// and started so the server wiring stays unconditional; it parks idle until closed.
	Enabled bool

	Logger     *slog.Logger
	Connector  Connector
	Meters     meter.Service
	LockClient *pglock.Client

	ReconcileInterval time.Duration
}

func (c Config) Validate() error {
	var errs []error

	if c.Logger == nil {
		errs = append(errs, errors.New("logger is required"))
	}

	if c.Enabled {
		if c.Connector == nil {
			errs = append(errs, errors.New("connector is required"))
		}

		if c.Meters == nil {
			errs = append(errs, errors.New("meter service is required"))
		}

		if c.LockClient == nil {
			errs = append(errs, errors.New("distributed lock client is required"))
		}
	}

	if c.ReconcileInterval < 0 {
		errs = append(errs, errors.New("reconcile interval must not be negative"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// Reconciler is the meter cache lifecycle reconciler. Start blocks running reconciliation
// passes until Close, holding one global Postgres lock across each leadership term so at
// most one instance mutates cache views at a time (same actor pattern as the notification
// event handler).
type Reconciler struct {
	logger     *slog.Logger
	connector  Connector
	meters     meter.Service
	lockClient *pglock.Client

	enabled           bool
	reconcileInterval time.Duration

	running atomic.Bool
	// disabled latches when the capability probe reports the deployment can never run the
	// cache; it stops reconciliation attempts without stopping the process.
	disabled    atomic.Bool
	ctxCancel   context.CancelFunc
	stopCh      chan struct{}
	stopChClose func()
}

func New(config Config) (*Reconciler, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	if config.ReconcileInterval == 0 {
		config.ReconcileInterval = DefaultReconcileInterval
	}

	stopCh := make(chan struct{})

	return &Reconciler{
		logger:            config.Logger,
		connector:         config.Connector,
		meters:            config.Meters,
		lockClient:        config.LockClient,
		enabled:           config.Enabled,
		reconcileInterval: config.ReconcileInterval,
		stopCh:            stopCh,
		stopChClose:       sync.OnceFunc(func() { close(stopCh) }),
	}, nil
}

func (r *Reconciler) Start() error {
	if !r.running.CompareAndSwap(false, true) {
		return errors.New("meter cache reconciler is already running")
	}

	defer func() {
		if err := recover(); err != nil {
			r.logger.Error("meter cache reconciler panicked",
				"error", err,
				"code.stacktrace", string(debug.Stack()))
			_ = r.Close()
		}
	}()

	var ctx context.Context

	ctx, r.ctxCancel = context.WithCancel(context.Background())
	defer r.ctxCancel()

	if !r.enabled {
		r.logger.Debug("meter cache is disabled, reconciler is idle")
	}

	for r.running.Load() && r.enabled && !r.disabled.Load() {
		err := r.lockClient.Do(ctx, reconcilerLeaderLockKey, func(rCtx context.Context, _ *pglock.Lock) error {
			return r.lead(rCtx)
		})
		if err != nil {
			if errors.Is(err, pglock.ErrNotAcquired) {
				r.logger.DebugContext(ctx, "meter cache reconciliation skipped: lock is not acquired")
				continue
			}

			return fmt.Errorf("failed to acquire meter cache reconciliation lock: %w", err)
		}
	}

	// This method runs as a run.Group actor: returning shuts the whole process down. A
	// reconciler that has nothing to do (disabled by config or by the capability probe)
	// therefore parks here until Close instead of returning early.
	<-r.stopCh

	return nil
}

func (r *Reconciler) Close() error {
	if r.running.CompareAndSwap(true, false) {
		r.logger.Debug("closing meter cache reconciler")

		r.ctxCancel()
		r.stopChClose()
	}

	return nil
}

// lead runs reconciliation passes while this instance holds the leader lock. The lock
// client cancels ctx when the lease is lost, aborting a pass mid-flight; that is safe
// because every pass operation is idempotent and a duplicate backfill from the next leader
// resolves by newest-wins against whatever the aborted one already inserted.
func (r *Reconciler) lead(ctx context.Context) error {
	ticker := time.NewTicker(r.reconcileInterval)
	defer ticker.Stop()

	probed := false

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-r.stopCh:
			r.logger.DebugContext(ctx, "close event received: stopping meter cache reconciler")
			return nil
		case <-ticker.C:
			// The capability probe gates the first pass of each leadership term: an
			// unsupported deployment (no refreshable views, missing grants) disables the
			// reconciler for the process lifetime instead of failing the server, while a
			// transient probe failure (ClickHouse briefly unreachable) is retried on the
			// next tick.
			if !probed {
				version, err := r.connector.ProbeMeterCacheCapabilities(ctx)
				if err != nil {
					if errors.Is(err, clickhouse.ErrMeterCacheUnsupported) {
						r.disabled.Store(true)
						r.logger.ErrorContext(ctx, "meter cache reconciler disabled: ClickHouse deployment lacks required capabilities", "error", err)

						return nil
					}

					r.logger.WarnContext(ctx, "meter cache capability probe failed, retrying", "error", err)

					continue
				}

				r.logger.InfoContext(ctx, "meter cache reconciler active", "clickhouse_version", version)

				probed = true
			}

			if err := r.reconcile(ctx); err != nil {
				r.logger.ErrorContext(ctx, "meter cache reconciliation pass failed", "error", err)
			}
		}
	}
}

// reconcile runs one full reconciliation pass: desired state from the meter definitions,
// actual state from ClickHouse, then per-view convergence and orphan-row GC. Per-view
// failures are collected instead of aborting the pass so one broken meter cannot block
// every other meter's lifecycle.
func (r *Reconciler) reconcile(ctx context.Context) error {
	// The zero page returns all meters; soft-deleted ones are excluded by default, which is
	// exactly the deletion signal: a deleted meter's view stops being desired.
	meterList, err := r.meters.ListMeters(ctx, meter.ListMetersParams{WithoutNamespace: true})
	if err != nil {
		return fmt.Errorf("list meters: %w", err)
	}

	actual, err := r.connector.ListActualViews(ctx)
	if err != nil {
		return fmt.Errorf("list actual meter cache views: %w", err)
	}

	actualByName := make(map[string]clickhouse.MeterCacheView, len(actual))
	for _, view := range actual {
		actualByName[view.Name] = view
	}

	desired, keepHashes := r.desiredState(meterList.Items, actualByName)

	var errs []error

	// Marker maintenance runs before planning: expired-unhealed markers force a re-backfill
	// of their view, and healed markers must be deleted while the healing refresh is still
	// the latest one observable — the reader's heal rule regresses once refreshes advance
	// past the heal bound. A failed maintenance pass degrades to conservative reads (live
	// fallback on the marked ranges), never to trusting a stale bucket.
	markerRepairs, err := r.connector.ReconcileMeterCacheMarkers(ctx, actual)
	if err != nil {
		errs = append(errs, fmt.Errorf("reconcile meter cache markers: %w", err))
	}

	// Undesired views first (deleted meters, pre-change shapes, foreign prefix squatters):
	// dropping before creating keeps a shape swap from briefly running two refresh
	// schedules against the shared target.
	for _, view := range actual {
		if _, ok := desired[view.Name]; ok {
			continue
		}

		r.logger.InfoContext(ctx, "meter cache: dropping view without a desired meter", "view", view.Name, "meter", view.MeterKey)

		if err := r.connector.DropMeterCache(ctx, view.Name); err != nil {
			errs = append(errs, fmt.Errorf("drop meter cache view %s: %w", view.Name, err))
		}
	}

	repairAge := r.connector.MeterCacheRepairAge()
	now := time.Now()

	for _, name := range slices.Sorted(maps.Keys(desired)) {
		d := desired[name]

		var actualView *clickhouse.MeterCacheView
		if view, ok := actualByName[name]; ok {
			actualView = &view
		}

		action, reason := planViewAction(d, actualView, slices.Contains(markerRepairs, name), repairAge, now)
		if action == viewActionNone {
			// A converged view's coverage watermark advances with its refreshes so it stays
			// meaningful across ClickHouse restarts (which wipe system.view_refreshes); the
			// stamp fires at most once per refresh cycle because LastSuccessTime only moves
			// when a refresh completes.
			if watermark, ok := coverageWatermark(actualView); ok &&
				actualView.LastSuccessTime != nil && actualView.LastSuccessTime.After(watermark) {
				if err := r.connector.StampMeterCacheCoverage(ctx, *actualView, *actualView.LastSuccessTime); err != nil {
					errs = append(errs, fmt.Errorf("stamp meter cache coverage for %s: %w", name, err))
				}
			}

			continue
		}

		r.logger.InfoContext(ctx, "meter cache: converging view",
			"view", name, "namespace", d.meter.Namespace, "meter", d.meter.Key, "action", string(action), "reason", reason)

		if action == viewActionRecreate {
			if err := r.connector.DropMeterCache(ctx, name); err != nil {
				errs = append(errs, fmt.Errorf("drop meter cache view %s: %w", name, err))

				continue
			}
		}

		if err := r.connector.EnsureMeterCache(ctx, d.meter.Namespace, d.meter); err != nil {
			errs = append(errs, fmt.Errorf("ensure meter cache for %s/%s: %w", d.meter.Namespace, d.meter.Key, err))
		}
	}

	// Orphan-row GC runs every pass, last: the ensures above may have written new-shape
	// rows whose hashes must be in the keep set by the time old-shape rows are deleted.
	// Rows a dying previous leader's in-flight old-shape backfill lands after this delete
	// stay orphaned until the next pass, which is acceptable because reads filter on the
	// current meter hash and never see them (G8).
	if err := r.connector.DeleteMeterCacheOrphanRows(ctx, keepHashes); err != nil {
		errs = append(errs, fmt.Errorf("delete orphan meter cache rows: %w", err))
	}

	return errors.Join(errs...)
}

// desiredView is one entry of the reconciler's desired state: the meter that should be
// served by the view and the name/hashes its deployment must converge to.
type desiredView struct {
	meter meter.Meter
	view  clickhouse.MeterCacheDesiredView
}

// desiredState maps the current meter definitions onto desired cache views, keyed by view
// name, plus the meter-hash keep set for orphan-row GC.
//
// Meters the generator refuses (reserved group-by aliases, G9) are logged and skipped —
// they are served live by the read gate, never a reconciliation error. Their hashes are
// also absent from the keep set, which is safe because the same generator refusal means no
// rows were ever written for such a shape.
func (r *Reconciler) desiredState(meters []meter.Meter, actualByName map[string]clickhouse.MeterCacheView) (map[string]desiredView, []string) {
	// Deterministic order makes same-shape conflict resolution stable across passes.
	sorted := slices.SortedFunc(slices.Values(meters), func(a, b meter.Meter) int {
		if c := strings.Compare(a.Namespace, b.Namespace); c != 0 {
			return c
		}

		return strings.Compare(a.Key, b.Key)
	})

	desired := make(map[string]desiredView, len(sorted))
	keepHashes := make([]string, 0, len(sorted))

	for _, m := range sorted {
		dv, err := r.connector.DesiredMeterCacheView(m.Namespace, m)
		if err != nil {
			r.logger.Warn("meter cache: meter is not cacheable, serving it live", "namespace", m.Namespace, "meter", m.Key, "error", err)

			continue
		}

		keepHashes = append(keepHashes, dv.MeterHash)

		current, taken := desired[dv.Name]
		if !taken {
			desired[dv.Name] = desiredView{meter: m, view: dv}

			continue
		}

		// Same-shape conflict: meters with identical shape share a meter hash and thus one
		// view name — either within one namespace or across namespaces whose 8-hex name
		// folds collide — but the view stamps a single namespace and meter_key into its
		// rows, so it can only ever serve one of them; the read gate refuses it for the
		// others (they stay live). Ownership prefers the meter the deployed view was
		// created for so it does not flap between passes; without a deployed view the
		// (namespace, key)-first meter wins.
		loser := m
		if actual, ok := actualByName[dv.Name]; ok && actual.MetadataOK &&
			actual.Namespace == m.Namespace && actual.MeterKey == m.Key &&
			(current.meter.Namespace != m.Namespace || current.meter.Key != m.Key) {
			loser = current.meter
			desired[dv.Name] = desiredView{meter: m, view: dv}
		}

		r.logger.Warn("meter cache: meter shares its cache shape with another meter, serving it live",
			"namespace", loser.Namespace, "meter", loser.Key, "owner", desired[dv.Name].meter.Key)
	}

	slices.Sort(keepHashes)

	return desired, slices.Compact(keepHashes)
}

// viewAction is the reconciler's convergence decision for one desired cache view.
type viewAction string

const (
	viewActionNone viewAction = "none"
	// viewActionEnsure creates the view if missing, backfills settled history, and stamps
	// backfilled_at; over an existing view it is the repair path (re-backfill + re-stamp).
	viewActionEnsure viewAction = "ensure"
	// viewActionRecreate drops the deployed view first, then performs ensure.
	viewActionRecreate viewAction = "recreate"
)

// coverageWatermark returns a deployed view's durable coverage watermark: the newest
// instant up to which its cache content is known continuously maintained. It is the later
// of the backfill stamp (a completed backfill covers all history settled before it) and
// the covered_at stamp the reconciler advances while refreshes stay continuous. ok is
// false while the view is unstamped — there is no coverage to speak of before the first
// backfill completes.
func coverageWatermark(view *clickhouse.MeterCacheView) (time.Time, bool) {
	if view == nil || view.BackfilledAt == nil {
		return time.Time{}, false
	}

	watermark := *view.BackfilledAt
	if view.CoveredAt != nil && view.CoveredAt.After(watermark) {
		watermark = *view.CoveredAt
	}

	return watermark, true
}

// planViewAction decides how to converge one desired view given its deployed counterpart
// (nil when not deployed) and whether marker maintenance flagged it for repair. It returns
// the action plus a short reason for operator logs.
func planViewAction(d desiredView, actual *clickhouse.MeterCacheView, needsMarkerRepair bool, repairAge time.Duration, now time.Time) (viewAction, string) {
	if actual == nil {
		return viewActionEnsure, "view missing"
	}

	if !actual.MetadataOK {
		return viewActionRecreate, "unparseable view metadata"
	}

	// A hash mismatch at the desired name means the deployed object was not produced by the
	// current generator for this meter (corrupt or hand-altered comment); a
	// namespace/key/event-type mismatch means it was created for a same-shape meter — or a
	// name-fold-colliding namespace's meter — that is no longer desired (had it been,
	// ownership in desiredState would have followed the deployed view).
	if actual.Namespace != d.meter.Namespace || actual.MeterKey != d.meter.Key ||
		actual.EventType != d.meter.EventType || actual.MeterHash != d.view.MeterHash {
		return viewActionRecreate, "view metadata mismatch"
	}

	// DDL drift: refresh cadence, freshness horizon, EventFrom, or the generated SELECT
	// changed since deployment. The recreate always re-backfills even though only some
	// drifts strictly need it (a decreased usage age or an earlier EventFrom leave settled
	// history the old view never cached): the stored hash cannot reveal which input moved,
	// skipping a needed re-backfill silently undercounts forever, and a redundant one is
	// idempotent under newest-wins.
	if actual.DDLHash != d.view.DDLHash {
		return viewActionRecreate, "ddl drift"
	}

	// G3: an unstamped view is a create-or-backfill sequence some actor started and never
	// finished (leader crash mid-backfill). Ensure re-runs backfill and stamps; the create
	// inside it is a no-op on the existing view.
	if actual.BackfilledAt == nil {
		return viewActionEnsure, "backfill unstamped"
	}

	// Extended refresh outage: refreshes only recompute buckets that settled within the
	// dirty-window slack (repairAge) of them, so any longer refresh gap leaves buckets
	// settled early in the gap permanently missing from the cache, and only a re-backfill
	// closes it. The gap is measured against the durable coverage watermark, not against
	// system.view_refreshes alone: that table is per-server in-memory state a ClickHouse
	// restart wipes, so after a restart the refresh state is first nil and then fresh —
	// both would hide an outage that preceded or spanned the restart. A repair re-stamps
	// the backfill (advancing the watermark), which doubles as the repair throttle: a view
	// whose refreshes stay broken is re-repaired at most once per repairAge instead of on
	// every pass. All three arms below share that throttle:
	//
	//   - refreshes resumed after a gap wider than the watermark can vouch for,
	//   - refreshes have been failing for longer than repairAge (the watermark stopped
	//     advancing with them),
	//   - no refresh since ClickHouse startup while the watermark already aged out — the
	//     restart followed an outage the post-restart refresh state cannot reveal. A fresh
	//     watermark with nil refresh state is a fresh restart and deliberately not
	//     repaired: the first scheduled refresh covers it, and mass re-backfilling every
	//     view on every ClickHouse restart would be a self-inflicted load storm.
	if watermark, ok := coverageWatermark(actual); ok {
		if actual.LastSuccessTime != nil {
			if actual.LastSuccessTime.Sub(watermark) > repairAge {
				return viewActionEnsure, "refresh outage"
			}

			if now.Sub(*actual.LastSuccessTime) > repairAge && now.Sub(watermark) > repairAge {
				return viewActionEnsure, "refresh outage"
			}
		} else if now.Sub(watermark) > repairAge {
			return viewActionEnsure, "refresh outage"
		}
	}

	// Expired-unhealed invalidation markers: late events landed in settled buckets while
	// refreshes could not cover them, and the heal window has passed — no future refresh
	// can prove the buckets converged, so reads of the marked ranges stay live until a
	// re-backfill covers the markers (after which marker maintenance deletes them).
	if needsMarkerRepair {
		return viewActionEnsure, "expired unhealed markers"
	}

	return viewActionNone, ""
}
