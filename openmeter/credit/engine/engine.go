package engine

import (
	"context"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type RunParams struct {
	// Meter for the current run.
	Meter meter.Meter
	// List of all grants that are active at the relevant period at some point.
	Grants []grant.Grant
	// End of the period to burn down the grants for.
	Until time.Time
	// Starting snapshot of the balances at the START OF THE PERIOD.
	StartingSnapshot balance.Snapshot
	// ResetBehavior defines the behavior of the engine when a reset is encountered.
	ResetBehavior grant.ResetBehavior
	// Timeline of the resets that occurred in the period.
	// The resets must occur AFTER the starting snapshot and NOT AFTER the until time. (exclusive - inclusive)
	Resets timeutil.SimpleTimeline
}

func (p RunParams) Clone() RunParams {
	grants := make([]grant.Grant, len(p.Grants))
	copy(grants, p.Grants)

	resets := timeutil.NewSimpleTimeline(p.Resets.GetTimes())

	res := RunParams{
		Meter:            p.Meter,
		Grants:           grants,
		Until:            p.Until,
		StartingSnapshot: p.StartingSnapshot,
		ResetBehavior:    p.ResetBehavior,
		Resets:           resets,
	}

	return res
}

type RunResult struct {
	// Snapshot of the balances at the END OF THE PERIOD.
	Snapshot balance.Snapshot
	// History of the grant burn down.
	History GrantBurnDownHistory
	// RunParams used to produce the result.
	RunParams RunParams
}

// TotalAvailableGrantAmount is the total amount of grants either currently active + the used amount of currently inactive grants.
func (r RunResult) TotalAvailableGrantAmount() float64 {
	// First, let's calculate the total amount of grants active at the end of the period.
	activeAmount := lo.Reduce(r.RunParams.Grants, func(agg alpacadecimal.Decimal, grant grant.Grant, _ int) alpacadecimal.Decimal {
		if !grant.ActiveAt(r.RunParams.Until) {
			return agg
		}

		return agg.Add(alpacadecimal.NewFromFloat(grant.Amount))
	}, alpacadecimal.NewFromFloat(0))

	// Second, let's calculate the used-up amount of since inactive grants.
	inactiveGrants := lo.Filter(r.RunParams.Grants, func(grant grant.Grant, _ int) bool {
		return !grant.ActiveAt(r.RunParams.Until)
	})

	usedInactive := alpacadecimal.NewFromFloat(0)
	if len(inactiveGrants) > 0 {
		for _, seg := range r.History.Segments() {
			for _, usage := range seg.GrantUsages {
				if lo.SomeBy(inactiveGrants, func(grant grant.Grant) bool {
					return grant.ID == usage.GrantID
				}) {
					usedInactive = usedInactive.Add(alpacadecimal.NewFromFloat(usage.Usage))
				}
			}
		}
	}

	return activeAmount.Add(usedInactive).InexactFloat64()
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
	QueryUsage QueryUsageFn
}

func NewEngine(conf EngineConfig) Engine {
	return &engine{
		EngineConfig: conf,
	}
}

// engine burns down grants based on usage following the rules of Grant BurnDown.
type engine struct {
	EngineConfig
}

// Ensure engine implements Engine
var _ Engine = (*engine)(nil)
