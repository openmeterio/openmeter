package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/charges"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
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
		field.Time("invoice_at"),

		field.Enum("type").
			GoType(charges.IntentType("")).
			Immutable(),

		field.Enum("status").
			GoType(charges.ChargeStatus("")),

		field.Enum("settlement_mode").
			GoType(productcatalog.SettlementMode("")).
			Immutable(),

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

		field.JSON("tax_config", productcatalog.TaxConfig{}).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Optional(),

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
			StorageKey(edge.Column("id")).
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Unique(),
		edge.To("usage_based", ChargeUsageBased.Type).
			StorageKey(edge.Column("id")).
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Unique(),
		// Realizations
		edge.To("standard_invoice_realizations", ChargeStandardInvoiceRealization.Type),
		edge.To("credit_realizations", ChargeCreditRealization.Type),
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

type ChargeUsageBased struct {
	ent.Schema
}

func (ChargeUsageBased) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.NamespaceMixin{},
		entutils.IDMixin{},
	}
}

func (ChargeUsageBased) Fields() []ent.Field {
	return []ent.Field{
		field.String("price").
			GoType(&productcatalog.Price{}).
			ValueScanner(PriceValueScanner).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}),

		field.String("feature_key").
			NotEmpty(),

		field.String("discounts").
			GoType(&productcatalog.Discounts{}).
			ValueScanner(DiscountsValueScanner).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Optional().
			Nillable(),
	}
}

func (ChargeUsageBased) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "id").
			Unique(),
	}
}

func (ChargeUsageBased) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("charge", Charge.Type).
			Ref("usage_based").
			Unique().
			Required(),
	}
}

type ChargeFlatFee struct {
	ent.Schema
}

func (ChargeFlatFee) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.NamespaceMixin{},
		entutils.IDMixin{},
	}
}

func (ChargeFlatFee) Fields() []ent.Field {
	return []ent.Field{
		field.String("payment_term").
			GoType(productcatalog.PaymentTermType("")).
			NotEmpty(),

		field.String("discounts").
			GoType(&productcatalog.Discounts{}).
			ValueScanner(DiscountsValueScanner).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Optional().
			Nillable(),

		field.Enum("pro_rating").
			GoType(charges.ProRatingModeAdapterEnum("")),

		field.String("feature_key").
			Optional().
			NotEmpty().
			Nillable(),

		field.Other("amount_before_proration", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),

		field.Other("amount_after_proration", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),
	}
}

func (ChargeFlatFee) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("charge", Charge.Type).
			Ref("flat_fee").
			Unique().
			Required(),
	}
}

func (ChargeFlatFee) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "id").
			Unique(),
	}
}

type ChargeStandardInvoiceRealization struct {
	ent.Schema
}

func (ChargeStandardInvoiceRealization) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.NamespaceMixin{},
		entutils.IDMixin{},
		entutils.TimeMixin{},
		entutils.AnnotationsMixin{},
		TotalsMixin{},
	}
}

func (ChargeStandardInvoiceRealization) Fields() []ent.Field {
	return []ent.Field{
		field.String("line_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Immutable(),

		field.String("charge_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Immutable(),

		field.Time("service_period_from"),
		field.Time("service_period_to"),

		field.Enum("status").
			GoType(charges.StandardInvoiceRealizationStatus("")),

		field.Other("metered_service_period_quantity", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),

		field.Other("metered_pre_service_period_quantity", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),
	}
}

func (ChargeStandardInvoiceRealization) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "charge_id", "line_id").
			Unique(),
	}
}

func (ChargeStandardInvoiceRealization) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("charge", Charge.Type).
			Ref("standard_invoice_realizations").
			Field("charge_id").
			Unique().
			Required().
			Immutable(),
		edge.To("billing_invoice_line", BillingInvoiceLine.Type).
			Field("line_id").
			Unique().
			Required().
			Immutable(),
		edge.To("credit_realization", ChargeCreditRealization.Type).
			Unique(),
	}
}

type ChargeCreditRealization struct {
	ent.Schema
}

func (ChargeCreditRealization) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.NamespaceMixin{},
		entutils.IDMixin{},
		entutils.TimeMixin{},
		entutils.AnnotationsMixin{},
	}
}

func (ChargeCreditRealization) Fields() []ent.Field {
	return []ent.Field{
		field.String("charge_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Immutable(),

		field.String("std_realization_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Optional().
			NotEmpty().
			Nillable(),

		field.Other("amount", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),

		field.Time("service_period_from"),
		field.Time("service_period_to"),
	}
}

func (ChargeCreditRealization) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("charge", Charge.Type).
			Ref("credit_realizations").
			Field("charge_id").
			Unique().
			Required().
			Immutable(),
		edge.From("standard_invoice_realization", ChargeStandardInvoiceRealization.Type).
			Ref("credit_realization").
			Field("std_realization_id").
			Unique(),
	}
}
