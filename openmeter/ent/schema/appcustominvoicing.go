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

// AppCustomInvoicing holds the schema definition for the AppCustomInvoicing entity.
type AppCustomInvoicing struct {
	ent.Schema
}

func (AppCustomInvoicing) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
	}
}

func (AppCustomInvoicing) Fields() []ent.Field {
	return []ent.Field{
		field.Bool("skip_draft_sync_hook").Default(false),
		field.Bool("skip_issuing_sync_hook").Default(false),
	}
}

func (AppCustomInvoicing) Indexes() []ent.Index {
	return []ent.Index{}
}

func (AppCustomInvoicing) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("customer_apps", AppCustomInvoicingCustomer.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		// We need to set StorageKey instead of Field, othwerwise ent generate breaks.
		edge.To("app", App.Type).
			StorageKey(edge.Column("id")).
			Unique().
			Immutable().
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

// AppCustomInvoicingCustomer holds the schema definition for the AppCustomInvoicingCustomer entity.
type AppCustomInvoicingCustomer struct {
	ent.Schema
}

func (AppCustomInvoicingCustomer) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
		entutils.MetadataMixin{},
	}
}

func (AppCustomInvoicingCustomer) Fields() []ent.Field {
	return []ent.Field{
		field.String("app_id").NotEmpty().Immutable().SchemaType(map[string]string{
			dialect.Postgres: "char(26)",
		}),
		field.String("customer_id").NotEmpty().Immutable().SchemaType(map[string]string{
			dialect.Postgres: "char(26)",
		}),
	}
}

func (AppCustomInvoicingCustomer) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "app_id", "customer_id").
			Unique(),
	}
}

func (AppCustomInvoicingCustomer) Edges() []ent.Edge {
	return []ent.Edge{
		// We don't add an edge to App as it would also point to the `app_id`` field.
		// This would make ent to generate both a SetAppID and a SetCustomInvoicingAppID methods.
		// Where both methods are required to set but setting both errors out.
		// The App has the same ID as the CustomInvoicingApp so it's simple to resolve.
		edge.From("custom_invoicing_app", AppCustomInvoicing.Type).
			Ref("customer_apps").
			Field("app_id").
			Immutable().
			Required().
			Unique(),
		edge.To("customer", Customer.Type).
			Field("customer_id").
			Unique().
			Immutable().
			Required().
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}
