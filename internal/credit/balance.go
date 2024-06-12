package credit

// Represents a point in time balance of grants
type GrantBalanceMap map[GrantID]float64

func (g GrantBalanceMap) Copy() GrantBalanceMap {
	r := make(GrantBalanceMap, len(g))
	for k, v := range g {
		r[k] = v
	}
	return r
}

func (g GrantBalanceMap) Burn(grantID GrantID, amount float64) {
	balance := g[grantID]
	g[grantID] = balance - amount
}

func (g GrantBalanceMap) Set(grantID GrantID, amount float64) {
	g[grantID] = amount
}
