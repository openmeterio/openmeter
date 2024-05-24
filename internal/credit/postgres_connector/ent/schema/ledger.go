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

package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

var defaultHighwatermark, _ = time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")

type Ledger struct {
	ent.Schema
}

// Mixin of the Ledger.
func (Ledger) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
		IDMixin{},
	}
}

// Fields of the Ledger.
func (Ledger) Fields() []ent.Field {
	return []ent.Field{
		field.String("namespace").NotEmpty().Immutable(),
		field.String("subject").NotEmpty().Immutable(),
		field.JSON("metadata", map[string]string{}).Optional(),
		field.Time("highwatermark").Default(func() time.Time {
			return defaultHighwatermark
		}),
	}
}

// Indexes of the Ledger.
func (Ledger) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "subject").Unique(),
	}
}
