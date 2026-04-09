package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

func creditRealizationLineageNow() time.Time {
	return clock.Now().Truncate(time.Microsecond)
}

type CreditRealizationLineage struct {
	ent.Schema
}

func (CreditRealizationLineage) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
	}
}

func (CreditRealizationLineage) Fields() []ent.Field {
	return []ent.Field{
		field.String("root_realization_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			NotEmpty().
			Immutable(),
		field.String("customer_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			NotEmpty().
			Immutable(),
		field.String("currency").
			GoType(currencyx.Code("")).
			NotEmpty().
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "varchar(3)",
			}),
		field.Enum("origin_kind").
			GoType(creditrealization.LineageOriginKind("")).
			Immutable(),
		field.Time("created_at").
			Default(creditRealizationLineageNow).
			Immutable(),
	}
}

func (CreditRealizationLineage) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("segments", CreditRealizationLineageSegment.Type),
	}
}

func (CreditRealizationLineage) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "root_realization_id").Unique(),
		index.Fields("namespace", "customer_id"),
	}
}

type CreditRealizationLineageSegment struct {
	ent.Schema
}

func (CreditRealizationLineageSegment) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
	}
}

func (CreditRealizationLineageSegment) Fields() []ent.Field {
	return []ent.Field{
		field.String("lineage_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			NotEmpty().
			Immutable(),
		field.Other("amount", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),
		field.Enum("state").
			GoType(creditrealization.LineageSegmentState("")).
			Immutable(),
		field.String("backing_transaction_group_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Optional().
			NotEmpty().
			Nillable(),
		field.Time("closed_at").
			Optional().
			Nillable(),
		field.Time("created_at").
			Default(creditRealizationLineageNow).
			Immutable(),
	}
}

func (CreditRealizationLineageSegment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("lineage", CreditRealizationLineage.Type).
			Ref("segments").
			Field("lineage_id").
			Required().
			Unique().
			Immutable(),
	}
}

func (CreditRealizationLineageSegment) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("lineage_id"),
		index.Fields("lineage_id", "closed_at"),
	}
}
