package worker

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric/noop"

	eventmodels "github.com/openmeterio/openmeter/openmeter/event/models"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/rte"
	ingestevents "github.com/openmeterio/openmeter/openmeter/sink/flushhandler/ingestnotification/events"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
)

type fixedSources struct{ selectors []rte.DemandSelector }

func (s fixedSources) Resolve(context.Context, marshaler.Event) ([]rte.DemandSelector, error) {
	return s.selectors, nil
}

type recordingDemand struct {
	rte.DemandService
	invalidate func(context.Context, rte.InvalidateInput) error
}

func (d recordingDemand) Invalidate(ctx context.Context, input rte.InvalidateInput) error {
	return d.invalidate(ctx, input)
}

func TestCreditUsageVisibilityWait(t *testing.T) {
	t.Run("already visible", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
		defer cancel()
		require.NoError(t, waitForCreditUsage(ctx, time.Now().Add(-2*time.Second)))
	})
	t.Run("cancellation", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
		defer cancel()
		require.ErrorIs(t, waitForCreditUsage(ctx, time.Now().Add(time.Hour)), context.DeadlineExceeded)
	})
	t.Run("future clock without processing timeout", func(t *testing.T) {
		require.ErrorContains(t, waitForCreditUsage(t.Context(), time.Now().Add(time.Hour)), "ahead of the credit visibility window")
	})
	t.Run("missing stored timestamp", func(t *testing.T) {
		require.ErrorContains(t, waitForCreditUsage(t.Context(), time.Time{}), "storedAt is required")
	})
}

func TestSourceInvalidationWaitsOnlyForCreditIngest(t *testing.T) {
	for _, test := range []struct {
		name     string
		kind     rte.CalculationKind
		ingest   bool
		storedAt time.Time
	}{
		{name: "credit batch reaches first visible second", kind: rte.CalculationKindCreditBalance, ingest: true, storedAt: time.Now()},
		{name: "metered batch skips credit cutoff", kind: rte.CalculationKindMeteredEntitlement, ingest: true, storedAt: time.Now().Add(time.Hour)},
		{name: "ledger fact skips ingest cutoff", kind: rte.CalculationKindCreditBalance},
	} {
		t.Run(test.name, func(t *testing.T) {
			// Given a routed fact and a demand spy that observes invalidation time.
			var event marshaler.Event = &ledger.TransactionGroupCommittedEvent{Namespace: "ns", GroupID: "group"}
			if test.ingest {
				event = &ingestevents.EventBatchedIngest{Namespace: eventmodels.NamespaceID{ID: "ns"}, SubjectKey: "subject", StoredAt: test.storedAt}
			}
			msg, err := marshaler.New(nil).Marshal(event)
			require.NoError(t, err)
			ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
			defer cancel()
			msg.SetContext(ctx)
			calls := 0
			w := sourceWorker(t, test.kind, func(_ context.Context, input rte.InvalidateInput) error {
				calls++
				if test.ingest && test.kind == rte.CalculationKindCreditBalance {
					require.Greater(t, time.Now().Unix(), test.storedAt.Unix())
				}
				return nil
			})

			// When the source handler reaches the single atomic invalidation.
			require.NoError(t, w.handleSource(msg, event))

			// Then only credit ingestion waits for stored usage to become visible.
			require.Equal(t, 1, calls)
		})
	}
}

func TestCanceledCreditIngestDoesNotInvalidate(t *testing.T) {
	// Given credit ingestion whose visibility wait exceeds the processing deadline.
	event := &ingestevents.EventBatchedIngest{Namespace: eventmodels.NamespaceID{ID: "ns"}, SubjectKey: "subject", StoredAt: time.Now().Add(time.Hour)}
	msg, err := marshaler.New(nil).Marshal(event)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
	defer cancel()
	msg.SetContext(ctx)
	w := sourceWorker(t, rte.CalculationKindCreditBalance, func(context.Context, rte.InvalidateInput) error {
		t.Fatal("canceled batch invalidated demand")
		return nil
	})

	// When the per-message context expires, retry still owns the source fact.
	require.ErrorIs(t, w.handleSource(msg, event), context.DeadlineExceeded)
}

func sourceWorker(t *testing.T, kind rte.CalculationKind, invalidate func(context.Context, rte.InvalidateInput) error) *worker {
	t.Helper()
	meter := noop.NewMeterProvider().Meter("rte-worker-test")
	count, err := meter.Int64Counter("events")
	require.NoError(t, err)
	lag, err := meter.Float64Histogram("lag")
	require.NoError(t, err)
	return &worker{sources: fixedSources{selectors: []rte.DemandSelector{{Namespace: "ns", Kind: kind}}}, demand: recordingDemand{invalidate: invalidate}, sourceEvents: count, sourceLag: lag}
}
