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

package commonhttp

import (
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

func GetSortOrder[TInput comparable](asc TInput, inp *TInput) sortx.Order {
	return defaultx.WithDefault(
		convert.SafeDeRef[TInput, sortx.Order](
			inp,
			func(o TInput) *sortx.Order {
				if o == asc {
					return convert.ToPointer(sortx.OrderAsc)
				}
				return convert.ToPointer(sortx.OrderDesc)
			},
		),
		sortx.OrderNone,
	)
}
