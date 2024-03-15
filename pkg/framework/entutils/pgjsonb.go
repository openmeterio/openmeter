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

	"github.com/openmeterio/openmeter/pkg/slicesx"
)

// JSONBIn returns a function that filters the given JSONB field by the given key and value
// Caveats:
// - PostgreSQL only
// - The field must be a JSONB field
// - The value must be a string (no support for other types, ->> converts all values to string)
// - This might not work if there's a join involved in the query, so add unit tests
func JSONBIn(field string, key string, values []string) func(*sql.Selector) {
	return func(s *sql.Selector) {
		// This is just a safeguard, it should never happen, but if it's not in place, then if
		// len(values) == 0, then generated SQL query will be field->>'key' IN (), which is invalid in SQL
		if len(values) == 0 {
			s.Where(sql.P(func(b *sql.Builder) {
				b.WriteString("false")
			}))
			return
		}
		s.Where(sql.P(func(b *sql.Builder) {
			b.WriteString("(")
			b.WriteString(field)
			b.WriteString("->>'")
			b.WriteString(key)
			b.WriteString("' IN (")
			b.Args(slicesx.Map(values, func(f string) any {
				return f
			})...)
			b.WriteString(")")
			b.WriteString(")")
		}))
	}
}
