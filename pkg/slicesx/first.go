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

// returns the first element in the slice where the predicate returns true
// if second argument is true the returns the last not the first
func First[T any](s []T, f func(T) bool, last bool) (*T, bool) {
	if s == nil {
		return nil, false
	}

	if last {
		for i := len(s) - 1; i >= 0; i-- {
			if f(s[i]) {
				return &s[i], true
			}
		}
		return nil, false
	}

	for _, v := range s {
		if f(v) {
			return &v, true
		}
	}

	return nil, false
}
