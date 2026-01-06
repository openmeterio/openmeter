package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// Customer stores information about a customer
type Customer struct {
	ent.Schema
}

func (Customer) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.ResourceMixin{},
		entutils.CustomerAddressMixin{
			FieldPrefix: "billing",
		},
		entutils.AnnotationsMixin{},
	}
}

func (Customer) Fields() []ent.Field {
	return []ent.Field{
		// We store non-set in db as an emoty string instead of null
		// because we can only add unique indexes on fields that are not nullable.
		field.String("key").Optional(),
		field.String("primary_email").Optional().Nillable(),
		field.String("currency").GoType(currencyx.Code("")).MinLen(3).MaxLen(3).Optional().Nillable(),
	}
}

func (Customer) Indexes() []ent.Index {
	return []ent.Index{
		// Indexes because of API filters
		index.Fields("namespace", "key").
			Annotations(
				entsql.IndexWhere("deleted_at IS NULL"),
			).
			Unique(),
		index.Fields("namespace", "key", "deleted_at"),
		index.Fields("name"),
		index.Fields("primary_email"),
		// Indexes because of API OrderBy
		index.Fields("created_at"),
	}
}

func (Customer) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("apps", AppCustomer.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("subjects", CustomerSubjects.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("billing_customer_override", BillingCustomerOverride.Type).
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("billing_invoice", BillingInvoice.Type),
		edge.To("subscription", Subscription.Type),
		edge.To("entitlements", Entitlement.Type),
	}
}

// CustomerSubject stores the subject keys for a customer
type CustomerSubjects struct {
	ent.Schema
}

func (CustomerSubjects) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.NamespaceMixin{},
	}
}

func (CustomerSubjects) Fields() []ent.Field {
	return []ent.Field{
		field.String("customer_id").
			Immutable().
			NotEmpty().
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}),
		field.String("subject_key").
			Immutable().
			NotEmpty(),
		field.Time("created_at").
			Default(clock.Now).
			Immutable(),
		field.Time("deleted_at").
			Optional().
			Nillable(),
	}
}

func (CustomerSubjects) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "customer_id", "deleted_at"),
		index.Fields("namespace", "subject_key").
			Annotations(
				// Partial index: We skip the index on active non deleted customer subjects.
				entsql.IndexWhere("deleted_at IS NULL"),
			).
			Unique(),
		// These two indexes are best picked up by ent's `.WithCustomer()`... edges where it generates
		// ...WHERE id IN (SELECT ...) type queries.
		index.Fields("deleted_at"),
		index.Fields("subject_key"),
		// For other common queries based on analytics
		index.Fields("customer_id"),
		index.Fields("deleted_at", "customer_id"),
	}
}

func (CustomerSubjects) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("customer", Customer.Type).
			Ref("subjects").
			Field("customer_id").
			Required().
			Immutable().
			Unique(),
		// FIXME: enable foreign key constraints
		// Ent doesn't support foreign key constraints on non ID fields (key)
		// https://github.com/ent/ent/issues/2549
		// edge.From("key", Subject.Type).
		// 	Ref("subject").
		// 	Field("subject_key").
		// 	Required().
		// 	Immutable(),
	}
}
