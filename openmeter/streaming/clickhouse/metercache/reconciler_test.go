package metercache

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming/clickhouse"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// fakeMeterShapeHash mirrors the production property the reconciler depends on — meters
// with identical shape map to identical view names and meter hashes — without depending on
// the clickhouse package's unexported hash implementation.
func fakeMeterShapeHash(m meter.Meter) uint64 {
	h := fnv.New64a()

	parts := []string{m.EventType, string(m.Aggregation), lo.FromPtr(m.ValueProperty)}
	for _, key := range slices.Sorted(maps.Keys(m.GroupBy)) {
		parts = append(parts, key, m.GroupBy[key])
	}

	_, _ = h.Write([]byte(strings.Join(parts, "|")))

	return h.Sum64()
}

func fakeDesiredView(namespace string, m meter.Meter, ddlSalt string) clickhouse.MeterCacheDesiredView {
	shapeHash := fakeMeterShapeHash(m)

	ns := fnv.New32a()
	_, _ = ns.Write([]byte(namespace))

	ddl := fnv.New64a()
	_, _ = fmt.Fprintf(ddl, "%016x|%s", shapeHash, ddlSalt)

	return clickhouse.MeterCacheDesiredView{
		Name:      fmt.Sprintf("om_meter_cache_mv_%08x_%016x", ns.Sum32(), shapeHash),
		MeterHash: fmt.Sprintf("%016x", shapeHash),
		DDLHash:   fmt.Sprintf("%016x", ddl.Sum64()),
	}
}

type fakeConnector struct {
	repairAge time.Duration
	// ddlSalt feeds the fake DDL hash so tests can simulate config-driven DDL drift by
	// deploying an actual view recorded under a different salt.
	ddlSalt string

	actual []clickhouse.MeterCacheView

	desiredErr map[string]error
	ensureErr  map[string]error

	ensured     []string
	dropped     []string
	gcKeepSets  [][]string
	probeCalled int
	probeErr    error
}

func meterRef(namespace string, m meter.Meter) string {
	return namespace + "/" + m.Key
}

func (f *fakeConnector) DesiredMeterCacheView(namespace string, m meter.Meter) (clickhouse.MeterCacheDesiredView, error) {
	if err := f.desiredErr[meterRef(namespace, m)]; err != nil {
		return clickhouse.MeterCacheDesiredView{}, err
	}

	return fakeDesiredView(namespace, m, f.ddlSalt), nil
}

func (f *fakeConnector) ListActualViews(ctx context.Context) ([]clickhouse.MeterCacheView, error) {
	return slices.Clone(f.actual), nil
}

func (f *fakeConnector) EnsureMeterCache(ctx context.Context, namespace string, m meter.Meter) error {
	f.ensured = append(f.ensured, meterRef(namespace, m))

	return f.ensureErr[meterRef(namespace, m)]
}

func (f *fakeConnector) DropMeterCache(ctx context.Context, viewName string) error {
	f.dropped = append(f.dropped, viewName)

	return nil
}

func (f *fakeConnector) DeleteMeterCacheOrphanRows(ctx context.Context, keepMeterHashes []string) error {
	f.gcKeepSets = append(f.gcKeepSets, slices.Clone(keepMeterHashes))

	return nil
}

func (f *fakeConnector) ProbeMeterCacheCapabilities(ctx context.Context) (string, error) {
	f.probeCalled++

	return "25.12.3.12345", f.probeErr
}

func (f *fakeConnector) MeterCacheRepairAge() time.Duration {
	// A zero repair age would make every deployed view look outage-aged; default to a value
	// comfortably above the freshly-refreshed timestamps deployedView fabricates.
	if f.repairAge == 0 {
		return 30 * time.Minute
	}

	return f.repairAge
}

type fakeMeterService struct {
	meters     []meter.Meter
	lastParams meter.ListMetersParams
}

func (f *fakeMeterService) ListMeters(ctx context.Context, params meter.ListMetersParams) (pagination.Result[meter.Meter], error) {
	f.lastParams = params

	return pagination.Result[meter.Meter]{Items: slices.Clone(f.meters), TotalCount: len(f.meters)}, nil
}

func (f *fakeMeterService) GetMeterByIDOrSlug(ctx context.Context, input meter.GetMeterInput) (meter.Meter, error) {
	return meter.Meter{}, errors.New("not implemented")
}

func newTestMeter(namespace, key, eventType string, groupBy map[string]string) meter.Meter {
	return meter.Meter{
		ManagedResource: models.ManagedResource{
			NamespacedModel: models.NamespacedModel{Namespace: namespace},
			ID:              key,
			Name:            key,
		},
		Key:           key,
		EventType:     eventType,
		Aggregation:   meter.MeterAggregationSum,
		ValueProperty: lo.ToPtr("$.value"),
		GroupBy:       groupBy,
	}
}

// newTestReconciler builds a pass-only reconciler: unit tests drive reconcile directly, so
// the leader lock and lifecycle fields stay zero.
func newTestReconciler(connector Connector, meters meter.Service) *Reconciler {
	return &Reconciler{
		logger:    slog.Default(),
		connector: connector,
		meters:    meters,
	}
}

// deployedView renders the actual-state view EnsureMeterCache would leave behind for a
// meter under the fake hash scheme: matching hashes, stamped, recently refreshed.
func deployedView(namespace string, m meter.Meter, ddlSalt string, now time.Time) clickhouse.MeterCacheView {
	desired := fakeDesiredView(namespace, m, ddlSalt)

	return clickhouse.MeterCacheView{
		Name:            desired.Name,
		MetadataOK:      true,
		MeterKey:        m.Key,
		EventType:       m.EventType,
		MeterHash:       desired.MeterHash,
		DDLHash:         desired.DDLHash,
		BackfilledAt:    lo.ToPtr(now.Add(-time.Minute)),
		LastSuccessTime: lo.ToPtr(now.Add(-time.Minute)),
	}
}

func TestReconcile(t *testing.T) {
	now := time.Now()

	t.Run("CreatesMissingViewAndListsWithoutNamespace", func(t *testing.T) {
		// given:
		// - one meter, no deployed views
		// when:
		// - a pass runs
		// then:
		// - the meter's view is ensured, its hash is kept by GC, and the meter listing is
		//   cross-namespace excluding soft-deleted meters
		m := newTestMeter("ns-1", "meter-1", "api-calls", nil)
		connector := &fakeConnector{}
		meterService := &fakeMeterService{meters: []meter.Meter{m}}

		r := newTestReconciler(connector, meterService)
		require.NoError(t, r.reconcile(t.Context()))

		require.Equal(t, []string{"ns-1/meter-1"}, connector.ensured)
		require.Empty(t, connector.dropped)
		require.Equal(t, [][]string{{fakeDesiredView("ns-1", m, "").MeterHash}}, connector.gcKeepSets)

		require.True(t, meterService.lastParams.WithoutNamespace)
		require.False(t, meterService.lastParams.IncludeDeleted)
	})

	t.Run("ConvergedViewIsLeftAlone", func(t *testing.T) {
		// given:
		// - one meter whose deployed view matches on every axis (hashes, key, stamp, health)
		// then:
		// - the pass performs no view mutation, only the GC probe
		m := newTestMeter("ns-1", "meter-1", "api-calls", nil)
		connector := &fakeConnector{actual: []clickhouse.MeterCacheView{deployedView("ns-1", m, "", now)}}

		r := newTestReconciler(connector, &fakeMeterService{meters: []meter.Meter{m}})
		require.NoError(t, r.reconcile(t.Context()))

		require.Empty(t, connector.ensured)
		require.Empty(t, connector.dropped)
		require.Len(t, connector.gcKeepSets, 1)
	})

	t.Run("DropsViewOfDeletedMeter", func(t *testing.T) {
		// given:
		// - a deployed view but no meter desiring it (the meter was deleted)
		// then:
		// - the view is dropped and the GC keep set is empty (rows are orphans)
		m := newTestMeter("ns-1", "meter-1", "api-calls", nil)
		view := deployedView("ns-1", m, "", now)
		connector := &fakeConnector{actual: []clickhouse.MeterCacheView{view}}

		r := newTestReconciler(connector, &fakeMeterService{})
		require.NoError(t, r.reconcile(t.Context()))

		require.Empty(t, connector.ensured)
		require.Equal(t, []string{view.Name}, connector.dropped)
		require.Len(t, connector.gcKeepSets, 1)
		require.Empty(t, connector.gcKeepSets[0])
	})

	t.Run("ShapeChangeSwapsView", func(t *testing.T) {
		// given:
		// - a view deployed for the meter's previous shape (no group-by), while the meter now
		//   has a group-by dimension (different hash, different name)
		// then:
		// - the old view is dropped, the new one is ensured, and only the new hash is kept
		oldShape := newTestMeter("ns-1", "meter-1", "api-calls", nil)
		newShape := newTestMeter("ns-1", "meter-1", "api-calls", map[string]string{"group1": "$.group1"})

		oldView := deployedView("ns-1", oldShape, "", now)
		connector := &fakeConnector{actual: []clickhouse.MeterCacheView{oldView}}

		r := newTestReconciler(connector, &fakeMeterService{meters: []meter.Meter{newShape}})
		require.NoError(t, r.reconcile(t.Context()))

		require.Equal(t, []string{oldView.Name}, connector.dropped)
		require.Equal(t, []string{"ns-1/meter-1"}, connector.ensured)
		require.Equal(t, [][]string{{fakeDesiredView("ns-1", newShape, "").MeterHash}}, connector.gcKeepSets)
	})

	t.Run("DDLDriftRecreatesView", func(t *testing.T) {
		// given:
		// - a deployed view recorded under a different DDL hash (config or generator change)
		// then:
		// - the view is dropped and re-ensured (re-backfill is implied by ensure)
		m := newTestMeter("ns-1", "meter-1", "api-calls", nil)
		staleView := deployedView("ns-1", m, "old-config", now)
		connector := &fakeConnector{actual: []clickhouse.MeterCacheView{staleView}}

		r := newTestReconciler(connector, &fakeMeterService{meters: []meter.Meter{m}})
		require.NoError(t, r.reconcile(t.Context()))

		require.Equal(t, []string{staleView.Name}, connector.dropped)
		require.Equal(t, []string{"ns-1/meter-1"}, connector.ensured)
	})

	t.Run("UnstampedViewIsEnsuredWithoutDrop", func(t *testing.T) {
		// given:
		// - a matching view whose backfill stamp is missing (leader died mid-backfill)
		// then:
		// - ensure repairs it in place; no drop, so scheduled refreshes keep running
		m := newTestMeter("ns-1", "meter-1", "api-calls", nil)
		view := deployedView("ns-1", m, "", now)
		view.BackfilledAt = nil
		connector := &fakeConnector{actual: []clickhouse.MeterCacheView{view}}

		r := newTestReconciler(connector, &fakeMeterService{meters: []meter.Meter{m}})
		require.NoError(t, r.reconcile(t.Context()))

		require.Empty(t, connector.dropped)
		require.Equal(t, []string{"ns-1/meter-1"}, connector.ensured)
	})

	t.Run("AliasRejectedMeterIsSkipped", func(t *testing.T) {
		// given:
		// - a meter the generator refuses (simulated via desiredErr) and a healthy meter
		// then:
		// - only the healthy meter is ensured and only its hash is kept
		rejected := newTestMeter("ns-1", "meter-rejected", "api-calls", map[string]string{"windowstart": "$.ws"})
		healthy := newTestMeter("ns-1", "meter-healthy", "api-calls", nil)

		connector := &fakeConnector{
			desiredErr: map[string]error{"ns-1/meter-rejected": errors.New("reserved alias")},
		}

		r := newTestReconciler(connector, &fakeMeterService{meters: []meter.Meter{rejected, healthy}})
		require.NoError(t, r.reconcile(t.Context()))

		require.Equal(t, []string{"ns-1/meter-healthy"}, connector.ensured)
		require.Equal(t, [][]string{{fakeDesiredView("ns-1", healthy, "").MeterHash}}, connector.gcKeepSets)
	})

	t.Run("EnsureFailureDoesNotBlockOtherMeters", func(t *testing.T) {
		// given:
		// - two missing views where the first ensure fails
		// then:
		// - the pass still ensures the second and runs GC, and reports the failure
		m1 := newTestMeter("ns-1", "meter-1", "api-calls", nil)
		m2 := newTestMeter("ns-1", "meter-2", "other-calls", nil)

		connector := &fakeConnector{
			ensureErr: map[string]error{"ns-1/meter-1": errors.New("boom")},
		}

		r := newTestReconciler(connector, &fakeMeterService{meters: []meter.Meter{m1, m2}})
		err := r.reconcile(t.Context())
		require.ErrorContains(t, err, "ns-1/meter-1")

		require.ElementsMatch(t, []string{"ns-1/meter-1", "ns-1/meter-2"}, connector.ensured)
		require.Len(t, connector.gcKeepSets, 1)
	})

	t.Run("SameShapeConflictOwnershipFollowsDeployedView", func(t *testing.T) {
		// given:
		// - two meters with identical shape in one namespace (one hash, one view name) where
		//   the deployed view was created for the (namespace, key)-later meter
		// then:
		// - the deployed owner keeps the view (no flapping) and the other meter reads live
		first := newTestMeter("ns-1", "meter-a", "api-calls", nil)
		second := newTestMeter("ns-1", "meter-b", "api-calls", nil)

		connector := &fakeConnector{actual: []clickhouse.MeterCacheView{deployedView("ns-1", second, "", now)}}

		r := newTestReconciler(connector, &fakeMeterService{meters: []meter.Meter{first, second}})
		require.NoError(t, r.reconcile(t.Context()))

		require.Empty(t, connector.dropped)
		require.Empty(t, connector.ensured, "the deployed view already serves the owner")
	})

	t.Run("SameShapeConflictWithoutDeployedViewFirstKeyWins", func(t *testing.T) {
		// given:
		// - the same conflict with nothing deployed yet
		// then:
		// - exactly one view is ensured, for the (namespace, key)-first meter
		first := newTestMeter("ns-1", "meter-a", "api-calls", nil)
		second := newTestMeter("ns-1", "meter-b", "api-calls", nil)

		connector := &fakeConnector{}

		r := newTestReconciler(connector, &fakeMeterService{meters: []meter.Meter{second, first}})
		require.NoError(t, r.reconcile(t.Context()))

		require.Equal(t, []string{"ns-1/meter-a"}, connector.ensured)
	})
}

func TestDisabledReconcilerParksUntilClose(t *testing.T) {
	// given:
	// - a reconciler constructed with the cache disabled (the unconditional server wiring)
	// when/then:
	// - Start must block until Close: it runs as a run.Group actor, and an early return
	//   would shut down the whole server
	r, err := New(Config{Logger: slog.Default()})
	require.NoError(t, err)

	started := make(chan error, 1)
	go func() {
		started <- r.Start()
	}()

	select {
	case err := <-started:
		t.Fatalf("disabled reconciler returned from Start before Close: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	require.NoError(t, r.Close())

	select {
	case err := <-started:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("disabled reconciler did not stop after Close")
	}
}

func TestPlanViewAction(t *testing.T) {
	now := time.Now()
	repairAge := 30 * time.Minute

	m := newTestMeter("ns-1", "meter-1", "api-calls", nil)
	desired := desiredView{meter: m, view: fakeDesiredView("ns-1", m, "")}

	healthy := deployedView("ns-1", m, "", now)

	tests := []struct {
		name   string
		actual func() *clickhouse.MeterCacheView
		action viewAction
	}{
		{
			name:   "missing view is ensured",
			actual: func() *clickhouse.MeterCacheView { return nil },
			action: viewActionEnsure,
		},
		{
			name: "healthy converged view needs nothing",
			actual: func() *clickhouse.MeterCacheView {
				view := healthy
				return &view
			},
			action: viewActionNone,
		},
		{
			name: "unparseable metadata is recreated",
			actual: func() *clickhouse.MeterCacheView {
				view := healthy
				view.MetadataOK = false
				return &view
			},
			action: viewActionRecreate,
		},
		{
			name: "meter key mismatch is recreated",
			actual: func() *clickhouse.MeterCacheView {
				view := healthy
				view.MeterKey = "someone-else"
				return &view
			},
			action: viewActionRecreate,
		},
		{
			name: "ddl drift is recreated",
			actual: func() *clickhouse.MeterCacheView {
				view := healthy
				view.DDLHash = "0000000000000000"
				return &view
			},
			action: viewActionRecreate,
		},
		{
			name: "unstamped backfill is ensured",
			actual: func() *clickhouse.MeterCacheView {
				view := healthy
				view.BackfilledAt = nil
				return &view
			},
			action: viewActionEnsure,
		},
		{
			name: "extended refresh outage is repaired",
			actual: func() *clickhouse.MeterCacheView {
				view := healthy
				view.LastSuccessTime = lo.ToPtr(now.Add(-repairAge - time.Minute))
				view.BackfilledAt = lo.ToPtr(now.Add(-repairAge - time.Minute))
				return &view
			},
			action: viewActionEnsure,
		},
		{
			name: "outage repair is throttled by a fresh backfill stamp",
			actual: func() *clickhouse.MeterCacheView {
				view := healthy
				view.LastSuccessTime = lo.ToPtr(now.Add(-repairAge - time.Minute))
				view.BackfilledAt = lo.ToPtr(now.Add(-time.Minute))
				return &view
			},
			action: viewActionNone,
		},
		{
			name: "never refreshed view is not repaired",
			actual: func() *clickhouse.MeterCacheView {
				view := healthy
				view.LastSuccessTime = nil
				view.BackfilledAt = lo.ToPtr(now.Add(-repairAge - time.Minute))
				return &view
			},
			action: viewActionNone,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			action, _ := planViewAction(desired, test.actual(), repairAge, now)
			require.Equal(t, test.action, action)
		})
	}
}
