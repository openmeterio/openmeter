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

package slicesx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDiff(t *testing.T) {
	tests := []struct {
		Name string

		Base []string
		New  []string

		ExpectedAdditions []string
		ExpectedRemovals  []string
	}{
		{
			Name: "No change",
			Base: []string{
				"diff-1",
				"diff-2",
			},
			New: []string{
				"diff-2",
				"diff-1",
			},
		},
		{
			Name: "Add new item",
			Base: []string{
				"diff-1",
				"diff-2",
				"diff-3",
			},
			New: []string{
				"diff-2",
				"diff-1",
			},
			ExpectedAdditions: []string{
				"diff-3",
			},
		},
		{
			Name: "Remove old item",
			Base: []string{
				"diff-2",
			},
			New: []string{
				"diff-2",
				"diff-1",
			},
			ExpectedRemovals: []string{
				"diff-1",
			},
		},
		{
			Name: "Add and remove items",
			Base: []string{
				"diff-1",
				"diff-3",
			},
			New: []string{
				"diff-2",
				"diff-1",
			},
			ExpectedAdditions: []string{
				"diff-3",
			},
			ExpectedRemovals: []string{
				"diff-2",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			diff := NewDiff(test.Base, test.New)

			assert.ElementsMatch(t, test.ExpectedAdditions, diff.Additions())
			assert.ElementsMatch(t, test.ExpectedRemovals, diff.Removals())
		})
	}
}
