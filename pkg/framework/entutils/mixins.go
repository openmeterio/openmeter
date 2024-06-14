package entutils

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"entgo.io/ent/schema/mixin"
	"github.com/oklog/ulid/v2"
)

// IDMixin adds the ID field to the schema
type IDMixin struct {
	mixin.Schema
}

// Fields of the IDMixin.
func (IDMixin) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			DefaultFunc(func() string {
				return ulid.Make().String()
			}).
			Unique().
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}),
	}
}

func (IDMixin) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("id"),
	}
}

// NamespaceMixin can be used for namespaced entities
type NamespaceMixin struct {
	mixin.Schema
}

// Fields of the IDMixin.
func (NamespaceMixin) Fields() []ent.Field {
	return []ent.Field{
		field.String("namespace").
			NotEmpty().
			Immutable(),
	}
}

// NamespaceMixin can be used for namespaced entities
type MetadataAnnotationsMixin struct {
	mixin.Schema
}

// Fields of the IDMixin.
func (MetadataAnnotationsMixin) Fields() []ent.Field {
	return []ent.Field{
		field.JSON("metadata", map[string]string{}).
			Optional().
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}),
	}
}

// TimeMixin adds the created_at and updated_at fields to the schema
type TimeMixin struct {
	mixin.Schema
}

// Fields of the TimeMixin.
func (TimeMixin) Fields() []ent.Field {
	return []ent.Field{
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
		field.Time("deleted_at").
			Optional().
			Nillable(),
	}
}
