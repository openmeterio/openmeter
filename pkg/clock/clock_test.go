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

package clock_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
)

func TestClock(t *testing.T) {
	clock.SetTime(testutils.GetRFC3339Time(t, "2024-06-30T15:39:00Z"))
	defer clock.ResetTime()

	now := clock.Now()
	diff := now.Sub(testutils.GetRFC3339Time(t, "2024-06-30T15:39:00Z"))
	if diff < 0 {
		diff = -diff
	}
	assert.True(t, diff < time.Second)
}
