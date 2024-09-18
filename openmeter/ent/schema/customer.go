package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Customer stores information about a customer
type Customer struct {
	ent.Schema
}

func (Customer) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.ResourceMixin{},
	}
}

func (Customer) Fields() []ent.Field {
	return []ent.Field{
		field.String("currency").GoType(models.CurrencyCode("")).MinLen(3).MaxLen(3).Optional().Nillable(),
		field.Enum("tax_provider").GoType(models.TaxProvider("")).Optional().Nillable(),
		field.Enum("invoicing_provider").GoType(models.InvoicingProvider("")).Optional().Nillable(),
		field.Enum("payment_provider").GoType(models.PaymentProvider("")).Optional().Nillable(),
		field.String("external_mapping_stripe_customer_id").Optional().Nillable(),

		// PII fields
		field.String("primary_email").Optional().Nillable(),
		field.String("billing_address_country").GoType(models.CountryCode("")).MinLen(2).MaxLen(2).Optional().Nillable(),
		field.String("billing_address_postal_code").Optional().Nillable(),
		field.String("billing_address_state").Optional().Nillable(),
		field.String("billing_address_city").Optional().Nillable(),
		field.String("billing_address_line1").Optional().Nillable(),
		field.String("billing_address_line2").Optional().Nillable(),
		field.String("billing_address_phone_number").Optional().Nillable(),
	}
}

func (Customer) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("subjects", CustomerSubjects.Type),
	}
}

// CustomerSubject stores the subject keys for a customer
type CustomerSubjects struct {
	ent.Schema
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
	}
}

func (CustomerSubjects) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("customer_id", "subject_key").
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
