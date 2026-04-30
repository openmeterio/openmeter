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

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/chargemeta"
	"github.com/openmeterio/openmeter/pkg/clock"
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
	s.From(sql.Table(table)).
		Select(chargesSearchV1Columns...).
		AppendSelectExprAs(sql.Raw("'"+string(chargeType)+"'"), "type")
}

func (ChargesSearchV1) Fields() []ent.Field {
	mixins := []ent.Mixin{
		chargemeta.Mixin{},
	}

	fields := []ent.Field{
		field.String("type").
			GoType(meta.ChargeType("")).
			Immutable(),
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
