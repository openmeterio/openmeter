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

package convert

import "time"

func ToPointer[T any](value T) *T {
	return &value
}

func MapToPointer[T comparable, U any](value map[T]U) *map[T]U {
	if len(value) == 0 {
		return nil
	}
	return &value
}

func ToStringLike[Source, Dest ~string](value *Source) *Dest {
	if value == nil {
		return nil
	}
	return ToPointer(Dest(*value))
}

func SafeConvert[T any, U any](value *T, fn func(T) U) *U {
	if value == nil {
		return nil
	}
	return ToPointer(fn(*value))
}

func SafeToUTC(t *time.Time) *time.Time {
	return SafeConvert(t, func(dt time.Time) time.Time {
		return dt.In(time.UTC)
	})
}
