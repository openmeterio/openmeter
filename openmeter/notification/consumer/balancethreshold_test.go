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

package consumer

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/entitlement/snapshot"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/convert"
)

func newNumericThreshold(v float64) notification.BalanceThreshold {
	return notification.BalanceThreshold{
		Value: v,
		Type:  api.NUMBER,
	}
}

func newPercentThreshold(v float64) notification.BalanceThreshold {
	return notification.BalanceThreshold{
		Value: v,
		Type:  api.PERCENT,
	}
}

func TestGetHighestMatchingBalanceThreshold(t *testing.T) {
	tcs := []struct {
		Name              string
		BalanceThresholds []notification.BalanceThreshold
		EntitlementValue  snapshot.EntitlementValue
		Expect            *notification.BalanceThreshold
	}{
		{
			Name: "Numerical values only",
			BalanceThresholds: []notification.BalanceThreshold{
				newNumericThreshold(20),
				newNumericThreshold(10),
				newNumericThreshold(30),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance: convert.ToPointer(10.0),
				Usage:   convert.ToPointer(20.0),
			},
			// Already used 20, so the matching threshold is the 20
			Expect: convert.ToPointer(newNumericThreshold(20)),
		},
		{
			Name: "Numerical values only - 100%",
			BalanceThresholds: []notification.BalanceThreshold{
				newNumericThreshold(20),
				newNumericThreshold(10),
				newNumericThreshold(30),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance: convert.ToPointer(0.0),
				Usage:   convert.ToPointer(30.0),
			},
			Expect: convert.ToPointer(newNumericThreshold(30)),
		},
		{
			Name: "Numerical values only - 100%+ with overage",
			BalanceThresholds: []notification.BalanceThreshold{
				newNumericThreshold(20),
				newNumericThreshold(10),
				newNumericThreshold(30),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance: convert.ToPointer(0.0),
				Usage:   convert.ToPointer(30.0),
				Overage: convert.ToPointer(10.0),
			},
			Expect: convert.ToPointer(newNumericThreshold(30)),
		},
		{
			Name: "Percentages with overage",
			BalanceThresholds: []notification.BalanceThreshold{
				newPercentThreshold(50),
				newPercentThreshold(100),
				newPercentThreshold(110),
				newPercentThreshold(120),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance: convert.ToPointer(0.0),
				Usage:   convert.ToPointer(110.0),
				Overage: convert.ToPointer(10.0),
			},
			Expect: convert.ToPointer(newPercentThreshold(110)),
		},
		{
			Name: "Mixed values",
			BalanceThresholds: []notification.BalanceThreshold{
				newNumericThreshold(20),
				newNumericThreshold(10),
				newNumericThreshold(30),
				newPercentThreshold(50),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance: convert.ToPointer(14.0),
				Usage:   convert.ToPointer(16.0),
			},
			Expect: convert.ToPointer(newPercentThreshold(50)),
		},
		// Corner cases
		{
			Name: "No grants",
			BalanceThresholds: []notification.BalanceThreshold{
				newNumericThreshold(20),
				newPercentThreshold(100),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance: convert.ToPointer(0.0),
				Usage:   convert.ToPointer(0.0),
			},
			Expect: nil,
		},
		{
			Name: "Last threshold is ",
			BalanceThresholds: []notification.BalanceThreshold{
				newNumericThreshold(20),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance: convert.ToPointer(0.0),
				Usage:   convert.ToPointer(30.0),
			},
			Expect: convert.ToPointer(newNumericThreshold(20)),
		},
		{
			Name: "Same threshold in percentage and number",
			BalanceThresholds: []notification.BalanceThreshold{
				newNumericThreshold(15),
				newPercentThreshold(50),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance: convert.ToPointer(14.0),
				Usage:   convert.ToPointer(16.0),
			},
			Expect: convert.ToPointer(newPercentThreshold(50)),
		},
		{
			Name: "Exact threshold match",
			BalanceThresholds: []notification.BalanceThreshold{
				newNumericThreshold(15),
				newPercentThreshold(50),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance: convert.ToPointer(15.0),
				Usage:   convert.ToPointer(15.0),
			},
			Expect: convert.ToPointer(newPercentThreshold(50)),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Name, func(t *testing.T) {
			got, err := getHighestMatchingThreshold(tc.BalanceThresholds, tc.EntitlementValue)
			assert.NoError(t, err)
			assert.Equal(t, tc.Expect, got)
		})
	}
}
