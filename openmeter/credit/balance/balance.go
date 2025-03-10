package balance

import (
	"time"

	"github.com/openmeterio/openmeter/openmeter/credit/grant"
)

func NewStartingMap(grants []grant.Grant, at time.Time) Map {
	balances := make(Map)
	for _, grant := range grants {
		if grant.ActiveAt(at) {
			balances.Set(grant.ID, grant.Amount)
		} else {
			balances.Set(grant.ID, 0.0)
		}
	}
	return balances
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

type SnapshottedUsage struct {
	Usage float64   `json:"usage"`
	Since time.Time `json:"since"`
}

func (s SnapshottedUsage) IsZero() bool {
	return s.Usage == 0.0 && s.Since.IsZero()
}

type Snapshot struct {
	Usage    SnapshottedUsage
	Balances Map
	Overage  float64
	At       time.Time
}

func (g Snapshot) Balance() float64 {
	return g.Balances.Balance()
}
