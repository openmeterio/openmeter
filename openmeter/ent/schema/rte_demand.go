package schema

import (
	"encoding/json"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

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
		field.Int64("refresh_interval").Default(0),
		field.Time("next_refresh_at").Optional().Nillable(),
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

// RTEInvalidation suppresses repeated source facts during the retention window.
// Replaying a fact after its receipt expires safely schedules extra calculation.
type RTEInvalidation struct{ ent.Schema }

func (RTEInvalidation) Mixin() []ent.Mixin { return []ent.Mixin{entutils.TimeMixin{}} }

func (RTEInvalidation) Fields() []ent.Field {
	return []ent.Field{
		field.String("source").NotEmpty().Immutable(),
		field.String("event_id").NotEmpty().Immutable(),
	}
}

func (RTEInvalidation) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("source", "event_id").Unique(),
		index.Fields("created_at", "id").StorageKey("rte_invalidation_retention"),
	}
}
