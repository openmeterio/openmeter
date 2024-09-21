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
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type BalanceSnapshot struct {
	ent.Schema
}

func (BalanceSnapshot) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
	}
}

func (BalanceSnapshot) Fields() []ent.Field {
	return []ent.Field{
		field.String("owner_id").Immutable().SchemaType(map[string]string{
			dialect.Postgres: "char(26)",
		}),
		field.JSON("grant_balances", balance.Map{}).Immutable().SchemaType(map[string]string{
			dialect.Postgres: "jsonb",
		}),
		field.Float("balance").Immutable().SchemaType(map[string]string{
			dialect.Postgres: "numeric",
		}),
		field.Float("overage").Immutable().SchemaType(map[string]string{
			dialect.Postgres: "numeric",
		}),
		field.Time("at").Immutable(),
	}
}

func (BalanceSnapshot) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "at"),
		index.Fields("namespace", "balance"),
		index.Fields("namespace", "balance", "at"),
	}
}

func (BalanceSnapshot) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("entitlement", Entitlement.Type).
			Ref("balance_snapshot").
			Field("owner_id").
			Required().
			Immutable().
			Unique(),
	}
}
