package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type IntegrationStripe struct {
	ent.Schema
}

func (IntegrationStripe) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
		entutils.AppID{},
	}
}

func (IntegrationStripe) Fields() []ent.Field {
	return []ent.Field{
		// Stripe specific fields TODO: these should go into a seperate table
		field.String("stripe_account_id").Optional().Nillable().Immutable(),
		field.Bool("stripe_livemode").Optional().Nillable().Immutable(),
	}
}

type IntegrationStripeCustomer struct {
	ent.Schema
}

func (IntegrationStripeCustomer) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
		entutils.AppID{},
	}
}

func (IntegrationStripeCustomer) Fields() []ent.Field {
	return []ent.Field{
		field.String("customer_id"),
		field.String("stripe_customer_id").Optional().Nillable().Immutable(),
	}
}
