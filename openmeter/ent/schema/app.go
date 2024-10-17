package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// App stores information about an installed app
type App struct {
	ent.Schema
}

func (App) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.ResourceMixin{},
	}
}

func (App) Fields() []ent.Field {
	return []ent.Field{
		field.String("type").GoType(appentitybase.AppType("")).Immutable(),
		field.String("status").GoType(appentitybase.AppStatus("")),
		field.Bool("is_default").Default(false),
	}
}

func (App) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "type"),
		index.Fields("namespace", "type", "is_default").
			Annotations(
				entsql.IndexWhere("is_default = true"),
			).
			Unique(),
	}
}

func (App) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("customer_apps", AppCustomer.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

// AppCustomer holds the schema definition for the AppCustomer entity.
type AppCustomer struct {
	ent.Schema
}

func (AppCustomer) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
	}
}

func (AppCustomer) Fields() []ent.Field {
	return []ent.Field{
		field.String("app_id").NotEmpty().Immutable().SchemaType(map[string]string{
			dialect.Postgres: "char(26)",
		}),
		field.String("customer_id").NotEmpty().Immutable().SchemaType(map[string]string{
			dialect.Postgres: "char(26)",
		}),
	}
}

func (AppCustomer) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "app_id", "customer_id").
			Unique(),
	}
}

func (AppCustomer) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("app", App.Type).
			Ref("customer_apps").
			Field("app_id").
			Immutable().
			Required().
			Unique(),
		edge.From("customer", Customer.Type).
			Ref("apps").
			Field("customer_id").
			Immutable().
			Required().
			Unique(),
	}
}
