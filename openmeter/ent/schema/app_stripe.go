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

// AppStripe holds the schema definition for the AppStripe entity.
type AppStripe struct {
	ent.Schema
}

func (AppStripe) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
	}
}

func (AppStripe) Fields() []ent.Field {
	return []ent.Field{
		field.String("api_key").Nillable().NotEmpty().Sensitive(),
		field.String("webhook_secret").NotEmpty().Sensitive(),
		field.String("stripe_account_id").Immutable(),
		field.Bool("stripe_livemode").Immutable(),
	}
}

func (AppStripe) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("customer_apps", AppStripeCustomer.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		// We need to set StorageKey instead of Field, othwerwise ent generate breaks.
		edge.To("app", App.Type).
			StorageKey(edge.Column("id")).
			Unique().
			Immutable().
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

// AppStripeCustomer holds the schema definition for the AppStripeCustomer entity.
type AppStripeCustomer struct {
	ent.Schema
}

func (AppStripeCustomer) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
	}
}

func (AppStripeCustomer) Fields() []ent.Field {
	return []ent.Field{
		field.String("app_id").NotEmpty().Immutable().SchemaType(map[string]string{
			dialect.Postgres: "char(26)",
		}),
		field.String("customer_id").NotEmpty().Immutable().SchemaType(map[string]string{
			dialect.Postgres: "char(26)",
		}),
		field.String("stripe_customer_id").Optional().Nillable(),
		field.String("stripe_default_payment_method_id").Optional().Nillable(),
	}
}

func (AppStripeCustomer) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "app_id", "customer_id").
			Unique(),
	}
}

func (AppStripeCustomer) Edges() []ent.Edge {
	return []ent.Edge{
		// We don't add an edge to App as it would also point to the `app_id`` field.
		// This would make ent to generate both a SetAppID and a SetStripeAppID methods.
		// Where both methods are required to set but setting both errors out.
		// The App has the same ID as the StripeApp so it's simple to resolve.
		edge.From("stripe_app", AppStripe.Type).
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
