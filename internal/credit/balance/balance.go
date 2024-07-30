package balance

import (
	"time"

	"github.com/openmeterio/openmeter/internal/credit/grant"
)

// Represents a point in time balance of grants
type Map map[string]float64

func (g Map) Copy() Map {
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

func (g Map) OverrideWith(gbm Map) {
	for k, v := range gbm {
		g[k] = v
	}
}

type Snapshot struct {
	Balances Map
	Overage  float64
	At       time.Time
}

func (g Snapshot) Balance() float64 {
	return g.Balances.Balance()
}
