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

package entutils

import (
	"entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/pkg/sortx"
)

func GetOrdering(order sortx.Order) []sql.OrderTermOption {
	type o = sql.OrderTermOption

	switch order {
	case sortx.OrderAsc:
		return []o{sql.OrderAsc()}
	case sortx.OrderDesc:
		return []o{sql.OrderDesc()}
	default:
		return getStrOrdering(string(order))
	}
}

func getStrOrdering(order string) []sql.OrderTermOption {
	type o = sql.OrderTermOption

	switch order {
	case string(sortx.OrderAsc):
		return []o{sql.OrderAsc()}
	case string(sortx.OrderDesc):
		return []o{sql.OrderDesc()}
	default:
		return []o{}
	}
}
