package engine

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type RunParams struct {
	// List of all grants that are active at the relevant period at some point.
	Grants []grant.Grant
	// Period to burn down the grants for.
	Period timeutil.Period
	// Starting snapshot of the balances at the START OF THE PERIOD.
	StartingSnapshot balance.Snapshot
}

type RunResult struct {
	// Snapshot of the balances at the END OF THE PERIOD.
	Snapshot balance.Snapshot
	// History of the grant burn down.
	History []GrantBurnDownHistorySegment
}

type Engine interface {
	// Burns down all grants in the defined period by the usage amounts.
	//
	// When the engine outputs a balance, it doesn't discriminate what should be in that balance.
	// If a grant is inactive at the end of the period, it will still be in the output.
	Run(ctx context.Context, params RunParams) (RunResult, error)
}

// TODO: should return alpacadecimal instead of float64, its fine to hard depend on it for now
type QueryUsageFn func(ctx context.Context, from, to time.Time) (float64, error)

type EngineConfig struct {
	Granularity meter.WindowSize
	QueryUsage  QueryUsageFn
}

func NewEngine(conf EngineConfig) Engine {
	return &engine{
		EngineConfig: conf,
	}
}

// engine burns down grants based on usage following the rules of Grant BurnDown.
type engine struct {
	EngineConfig

	// List of all grants that are active at the relevant period at some point.
	// Changes during execution, runtime state.
	grants []grant.Grant
}

// Ensure engine implements Engine
var _ Engine = (*engine)(nil)

func later(t1 time.Time, t2 time.Time) time.Time {
	if t1.After(t2) {
		return t1
	}
	return t2
}
