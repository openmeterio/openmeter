package calculator

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/openmeterio/openmeter/openmeter/credit"
	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/rte"
	"github.com/openmeterio/openmeter/openmeter/rte/meteredentitlement"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

var _ rte.IncrementalCalculator[meteredentitlement.Key, meteredentitlement.Balance] = (*calculator)(nil)

type phaseUsageSource interface {
	GetEntitlementBalanceWithUsageQuery(context.Context, models.NamespacedID, time.Time, balance.PhaseUsageQuery) (*meteredentitlement.Balance, error)
}

type emptyUsageCheckpoint struct {
	Version int       `json:"version"`
	From    time.Time `json:"from"`
}

func (c *calculator) CalculateIncremental(ctx context.Context, input rte.CalculateInput[meteredentitlement.Key], candidates []rte.Checkpoint) (rte.IncrementalResult[meteredentitlement.Balance], error) {
	var result rte.IncrementalResult[meteredentitlement.Balance]
	source, ok := c.source.(phaseUsageSource)
	if !ok {
		value, err := c.Calculate(ctx, input)
		return rte.IncrementalResult[meteredentitlement.Balance]{Value: value}, err
	}

	if err := input.Validate(); err != nil {
		return result, err
	}

	windows := emptyUsageWindows{until: input.At.Truncate(time.Minute)}
	for _, checkpoint := range candidates {
		var saved emptyUsageCheckpoint
		if err := json.Unmarshal(checkpoint.Data, &saved); err != nil || saved.Version != 1 || saved.From.IsZero() ||
			!saved.From.Before(checkpoint.At) || checkpoint.At.After(windows.until) {
			continue
		}

		windows.candidates = append(windows.candidates, timeutil.ClosedPeriod{From: saved.From, To: checkpoint.At})
	}

	value, err := source.GetEntitlementBalanceWithUsageQuery(credit.WithoutBalanceSnapshotWrites(ctx), input.Key.EntitlementID, input.At, windows.usage)
	if err != nil {
		return result, err
	}

	result.Value = *value
	if windows.reused {
		result.Use = rte.CheckpointReused
	}

	result.Discarded = windows.discarded
	if !windows.best.From.IsZero() && !slices.ContainsFunc(windows.candidates, windows.best.Equal) {
		payload, err := json.Marshal(emptyUsageCheckpoint{Version: 1, From: windows.best.From})
		if err != nil {
			return result, err
		}

		result.Checkpoint = &rte.Checkpoint{At: windows.best.To, Data: payload}
	}

	return result, nil
}

// A fresh count of zero proves every subinterval is empty under today's
// attribution and filters. Grant and reset transitions still run for each phase.
type emptyUsageWindows struct {
	candidates  []timeutil.ClosedPeriod
	verified    timeutil.ClosedPeriod
	probed      bool
	unsupported bool
	reused      bool
	discarded   []time.Time
	rejected    timeutil.ClosedPeriod

	until      time.Time
	run, best  timeutil.ClosedPeriod
	runPhases  int
	bestPhases int
}

func (w *emptyUsageWindows) usage(phase timeutil.ClosedPeriod, read balance.PhaseUsageReader) (float64, error) {
	if !w.probed {
		w.checkPrefix(phase, read)
	}

	query := phase
	if !w.verified.From.IsZero() && !query.From.Before(w.verified.From) && query.From.Before(w.verified.To) {
		w.reused = true
		if !query.To.After(w.verified.To) {
			w.observeEmpty(phase)
			return 0, nil
		}

		query.From = w.verified.To
	}

	value, empty, err := read.Value(query)
	if err != nil {
		return 0, err
	}

	if empty && !w.unsupported {
		w.observeEmpty(phase)
	} else if !empty {
		// One ordinary phase with usage can rule out several overlapping hints.
		for _, previous := range w.candidates {
			if !query.From.Before(previous.From) && !query.To.After(previous.To) {
				w.discard(previous.To)
			}
		}
	}

	return value, nil
}

func (w *emptyUsageWindows) checkPrefix(phase timeutil.ClosedPeriod, read balance.PhaseUsageReader) {
	for _, candidate := range w.candidates {
		// If a single phase spans the whole hint, it already takes only one query.
		if phase.From.Before(candidate.From) || !phase.To.Before(candidate.To) {
			continue
		}

		prefix := timeutil.ClosedPeriod{From: phase.From, To: candidate.To}
		w.probed = true
		hasEvents, err := read.HasEvents(prefix)
		if err != nil && !errors.Is(err, errors.ErrUnsupported) {
			// A transient error says nothing about the hint; use ordinary reads.
			return
		}

		switch {
		case err != nil:
			w.unsupported = true
			for _, cp := range w.candidates {
				w.discard(cp.To)
			}
		case hasEvents:
			w.rejected = prefix
			w.discard(candidate.To)
		default:
			w.verified = prefix
		}

		return
	}
}

func (w *emptyUsageWindows) discard(at time.Time) {
	if !slices.Contains(w.discarded, at) {
		w.discarded = append(w.discarded, at)
	}
}

func (w *emptyUsageWindows) observeEmpty(phase timeutil.ClosedPeriod) {
	if phase.To.After(w.until) {
		phase.To = w.until
	}

	if !phase.From.Before(phase.To) || (phase.From.Before(w.rejected.To) && phase.To.After(w.rejected.From)) {
		return
	}

	if w.run.To.Equal(phase.From) {
		w.run.To = phase.To
		w.runPhases++
	} else {
		w.run = phase
		w.runPhases = 1
	}

	// One scalar query already covers a single phase. Prefer the range saving
	// the most round trips, and the newer one on ties.
	if w.runPhases >= 2 && w.runPhases >= w.bestPhases {
		w.best = w.run
		w.bestPhases = w.runPhases
	}
}
