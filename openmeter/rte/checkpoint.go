package rte

import (
	"bytes"
	"context"
	"encoding/json"
	"slices"
	"time"
)

// Checkpoint holds calculator-private inputs at a domain-selected boundary.
// Data includes the information the calculator needs to verify it against the source.
type Checkpoint struct {
	At   time.Time       `json:"at"`
	Data json.RawMessage `json:"data"`
}

type Checkpoints []Checkpoint

// A large balance must still be calculable when its working data is too large to cache.
const (
	maxCheckpoints    = 4
	maxCheckpointSize = 64 << 10
)

// Candidates returns the newest eligible boundaries first. A retry or clock change
// may place the requested time before the newest stored boundary.
func (history Checkpoints) Candidates(at time.Time) Checkpoints {
	var candidates Checkpoints
	for _, cp := range history {
		if cp.At.IsZero() || cp.At.After(at) {
			continue
		}

		// PostgreSQL adds whitespace when returning jsonb. Apply the admission
		// limit to the same compact payload that was accepted on write.
		if len(cp.Data) > maxCheckpointSize {
			var compact bytes.Buffer
			if err := json.Compact(&compact, cp.Data); err != nil || compact.Len() > maxCheckpointSize {
				continue
			}

			cp.Data = compact.Bytes()
		} else if !json.Valid(cp.Data) {
			continue
		}

		candidates = append(candidates, cp)
	}

	slices.SortStableFunc(candidates, func(a, b Checkpoint) int { return b.At.Compare(a.At) })
	candidates = slices.CompactFunc(candidates, func(a, b Checkpoint) bool { return a.At.Equal(b.At) })

	return candidates[:min(len(candidates), maxCheckpoints)]
}

// IncrementalCalculator can reuse part of an earlier calculation after verifying
// its inputs. Each candidate is independent. A missing or incompatible checkpoint
// must still produce the same value as Calculate.
type IncrementalCalculator[K Key, V any] interface {
	Calculator[K, V]
	CalculateIncremental(context.Context, CalculateInput[K], []Checkpoint) (IncrementalResult[V], error)
}

type CheckpointUse uint8

const (
	CheckpointUnused CheckpointUse = iota
	CheckpointReused
	CheckpointAdvanced
)

type IncrementalResult[V any] struct {
	Value V

	// Checkpoint is optional. Reusing an existing boundary need not create another one.
	Checkpoint *Checkpoint

	// Discarded identifies boundaries disproved by the source. A later refresh
	// can then try another candidate without paying for the same failed probe.
	Discarded []time.Time

	// Use distinguishes reuse of an unchanged input from advancing an opening
	// balance through later source movements.
	Use CheckpointUse
}
