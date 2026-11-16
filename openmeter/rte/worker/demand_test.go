package worker

import (
	"context"
	"errors"
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
			logger:       testutils.NewDiscardLogger(t),
			pollInterval: time.Second,
			demand:       pollingDemand{process: process},
		}

		// When the worker drains the queue, it only waits after the failed claim.
		w.processDemand(ctx)

		require.Len(t, attempts, 3)
		require.Equal(t, attempts[0], attempts[1])
		require.Equal(t, time.Second, attempts[2].Sub(attempts[1]))
	})
}
