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

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/recurrence"
)

type Entitlement struct {
	ent.Schema
}

func (Entitlement) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.MetadataAnnotationsMixin{},
		entutils.TimeMixin{},
	}
}

func (Entitlement) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("entitlement_type").Values(entitlement.EntitlementType("").StrValues()...).Immutable(),
		field.String("feature_id").Immutable().SchemaType(map[string]string{
			dialect.Postgres: "char(26)",
		}),
		field.String("feature_key").NotEmpty().Immutable(),
		field.String("subject_key").NotEmpty().Immutable(),
		field.Time("measure_usage_from").Optional().Nillable().Immutable(),
		field.Float("issue_after_reset").Optional().Nillable().Immutable(),
		field.Uint8("issue_after_reset_priority").Optional().Nillable().Immutable(),
		field.Bool("is_soft_limit").Optional().Nillable().Immutable(),
		field.Bool("preserve_overage_at_reset").Optional().Nillable().Immutable(),
		field.JSON("config", []byte{}).SchemaType(map[string]string{
			dialect.Postgres: "jsonb",
		}).Optional(),
		field.Enum("usage_period_interval").Values(recurrence.RecurrenceInterval("").Values()...).Optional().Nillable().Immutable(),
		field.Time("usage_period_anchor").Optional().Nillable(),
		field.Time("current_usage_period_start").Optional().Nillable(),
		field.Time("current_usage_period_end").Optional().Nillable(),
	}
}

func (Entitlement) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "id"),
		index.Fields("namespace", "subject_key"),
		index.Fields("namespace", "id", "subject_key"),
		index.Fields("namespace", "feature_id", "id"),
		index.Fields("namespace", "current_usage_period_end"),
		// index for collecting all the entitlements with due resets
		index.Fields("current_usage_period_end", "deleted_at"),
	}
}

func (Entitlement) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("usage_reset", UsageReset.Type),
		edge.To("grant", Grant.Type),
		edge.To("balance_snapshot", BalanceSnapshot.Type),
		edge.From("feature", Feature.Type).
			Ref("entitlement").
			Field("feature_id").
			Required().
			Unique().
			Immutable(),
	}
}
