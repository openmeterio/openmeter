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

package entitlement_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/recurrence"
)

func TestUsagePeriod(t *testing.T) {
	t1 := time.Now().Truncate(time.Minute)

	t.Run("Should be inclusive on period From and exclusive on period To", func(t *testing.T) {
		up := entitlement.UsagePeriod{
			Interval: recurrence.RecurrencePeriodDaily,
			Anchor:   t1,
		}

		period, err := up.GetCurrentPeriodAt(t1)
		assert.NoError(t, err)
		assert.Equal(t, t1, period.From)
		assert.Equal(t, t1.AddDate(0, 0, 1), period.To)
	})
}

func TestMeasureUsageFromInput(t *testing.T) {
	t.Run("Should return time from input", func(t *testing.T) {
		t1 := time.Now().Truncate(time.Minute)
		m := &entitlement.MeasureUsageFromInput{}
		err := m.FromTime(t1)
		assert.NoError(t, err)
		assert.Equal(t, t1, m.Get())
	})

	t.Run("Should return time from CURRENT_PERIOD_START enum", func(t *testing.T) {
		t0 := time.Now().Truncate(time.Minute)
		t1 := t0.Add(-time.Hour)
		up := entitlement.UsagePeriod{
			Interval: recurrence.RecurrencePeriodDaily,
			Anchor:   t1,
		}

		m := &entitlement.MeasureUsageFromInput{}
		err := m.FromEnum(entitlement.MeasureUsageFromCurrentPeriodStart, up, t0)
		assert.NoError(t, err)
		assert.Equal(t, t1, m.Get())
	})

	t.Run("Should return time from CREATED_AT enum", func(t *testing.T) {
		t0 := time.Now().Truncate(time.Minute)
		t1 := t0.Add(-time.Hour)
		up := entitlement.UsagePeriod{
			Interval: recurrence.RecurrencePeriodDaily,
			Anchor:   t1,
		}

		m := &entitlement.MeasureUsageFromInput{}
		err := m.FromEnum(entitlement.MeasureUsageFromNow, up, t0)
		assert.NoError(t, err)
		assert.Equal(t, t0, m.Get())
	})
}
