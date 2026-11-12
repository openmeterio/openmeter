package worker

import (
	"context"
	"testing"
	"testing/synctest"
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

func TestIngestInvalidatesCreditDemandRegisteredDuringVisibilityWait(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// Given an ingest notification for a customer with no demand yet.
		sources := &fixedSources{}
		var invalidations []rte.InvalidateInput
		w := sourceWorker(t, sources, func(_ context.Context, input rte.InvalidateInput) error {
			invalidations = append(invalidations, input)
			return nil
		})

		event := &ingestevents.EventBatchedIngest{Namespace: eventmodels.NamespaceID{ID: "ns"}, SubjectKey: "subject", StoredAt: time.Now()}
		msg, err := marshaler.New(nil).Marshal(event)
		require.NoError(t, err)
		msg.SetContext(t.Context())

		// When a first pull registers before credit rating can see the batch.
		done := make(chan error, 1)
		go func() { done <- w.handleSource(msg, event) }()
		synctest.Wait()
		sources.selectors = []rte.DemandSelector{{Namespace: "ns", Kind: rte.CalculationKindCreditBalance, OwnerIDs: []string{"customer"}}}

		// Then the worker includes the new demand when the storage second closes.
		require.NoError(t, <-done)
		require.Len(t, invalidations, 1)
		require.Equal(t, sources.selectors, invalidations[0].Selectors)
		require.Greater(t, time.Now().Unix(), event.StoredAt.Unix())
	})
}

func TestLedgerFactInvalidatesDemandImmediately(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		event := &ledger.TransactionGroupCommittedEvent{Namespace: "ns", GroupID: "group"}
		msg, err := marshaler.New(nil).Marshal(event)
		require.NoError(t, err)
		msg.SetContext(t.Context())

		sources := fixedSources{selectors: []rte.DemandSelector{{Namespace: "ns", Kind: rte.CalculationKindCreditBalance}}}
		var invalidations []rte.InvalidateInput
		w := sourceWorker(t, sources, func(_ context.Context, input rte.InvalidateInput) error {
			invalidations = append(invalidations, input)
			return nil
		})

		start := time.Now()
		require.NoError(t, w.handleSource(msg, event))
		require.Equal(t, start, time.Now())
		require.Len(t, invalidations, 1)
		require.Equal(t, sources.selectors, invalidations[0].Selectors)
	})
}

func TestCanceledCreditIngestDoesNotInvalidate(t *testing.T) {
	// Given credit ingestion whose visibility wait exceeds the processing deadline.
	event := &ingestevents.EventBatchedIngest{Namespace: eventmodels.NamespaceID{ID: "ns"}, SubjectKey: "subject", StoredAt: time.Now().Add(time.Hour)}
	msg, err := marshaler.New(nil).Marshal(event)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
	defer cancel()
	msg.SetContext(ctx)
	w := sourceWorker(t, &fixedSources{}, func(context.Context, rte.InvalidateInput) error {
		t.Fatal("canceled batch invalidated demand")
		return nil
	})

	// When the per-message context expires, retry still owns the source fact.
	require.ErrorIs(t, w.handleSource(msg, event), context.DeadlineExceeded)
}

func sourceWorker(t *testing.T, sources rte.UpstreamResolver, invalidate func(context.Context, rte.InvalidateInput) error) *worker {
	t.Helper()
	meter := noop.NewMeterProvider().Meter("rte-worker-test")
	count, err := meter.Int64Counter("events")
	require.NoError(t, err)
	lag, err := meter.Float64Histogram("lag")
	require.NoError(t, err)

	return &worker{sources: sources, demand: recordingDemand{invalidate: invalidate}, sourceEvents: count, sourceLag: lag}
}
