package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/pkg/clock"
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
		// We don't use the time mixin because we don't want deleted_at
		field.Time("created_at").
			Default(clock.Now).
			Immutable(),
		field.Time("updated_at").
			Default(clock.Now).
			UpdateDefault(clock.Now),
	}
}

// Mixin of the Subject.
func (Subject) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
	}
}

// Indexes of the Subject.
func (Subject) Indexes() []ent.Index {
	return []ent.Index{
		// unique for each organization
		index.Fields("key", "namespace").Unique(),
		// we sort by display name
		index.Fields("display_name"),
	}
}

// Edges of the Subject.
func (Subject) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("entitlements", Entitlement.Type),

		// FIXME: enable foreign key constraints
		// Ent doesn't support foreign key constraints on non ID fields (key)
		// https://github.com/ent/ent/issues/2549
		// edge.To("subject_key", CustomerSubjects.Type),
	}
}
