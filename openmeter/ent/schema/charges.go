package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type Charge struct {
	ent.Schema
}

func (Charge) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.AnnotationsMixin{},
		entutils.ResourceMixin{},
	}
}

func (Charge) Fields() []ent.Field {
	return []ent.Field{
		field.String("customer_id").
			NotEmpty().
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}),

		field.Time("service_period_from"),
		field.Time("service_period_to"),
		field.Time("billing_period_from"),
		field.Time("billing_period_to"),
		field.Time("full_service_period_from"),
		field.Time("full_service_period_to"),

		field.Enum("type").
			GoType(meta.ChargeType("")).
			Immutable(),

		field.Enum("status").
			GoType(meta.ChargeStatus("")),

		field.String("unique_reference_id").
			Optional().
			Nillable(),

		field.String("currency").
			GoType(currencyx.Code("")).
			NotEmpty().
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "varchar(3)",
			}),

		field.Enum("managed_by").
			GoType(billing.InvoiceLineManagedBy("")),

		// Subscriptions metadata
		field.String("subscription_id").
			Optional().
			Nillable().
			Immutable(),

		field.String("subscription_phase_id").
			Optional().
			Nillable().
			Immutable(),

		field.String("subscription_item_id").
			Optional().
			Nillable().
			Immutable(),

		field.Time("advance_after").
			Optional().
			Nillable(),
		// TODO: Tax config!
	}
}

func (Charge) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "customer_id", "unique_reference_id").
			Annotations(
				entsql.IndexWhere("unique_reference_id IS NOT NULL AND deleted_at IS NULL"),
			).
			Unique(),
	}
}

func (Charge) Edges() []ent.Edge {
	return []ent.Edge{
		// Union type edges
		edge.To("flat_fee", ChargeFlatFee.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)).
			StorageKey(edge.Column("id")).
			Unique(),
		edge.To("credit_purchase", ChargeCreditPurchase.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)).
			StorageKey(edge.Column("id")).
			Unique(),
		edge.To("usage_based", ChargeUsageBased.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)).
			StorageKey(edge.Column("id")).
			Unique(),
		// Billing
		edge.To("billing_invoice_lines", BillingInvoiceLine.Type),
		edge.To("billing_split_line_groups", BillingInvoiceSplitLineGroup.Type),
		// Customer
		edge.From("customer", Customer.Type).
			Ref("charge_intents").
			Field("customer_id").
			Unique().
			Immutable().
			Required(),
		// Subscriptions
		edge.From("subscription", Subscription.Type).
			Ref("charge_intents").
			Field("subscription_id").
			Unique().
			Immutable(),
		edge.From("subscription_phase", SubscriptionPhase.Type).
			Ref("charge_intents").
			Field("subscription_phase_id").
			Unique().
			Immutable(),
		edge.From("subscription_item", SubscriptionItem.Type).
			Ref("charge_intents").
			Field("subscription_item_id").
			Unique().
			Immutable(),
	}
}
