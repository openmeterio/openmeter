package credit

import "time"

// Represents a point in time balance of grants
type GrantBalanceMap map[string]float64

func (g GrantBalanceMap) Copy() GrantBalanceMap {
	r := make(GrantBalanceMap, len(g))
	for k, v := range g {
		r[k] = v
	}
	return r
}

func (g GrantBalanceMap) Burn(grantID string, amount float64) {
	balance := g[grantID]
	g[grantID] = balance - amount
}

func (g GrantBalanceMap) Set(grantID string, amount float64) {
	g[grantID] = amount
}

// returns the combined balance of all grants
func (g GrantBalanceMap) Balance() float64 {
	var balance float64
	for _, v := range g {
		balance += v
	}
	return balance
}

func NewGrantBalanceMapFromStartingGrants(grants []Grant) GrantBalanceMap {
	m := GrantBalanceMap{}
	for _, g := range grants {
		m[g.ID] = g.Amount
	}
	return m
}

type GrantBalanceSnapshot struct {
	Balances GrantBalanceMap
	Overage  float64
	At       time.Time
}

func (g GrantBalanceSnapshot) Balance() float64 {
	return g.Balances.Balance()
}
