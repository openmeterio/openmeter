package worker

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/rte"
	"github.com/openmeterio/openmeter/openmeter/testutils"
)

type pollingDemand struct {
	rte.DemandService
	process func(context.Context) (bool, error)
}

func (d pollingDemand) Process(ctx context.Context) (bool, error) {
	return d.process(ctx)
}

func TestDemandWorkerDrainsDeferredFailures(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		// Given a durably deferred calculation followed by a repository outage.
		var attempts []time.Time
		process := func(context.Context) (bool, error) {
			attempts = append(attempts, time.Now())
			switch len(attempts) {
			case 1:
				return true, errors.New("calculation deferred")
			case 2:
				return false, errors.New("cannot claim work")
			default:
				cancel()
				return false, nil
			}
		}

		w := worker{
			logger:                 testutils.NewDiscardLogger(t),
			pollInterval:           time.Second,
			calculationConcurrency: 1,
			demand:                 pollingDemand{process: process},
		}

		// When the worker drains the queue, it only waits after the failed claim.
		w.pollDemands(ctx)

		require.Len(t, attempts, 3)
		require.Equal(t, attempts[0], attempts[1])
		require.Equal(t, time.Second, attempts[2].Sub(attempts[1]))
	})
}

func TestDemandWorkerConcurrencyAndShutdown(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// Given three slots and enough work to keep every slot busy.
		const concurrency = 3
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		release := make(chan struct{})
		cleanup := make(chan struct{})
		finishCleanup := sync.OnceFunc(func() { close(cleanup) })
		defer finishCleanup()
		var active, calls atomic.Int32

		w := worker{
			logger:                 testutils.NewDiscardLogger(t),
			pollInterval:           time.Hour,
			calculationConcurrency: concurrency,
			demand: pollingDemand{process: func(ctx context.Context) (bool, error) {
				active.Add(1)
				calls.Add(1)
				defer active.Add(-1)

				select {
				case <-release:
					return true, nil
				case <-ctx.Done():
					// Model an attempt finishing rollback after cancellation.
					<-cleanup
					return false, nil
				}
			}},
		}

		done := make(chan struct{})
		go func() {
			w.pollDemands(ctx)
			close(done)
		}()
		synctest.Wait()

		require.Equal(t, int32(concurrency), active.Load())
		require.Equal(t, int32(concurrency), calls.Load())

		// When one attempt finishes, that slot claims more work immediately.
		release <- struct{}{}
		synctest.Wait()

		require.Equal(t, int32(concurrency), active.Load())
		require.Equal(t, int32(concurrency+1), calls.Load())

		// Then shutdown cancels every slot and waits for in-flight work to finish.
		cancel()
		synctest.Wait()
		select {
		case <-done:
			t.Fatal("pool stopped before attempts finished")
		default:
		}

		finishCleanup()
		synctest.Wait()
		select {
		case <-done:
		default:
			t.Fatal("pool did not stop after attempts finished")
		}
		require.Zero(t, active.Load())
		require.Equal(t, int32(concurrency+1), calls.Load())
	})
}

func TestDemandWorkerWaitsOnlyWhenIdle(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// Given work followed by an empty queue, with a long calculation first.
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		var attempts []time.Time

		w := worker{
			logger:                 testutils.NewDiscardLogger(t),
			calculationConcurrency: 1,
			pollInterval:           time.Second,
			demand: pollingDemand{process: func(context.Context) (bool, error) {
				attempts = append(attempts, time.Now())
				switch len(attempts) {
				case 1:
					time.Sleep(5 * time.Second)
					return true, nil
				case 2:
					return false, nil
				default:
					cancel()
					return false, nil
				}
			}},
		}
		w.pollDemands(ctx)

		// Then work drains immediately and the empty attempt starts a full idle wait.
		require.Len(t, attempts, 3)
		require.Equal(t, 5*time.Second, attempts[1].Sub(attempts[0]))
		require.Equal(t, time.Second, attempts[2].Sub(attempts[1]))
	})
}

func TestDemandWorkerCanceledBeforeStart(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	var calls atomic.Int32

	w := worker{
		logger:                 testutils.NewDiscardLogger(t),
		calculationConcurrency: 3,
		pollInterval:           time.Hour,
		demand: pollingDemand{process: func(context.Context) (bool, error) {
			calls.Add(1)
			return false, nil
		}},
	}
	w.pollDemands(ctx)
	require.Zero(t, calls.Load())
}

func TestDemandWorkerCancellationInterruptsIdleWait(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		var calls atomic.Int32
		done := make(chan struct{})

		// Given workers waiting on an empty source.
		go func() {
			w := worker{
				logger:                 testutils.NewDiscardLogger(t),
				calculationConcurrency: 2,
				pollInterval:           time.Hour,
				demand: pollingDemand{process: func(context.Context) (bool, error) {
					calls.Add(1)
					return false, nil
				}},
			}
			w.pollDemands(ctx)
			close(done)
		}()
		synctest.Wait()
		require.Equal(t, int32(2), calls.Load())

		// When shutdown starts, then idle waits end without another attempt.
		cancel()
		synctest.Wait()
		select {
		case <-done:
		default:
			t.Fatal("idle workers did not stop")
		}
		require.Equal(t, int32(2), calls.Load())
	})
}
