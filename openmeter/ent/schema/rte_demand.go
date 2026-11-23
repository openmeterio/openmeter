package schema

import (
	"encoding/json"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type RTEDemand struct{ ent.Schema }

func (RTEDemand) Mixin() []ent.Mixin {
	return []ent.Mixin{entutils.NamespaceMixin{}, entutils.TimeMixin{}}
}

func (RTEDemand) Fields() []ent.Field {
	return []ent.Field{
		field.String("kind").NotEmpty().Immutable(),
		field.String("owner_id").NotEmpty().Immutable(),
		field.String("key_hash").NotEmpty().Immutable(),
		field.JSON("key", json.RawMessage{}).Immutable(),
		field.Bool("push_enabled").Default(false),
		field.Time("pull_until").Optional().Nillable(),
		field.Int64("dirty_version").Default(1),
		field.Int64("completed_version").Default(0),
		// Preserve previously recorded deletion deadlines when upgrading the worker.
		field.Time("source_delete_at").StorageKey("next_refresh_at").Optional().Nillable(),
		field.Time("last_calculated_at").Optional().Nillable(),
		field.Time("next_attempt_at"),
		field.Int("attempts").Default(0).NonNegative(),
	}
}

func (RTEDemand) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "kind", "key_hash").Unique().StorageKey("rte_demand_identity").Annotations(entsql.IndexWhere("deleted_at IS NULL")),
		index.Fields("namespace", "kind", "owner_id").StorageKey("rte_demand_owner"),
		index.Fields("next_attempt_at", "id").StorageKey("rte_demand_work").Annotations(
			entsql.IndexWhere("deleted_at IS NULL AND push_enabled AND (dirty_version > completed_version OR next_refresh_at IS NOT NULL)")),
	}
}

// RTEDemandLock gives each calculation its own PostgreSQL row lock. The row
// shares the demand's primary key and lives until that demand is deleted.
type RTEDemandLock struct{ ent.Schema }

func (RTEDemandLock) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").Immutable().Annotations(entsql.Annotation{Incremental: lo.ToPtr(false)}),
	}
}

func (RTEDemandLock) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("demand", RTEDemand.Type).StorageKey(edge.Column("id")).
			Unique().Immutable().Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}
