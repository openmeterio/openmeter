package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type PlanAddon struct {
	ent.Schema
}

func (PlanAddon) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.MetadataMixin{},
		entutils.AnnotationsMixin{},
		entutils.TimeMixin{},
	}
}

func (PlanAddon) Fields() []ent.Field {
	return []ent.Field{
		field.String("plan_id").
			NotEmpty().
			Immutable().
			Comment("The plan identifier the add-on is assigned to."),
		field.String("addon_id").
			NotEmpty().
			Immutable().
			Comment("The add-on identifier the plan is assigned to."),
		field.String("from_plan_phase").
			Comment("The key identifier of the plan phase from the add-on is available fro purchase."),
		field.Int("max_quantity").
			Optional().
			Nillable().
			Comment("The maximum quantity of the add-on that can be purchased."),
	}
}

func (PlanAddon) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("plan", Plan.Type).
			Ref("addons").
			Field("plan_id").
			Required().
			Immutable().
			Unique(),
		edge.From("addon", Addon.Type).
			Ref("plans").
			Field("addon_id").
			Required().
			Immutable().
			Unique(),
	}
}

func (PlanAddon) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "plan_id", "addon_id").
			Annotations(
				entsql.IndexWhere("deleted_at IS NULL"),
			).
			Unique(),
	}
}
