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

import "testing"

func TestChunk(t *testing.T) {
	tests := []struct {
		name string
		s    []int
		size int
		want [][]int
	}{
		{
			name: "empty slice",
			s:    []int{},
			size: 2,
			want: [][]int{},
		},
		{
			name: "nil slice",
			s:    nil,
			size: 2,
			want: nil,
		},
		{
			name: "size is zero",
			s:    []int{1, 2, 3},
			size: 0,
			want: [][]int{{1, 2, 3}},
		},
		{
			name: "size is greater than the slice length",
			s:    []int{1, 2, 3},
			size: 4,
			want: [][]int{{1, 2, 3}},
		},
		{
			name: "size is less than the slice length",
			s:    []int{1, 2, 3, 4, 5},
			size: 2,
			want: [][]int{{1, 2}, {3, 4}, {5}},
		},
		{
			name: "size is equal to the slice length",
			s:    []int{1, 2, 3},
			size: 3,
			want: [][]int{{1, 2, 3}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Chunk(tt.s, tt.size); !equal(got, tt.want) {
				t.Errorf("Chunk() = %v, want %v", got, tt.want)
			}
		})
	}
}

func equal(a, b [][]int) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if len(a[i]) != len(b[i]) {
			return false
		}

		for j := range a[i] {
			if a[i][j] != b[i][j] {
				return false
			}
		}
	}

	return true
}
