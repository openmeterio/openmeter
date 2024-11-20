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
	"github.com/openmeterio/openmeter/pkg/timezone"
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
	}
}

func (Customer) Fields() []ent.Field {
	return []ent.Field{
		field.String("primary_email").Optional().Nillable(),
		field.String("timezone").GoType(timezone.Timezone("")).Optional().Nillable(),
		field.String("currency").GoType(currencyx.Code("")).MinLen(3).MaxLen(3).Optional().Nillable(),
	}
}

func (Customer) Indexes() []ent.Index {
	return []ent.Index{
		// Indexes because of API filters
		index.Fields("name"),
		index.Fields("primary_email"),
		index.Fields("deleted_at"),
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
		// We use boolean for soft delete instead of time.Time
		// because we can only add unique indexes on fields that are not nullable.
		field.Bool("is_deleted").Default(false),
	}
}

func (CustomerSubjects) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "customer_id", "is_deleted"),
		index.Fields("namespace", "subject_key", "is_deleted").
			Annotations(
				// Partial index: We skip the index on active non deleted customer subjects.
				entsql.IndexWhere("is_deleted = false"),
			).
			Unique(),
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
	}
}
