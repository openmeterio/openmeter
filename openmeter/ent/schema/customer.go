package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

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
		field.String("address_country").MinLen(2).MaxLen(2).Optional().Nillable(),
		field.String("address_postal_code").Optional().Nillable(),
		field.String("address_state").Optional().Nillable(),
		field.String("address_city").Optional().Nillable(),
		field.String("address_line1").Optional().Nillable(),
		field.String("address_line2").Optional().Nillable(),
		field.String("address_phone_number").Optional().Nillable(),
	}
}

func (Customer) Indexes() []ent.Index {
	return []ent.Index{}
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

func (CustomerSubjects) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.TimeMixin{},
	}
}

func (CustomerSubjects) Fields() []ent.Field {
	return []ent.Field{
		field.String("customer_id").Immutable(),
		field.String("subject_key").Immutable(),
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
			Unique(),
	}
}
