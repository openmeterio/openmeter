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

package notification

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChannelIDsDifference(t *testing.T) {
	tests := []struct {
		Name string

		New []string
		Old []string

		ExpectedAdditions []string
		ExpectedRemovals  []string
	}{
		{
			Name: "No change",
			New: []string{
				"channel-1",
				"channel-2",
			},
			Old: []string{
				"channel-2",
				"channel-1",
			},
		},
		{
			Name: "Add new channel",
			New: []string{
				"channel-1",
				"channel-2",
				"channel-3",
			},
			Old: []string{
				"channel-2",
				"channel-1",
			},
			ExpectedAdditions: []string{
				"channel-3",
			},
		},
		{
			Name: "Remove old channel",
			New: []string{
				"channel-2",
			},
			Old: []string{
				"channel-2",
				"channel-1",
			},
			ExpectedRemovals: []string{
				"channel-1",
			},
		},
		{
			Name: "Add and remove channels",
			New: []string{
				"channel-1",
				"channel-3",
			},
			Old: []string{
				"channel-2",
				"channel-1",
			},
			ExpectedAdditions: []string{
				"channel-3",
			},
			ExpectedRemovals: []string{
				"channel-2",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			diff := NewChannelIDsDifference(test.New, test.Old)

			assert.ElementsMatch(t, test.ExpectedAdditions, diff.Additions())
			assert.ElementsMatch(t, test.ExpectedRemovals, diff.Removals())
		})
	}
}
