// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

// Whether the contents of the GrantBalanceMap exactly matches
// the list of provided grants.
// Return false if it has additional grants or if it misses any grants
func (g GrantBalanceMap) ExactlyForGrants(grants []Grant) bool {
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

func (g GrantBalanceMap) OverrideWith(gbm GrantBalanceMap) {
	for k, v := range gbm {
		g[k] = v
	}
}

type GrantBalanceSnapshot struct {
	Balances GrantBalanceMap
	Overage  float64
	At       time.Time
}

func (g GrantBalanceSnapshot) Balance() float64 {
	return g.Balances.Balance()
}
