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

// Map maps elements of a slice from T to M, returning a new slice.
func Map[T any, S any](s []T, f func(T) S) []S {
	// Nil input, return early.
	if s == nil {
		return nil
	}

	n := make([]S, len(s))

	for i, v := range s {
		n[i] = f(v)
	}

	return n
}
