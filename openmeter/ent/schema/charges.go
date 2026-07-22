package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"entgo.io/ent/schema/mixin"
	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

var chargesSearchV1Columns = []string{
	"id",
	"namespace",
	"metadata",
	"created_at",
	"updated_at",
	"deleted_at",
	"name",
	"description",
	"annotations",
	"customer_id",
	"service_period_from",
	"service_period_to",
	"billing_period_from",
	"billing_period_to",
	"full_service_period_from",
	"full_service_period_to",
	"status",
	"unique_reference_id",
	"currency",
	"custom_currency_id",
	"managed_by",
	"subscription_id",
	"subscription_phase_id",
	"subscription_item_id",
	"advance_after",
	"tax_code_id",
	"tax_behavior",
}

type ChargesSearchV1 struct {
	ent.View
}

func (ChargesSearchV1) Mixin() []ent.Mixin {
	// BigFatWarning: Do not use any mixins here, that are defining indexes or edges or ent generation will panic
	return []ent.Mixin{}
}

func (v ChargesSearchV1) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.ViewFor(dialect.Postgres, func(s *sql.Selector) {
			flatFee := sql.Dialect(dialect.Postgres).Select()
			usageBased := sql.Dialect(dialect.Postgres).Select()

			v.buildChargesSearchV1TableSelector(s, "charge_credit_purchases", meta.ChargeTypeCreditPurchase)
			v.buildChargesSearchV1TableSelector(flatFee, "charge_flat_fees", meta.ChargeTypeFlatFee)
			v.buildChargesSearchV1TableSelector(usageBased, "charge_usage_based", meta.ChargeTypeUsageBased)

			s.UnionAll(flatFee).UnionAll(usageBased)
		}),
	}
}

func (ChargesSearchV1) buildChargesSearchV1TableSelector(s *sql.Selector, table string, chargeType meta.ChargeType) {
	baseIntentDeletedAt := sql.Raw(`"intent_deleted_at"`)
	if chargeType == meta.ChargeTypeCreditPurchase {
		baseIntentDeletedAt = sql.Raw("NULL::timestamptz")
	}

	s.From(sql.Table(table)).
		Select(chargesSearchV1Columns...).
		AppendSelectExprAs(baseIntentDeletedAt, "base_intent_deleted_at").
		AppendSelectExprAs(sql.Raw("'"+string(chargeType)+"'"), "type")
}

func (ChargesSearchV1) Fields() []ent.Field {
	mixins := []ent.Mixin{
		ChargesMetaMixin{},
	}

	fields := []ent.Field{
		field.String("type").
			GoType(meta.ChargeType("")).
			Immutable(),
		field.Time("base_intent_deleted_at").
			Optional().
			Nillable(),
	}

	for _, mixin := range mixins {
		fields = append(fields, mixin.Fields()...)
	}

	return fields
}

type Charge struct {
	ent.Schema
}

func (Charge) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.NamespaceMixin{},
		entutils.IDMixin{},
	}
}

func (Charge) Fields() []ent.Field {
	return []ent.Field{
		field.Time("created_at").
			Default(clock.Now).
			Immutable(),
		field.Time("deleted_at").
			Optional().
			Nillable(),

		field.String("unique_reference_id").
			NotEmpty().
			Optional().
			Nillable().
			Immutable(),
		field.String("type").
			GoType(meta.ChargeType("")).
			Immutable(),

		// Per type edge fields for FKs, so that we can mandate the existence of the charge from each subtype
		field.String("charge_flat_fee_id").
			Optional().
			Nillable().
			Immutable(),
		field.String("charge_credit_purchase_id").
			Optional().
			Nillable().
			Immutable(),
		field.String("charge_usage_based_id").
			Optional().
			Nillable().
			Immutable(),
	}
}

func (Charge) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("flat_fee", ChargeFlatFee.Type).
			Field("charge_flat_fee_id").
			Ref("charge").
			Immutable().
			Unique(),
		edge.From("credit_purchase", ChargeCreditPurchase.Type).
			Field("charge_credit_purchase_id").
			Ref("charge").
			Immutable().
			Unique(),
		edge.From("usage_based", ChargeUsageBased.Type).
			Field("charge_usage_based_id").
			Ref("charge").
			Immutable().
			Unique(),
		// Billing
		edge.To("billing_invoice_lines", BillingInvoiceLine.Type),
		edge.To("billing_gathering_invoice_lines", BillingGatheringInvoiceLine.Type).
			StorageKey(edge.Symbol("billing_gathering_line_charge_fk")),
		edge.To("billing_split_line_groups", BillingInvoiceSplitLineGroup.Type),
		edge.To("credit_realization_lineages", CreditRealizationLineage.Type),
	}
}

func (Charge) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "unique_reference_id").
			Annotations(
				entsql.IndexWhere("unique_reference_id IS NOT NULL AND deleted_at IS NULL"),
			).
			Unique(),
	}
}

type ChargesMetaMixin = entutils.RecursiveMixin[chargesMetaMixin]

type chargesMetaMixin struct {
	mixin.Schema
}

func (chargesMetaMixin) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.AnnotationsMixin{},
		entutils.ResourceMixin{},
	}
}

func (chargesMetaMixin) Fields() []ent.Field {
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

		field.Enum("status").
			GoType(meta.ChargeStatus("")),

		field.String("unique_reference_id").
			Immutable().
			Optional().
			Nillable(),

		field.String("fiat_currency_code").
			StorageKey("currency").
			GoType(currencyx.Code("")).
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "varchar(3)",
			}).
			NotEmpty().
			Optional().
			Nillable(),

		field.String("custom_currency_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			NotEmpty().
			Optional().
			Nillable().
			Immutable(),

		field.Enum("managed_by").
			GoType(billing.InvoiceLineManagedBy("")).
			Immutable(),

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
			Nillable(),

		field.Time("advance_after").
			Optional().
			Nillable(),
		field.String("tax_code_id").
			NotEmpty().
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}),
		field.Enum("tax_behavior").
			GoType(productcatalog.TaxBehavior("")).
			Optional().
			Nillable().
			Immutable(),
	}
}

func (chargesMetaMixin) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "customer_id", "unique_reference_id").
			Annotations(
				entsql.IndexWhere("unique_reference_id IS NOT NULL AND deleted_at IS NULL"),
			).
			Unique(),
	}
}

func (chargesMetaMixin) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Checks(map[string]string{
			"currency_reference": `(currency IS NULL) <> (custom_currency_id IS NULL)`,
			"currency_not_empty": `currency IS NULL OR currency <> ''`,
		}),
	}
}

type ChargeCostBasisMixin = entutils.RecursiveMixin[chargeCostBasisMixin]

type chargeCostBasisMixin struct {
	mixin.Schema
}

func (chargeCostBasisMixin) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.NamespaceMixin{},
		entutils.IDMixin{},
		entutils.TimeMixin{},
	}
}

func (chargeCostBasisMixin) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("mode").
			GoType(costbasis.Mode("")).
			Immutable(),
		field.String("fiat_currency").
			GoType(currencyx.FiatCode("")).
			NotEmpty().
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "varchar(3)",
			}),
		field.String("currency_cost_basis_id").
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}),
		field.String("resolved_cost_basis_id").
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}),
		field.String("currency_id").
			NotEmpty().
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}),
		field.Other("manual_rate", alpacadecimal.Decimal{}).
			Optional().
			Nillable().
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),
		field.Other("resolved_cost_basis", alpacadecimal.Decimal{}).
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),
		field.Time("resolved_at").
			Optional().
			Nillable(),
	}
}

func chargeCostBasisCurrencyEdges(symbolPrefix string) []ent.Edge {
	return []ent.Edge{
		edge.To("currency_cost_basis", CurrencyCostBasis.Type).
			Field("currency_cost_basis_id").
			Unique().
			StorageKey(edge.Symbol(symbolPrefix + "_currency_cost_basis_fk")).
			Annotations(entsql.OnDelete(entsql.Restrict)),
		edge.To("resolved_currency_cost_basis", CurrencyCostBasis.Type).
			Field("resolved_cost_basis_id").
			Unique().
			StorageKey(edge.Symbol(symbolPrefix + "_resolved_cost_basis_fk")).
			Annotations(entsql.OnDelete(entsql.Restrict)),
		edge.To("custom_currency", CustomCurrency.Type).
			Field("currency_id").
			Unique().
			Required().
			Immutable().
			StorageKey(edge.Symbol(symbolPrefix + "_currency_fk")).
			Annotations(entsql.OnDelete(entsql.Restrict)),
	}
}

func (chargeCostBasisMixin) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("currency_cost_basis_id"),
		index.Fields("resolved_cost_basis_id"),
		index.Fields("currency_id"),
	}
}

func (chargeCostBasisMixin) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Checks(map[string]string{
			"fiat_currency_not_empty":      `fiat_currency <> ''`,
			"resolved_cost_basis_positive": `resolved_cost_basis IS NULL OR resolved_cost_basis > 0`,
			"state": `
				(
					mode = 'dynamic'
					AND currency_cost_basis_id IS NULL
					AND manual_rate IS NULL
					AND (
						(resolved_cost_basis_id IS NULL AND resolved_cost_basis IS NULL AND resolved_at IS NULL)
						OR (resolved_cost_basis_id IS NOT NULL AND resolved_cost_basis IS NOT NULL AND resolved_at IS NOT NULL)
					)
				)
				OR (
					mode = 'pinned'
					AND currency_cost_basis_id IS NOT NULL
					AND resolved_cost_basis_id IS NOT NULL
					AND resolved_cost_basis_id = currency_cost_basis_id
					AND manual_rate IS NULL
					AND resolved_cost_basis IS NOT NULL
					AND resolved_at IS NOT NULL
				)
				OR (
					mode = 'manual'
					AND currency_cost_basis_id IS NULL
					AND resolved_cost_basis_id IS NULL
					AND manual_rate > 0
					AND resolved_cost_basis IS NOT NULL
					AND resolved_at IS NOT NULL
				)
			`,
		}),
	}
}
