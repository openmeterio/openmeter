package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/invoicedusage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

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
		field.Time("invoice_at").
			Immutable(),

		field.Enum("settlement_mode").
			GoType(productcatalog.SettlementMode("")).
			Immutable(),

		field.String("discounts").
			GoType(&productcatalog.Discounts{}).
			ValueScanner(DiscountsValueScanner).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Optional().
			Nillable(),

		field.String("feature_key").
			NotEmpty().
			Immutable(),

		field.String("price").
			GoType(&productcatalog.Price{}).
			ValueScanner(PriceValueScanner).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Immutable(),

		field.String("current_realization_run_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Optional().
			Nillable(),

		field.Enum("status").
			GoType(usagebased.Status("")),
	}
}

func (ChargeUsageBased) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("charge", Charge.Type).
			Ref("usage_based").
			Unique().
			Required(),
		edge.To("runs", ChargeUsageBasedRuns.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("current_run", ChargeUsageBasedRuns.Type).
			Field("current_realization_run_id").
			Unique(),
	}
}

func (ChargeUsageBased) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "id").
			Unique(),
	}
}

func (ChargeUsageBased) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "charge_usage_based"},
	}
}

type ChargeUsageBasedRuns struct {
	ent.Schema
}

func (ChargeUsageBasedRuns) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.NamespaceMixin{},
		entutils.IDMixin{},
		entutils.TimeMixin{},
		totals.Mixin{},
	}
}

func (ChargeUsageBasedRuns) Fields() []ent.Field {
	return []ent.Field{
		field.String("charge_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Immutable(),

		field.Enum("type").
			GoType(usagebased.RealizationRunType("")).
			Immutable(),

		field.Time("asof"),

		field.Time("collection_end").
			Immutable(),

		field.Other("meter_value", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),
	}
}

func (ChargeUsageBasedRuns) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("usage_based", ChargeUsageBased.Type).
			Ref("runs").
			Field("charge_id").
			Unique().
			Required().
			Immutable(),
		edge.To("credit_allocations", ChargeUsageBasedRunCreditAllocations.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("invoiced_usage", ChargeUsageBasedRunInvoicedUsage.Type).
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("payment", ChargeUsageBasedRunPayment.Type).
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

func (ChargeUsageBasedRuns) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "charge_id"),
	}
}

type ChargeUsageBasedRunCreditAllocations struct {
	ent.Schema
}

func (ChargeUsageBasedRunCreditAllocations) Mixin() []ent.Mixin {
	return []ent.Mixin{
		creditrealization.Mixin{},
	}
}

func (ChargeUsageBasedRunCreditAllocations) Fields() []ent.Field {
	return []ent.Field{
		field.String("run_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Immutable(),
	}
}

func (ChargeUsageBasedRunCreditAllocations) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("run", ChargeUsageBasedRuns.Type).
			Ref("credit_allocations").
			Field("run_id").
			Unique().
			Required().
			Immutable(),
	}
}

type ChargeUsageBasedRunInvoicedUsage struct {
	ent.Schema
}

func (ChargeUsageBasedRunInvoicedUsage) Mixin() []ent.Mixin {
	return []ent.Mixin{
		invoicedusage.Mixin{},
	}
}

func (ChargeUsageBasedRunInvoicedUsage) Fields() []ent.Field {
	return []ent.Field{
		field.String("run_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Immutable(),
	}
}

func (ChargeUsageBasedRunInvoicedUsage) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("run", ChargeUsageBasedRuns.Type).
			Ref("invoiced_usage").
			Field("run_id").
			Unique().
			Required().
			Immutable(),
	}
}

type ChargeUsageBasedRunPayment struct {
	ent.Schema
}

func (ChargeUsageBasedRunPayment) Mixin() []ent.Mixin {
	return []ent.Mixin{
		payment.InvoicedMixin{},
	}
}

func (ChargeUsageBasedRunPayment) Fields() []ent.Field {
	return []ent.Field{
		field.String("run_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Immutable(),
	}
}

func (ChargeUsageBasedRunPayment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("run", ChargeUsageBasedRuns.Type).
			Ref("payment").
			Field("run_id").
			Unique().
			Required().
			Immutable(),
	}
}
