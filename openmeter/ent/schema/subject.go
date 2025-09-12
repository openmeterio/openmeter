package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// Subject holds the schema definition for the Subject entity.
type Subject struct {
	ent.Schema
}

// Fields of the Subject.
func (Subject) Fields() []ent.Field {
	return []ent.Field{
		field.String("key").NotEmpty(),
		field.String("display_name").Optional().Nillable(),
		field.String("stripe_customer_id").Optional().Nillable(),
		field.JSON("metadata", map[string]interface{}{}).
			Optional().
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}),
	}
}

// Mixin of the Subject.
func (Subject) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
	}
}

// Indexes of the Subject.
func (Subject) Indexes() []ent.Index {
	return []ent.Index{
		// unique for each organization
		index.Fields("namespace", "key").
			Annotations(
				entsql.IndexWhere("deleted_at IS NULL"),
			).Unique(),
		// we sort by display name
		index.Fields("display_name"),
		// so that we can fetch the recently created subjects, id is there for stable pagination
		index.Fields("created_at", "id"),
	}
}

// Edges of the Subject.
func (Subject) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("entitlements", Entitlement.Type),
	}
}
