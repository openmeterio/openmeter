package balance

import (
	"time"

	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/unitconfig"
)

// NewStartingSnapshot returns the complete zero-usage snapshot from which
// measurement starts.
func NewStartingSnapshot(grants []grant.Grant, at time.Time) Snapshot {
	balances := make(Map)
	for _, grant := range grants {
		if grant.ActiveAt(at) {
			balances.Set(grant.ID, grant.Amount)
		} else {
			balances.Set(grant.ID, 0.0)
		}
	}

	return Snapshot{
		Usage: SnapshottedUsage{
			Since: at,
			Usage: 0,
		},
		UsageSnapshot: &UsageSnapshot{
			Usage:           0,
			TotalGrantUsage: 0,
		},
		Balances: balances,
		Overage:  0,
		At:       at,
	}
}

// Represents a point in time balance of grants
type Map map[string]float64

func (g Map) Clone() Map {
	r := make(Map, len(g))
	for k, v := range g {
		r[k] = v
	}
	return r
}

func (g Map) Burn(grantID string, amount float64) {
	balance := g[grantID]
	g[grantID] = balance - amount
}

func (g Map) Set(grantID string, amount float64) {
	g[grantID] = amount
}

// returns the combined balance of all grants
func (g Map) Balance() float64 {
	var balance float64
	for _, v := range g {
		balance += v
	}
	return balance
}

// Whether the contents of the Map exactly matches
// the list of provided grants.
// Return false if it has additional grants or if it misses any grants
func (g Map) ExactlyForGrants(grants []grant.Grant) bool {
	gmap := map[string]struct{}{}
	for _, grant := range grants {
		gmap[grant.ID] = struct{}{}
	}

	if len(gmap) != len(g) {
		return false
	}

	for k := range gmap {
		if _, ok := g[k]; !ok {
			return false
		}
	}
	return true
}

// SnapshottedUsage is the legacy usage representation whose value is relative
// to an explicitly stored starting timestamp.
//
// Deprecated: use Snapshot.UsageSnapshot for complete usage-period state.
type SnapshottedUsage struct {
	Usage float64   `json:"usage"`
	Since time.Time `json:"since"`
}

// IsZero reports whether the legacy usage representation is unset.
//
// Deprecated: only use this while reading legacy snapshots.
func (s SnapshottedUsage) IsZero() bool {
	return s.Usage == 0.0 && s.Since.IsZero()
}

// UsageSnapshot is the cumulative usage state for the usage period containing
// the snapshot.
type UsageSnapshot struct {
	Usage           float64 `json:"usage"`
	TotalGrantUsage float64 `json:"totalGrantUsage"`
}

type Snapshot struct {
	// Usage is retained for compatibility with the legacy persistence shape.
	//
	// Deprecated: use UsageSnapshot for engine calculations.
	Usage SnapshottedUsage
	// UsageSnapshot is nil for snapshots created without complete usage-period
	// state.
	UsageSnapshot *UsageSnapshot

	Balances Map
	Overage  float64
	At       time.Time

	// UnitConfig records the conversion regime this snapshot's Usage/Balances were
	// computed under (OM-400). The resume path only reuses a snapshot whose UnitConfig
	// matches the owner's current one; a mismatch (e.g. a future backfill that sets an
	// entitlement's unit_config after raw-unit snapshots already exist) forces a
	// recompute rather than mixing raw and converted units. Nil = raw (no conversion).
	UnitConfig *unitconfig.UnitConfig
}

// Clone returns a snapshot whose mutable balances and unit configuration are
// independent from the source.
func (s Snapshot) Clone() Snapshot {
	cloned := s
	cloned.Balances = s.Balances.Clone()
	if s.UsageSnapshot != nil {
		usageSnapshot := *s.UsageSnapshot
		cloned.UsageSnapshot = &usageSnapshot
	}
	if s.UnitConfig != nil {
		unitConfig := s.UnitConfig.Clone()
		cloned.UnitConfig = &unitConfig
	}

	return cloned
}

func (g Snapshot) Balance() float64 {
	return g.Balances.Balance()
}
