package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

type BillingInvoiceSearchV1 struct {
	ent.View
}

func (BillingInvoiceSearchV1) Mixin() []ent.Mixin {
	// Views cannot use mixins that define indexes or edges.
	return []ent.Mixin{}
}

func (v BillingInvoiceSearchV1) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.ViewFor(dialect.Postgres, func(s *sql.Selector) {
			legacyGathering := sql.Dialect(dialect.Postgres).Select()
			dedicatedGathering := sql.Dialect(dialect.Postgres).Select()

			v.buildStandardInvoiceTableSelector(s)
			v.buildLegacyGatheringInvoiceTableSelector(legacyGathering)
			v.buildGatheringInvoiceTableSelector(dedicatedGathering)

			s.UnionAll(legacyGathering).UnionAll(dedicatedGathering)
		}),
	}
}

func (BillingInvoiceSearchV1) buildStandardInvoiceTableSelector(s *sql.Selector) {
	invoices := sql.Table("billing_invoices")

	s.From(invoices).
		Select(
			invoices.C("id"),
			invoices.C("namespace"),
			invoices.C("customer_id"),
			invoices.C("customer_name"),
			invoices.C("currency"),
		).
		AppendSelectAs(invoices.C("type"), "invoice_type").
		AppendSelect(
			invoices.C("status"),
			invoices.C("issued_at"),
		).
		AppendSelectAs(invoices.C("period_start"), "service_period_start").
		AppendSelectAs(invoices.C("period_end"), "service_period_end").
		AppendSelect(
			invoices.C("created_at"),
			invoices.C("updated_at"),
			invoices.C("deleted_at"),
			invoices.C("draft_until"),
			invoices.C("collection_at"),
			invoices.C("status_details_cache"),
			invoices.C("invoicing_app_external_id"),
			invoices.C("payment_app_external_id"),
			invoices.C("tax_app_external_id"),
			invoices.C("schema_level"),
		).
		Where(sql.ExprP(invoices.C("status") + " <> '" + string(billing.StandardInvoiceStatusGathering) + "'"))
}

func (BillingInvoiceSearchV1) buildLegacyGatheringInvoiceTableSelector(s *sql.Selector) {
	invoices := sql.Table("billing_invoices")
	customers := sql.Table("customers")
	dedicatedGatheringInvoices := sql.Table("billing_gathering_invoices")

	s.From(invoices).
		Join(customers).
		OnP(sql.And(
			sql.ColumnsEQ(customers.C("namespace"), invoices.C("namespace")),
			sql.ColumnsEQ(customers.C("id"), invoices.C("customer_id")),
		)).
		LeftJoin(dedicatedGatheringInvoices).
		OnP(sql.And(
			sql.ColumnsEQ(dedicatedGatheringInvoices.C("namespace"), invoices.C("namespace")),
			sql.ColumnsEQ(dedicatedGatheringInvoices.C("id"), invoices.C("id")),
		)).
		Select(
			invoices.C("id"),
			invoices.C("namespace"),
			invoices.C("customer_id"),
		).
		AppendSelectAs(customers.C("name"), "customer_name").
		AppendSelect(invoices.C("currency")).
		AppendSelectExprAs(sql.Raw("'gathering'"), "invoice_type").
		AppendSelectExprAs(sql.Raw("'gathering'"), "status").
		AppendSelectExprAs(sql.Raw("NULL::timestamptz"), "issued_at").
		AppendSelectAs(invoices.C("period_start"), "service_period_start").
		AppendSelectAs(invoices.C("period_end"), "service_period_end").
		AppendSelect(
			invoices.C("created_at"),
			invoices.C("updated_at"),
			invoices.C("deleted_at"),
		).
		AppendSelectExprAs(sql.Raw("NULL::timestamptz"), "draft_until").
		AppendSelect(invoices.C("collection_at")).
		AppendSelectExprAs(sql.Raw("NULL::jsonb"), "status_details_cache").
		AppendSelectExprAs(sql.Raw("NULL::text"), "invoicing_app_external_id").
		AppendSelectExprAs(sql.Raw("NULL::text"), "payment_app_external_id").
		AppendSelectExprAs(sql.Raw("NULL::text"), "tax_app_external_id").
		AppendSelect(invoices.C("schema_level")).
		Where(sql.And(
			sql.ExprP(invoices.C("status")+" = '"+string(billing.StandardInvoiceStatusGathering)+"'"),
			sql.IsNull(dedicatedGatheringInvoices.C("id")),
		))
}

func (BillingInvoiceSearchV1) buildGatheringInvoiceTableSelector(s *sql.Selector) {
	invoices := sql.Table("billing_gathering_invoices")
	customers := sql.Table("customers")

	s.From(invoices).
		Join(customers).
		OnP(sql.And(
			sql.ColumnsEQ(customers.C("namespace"), invoices.C("namespace")),
			sql.ColumnsEQ(customers.C("id"), invoices.C("customer_id")),
		)).
		Select(
			invoices.C("id"),
			invoices.C("namespace"),
			invoices.C("customer_id"),
		).
		AppendSelectAs(customers.C("name"), "customer_name").
		AppendSelect(invoices.C("currency")).
		AppendSelectExprAs(sql.Raw("'gathering'"), "invoice_type").
		AppendSelectExprAs(sql.Raw("'gathering'"), "status").
		AppendSelectExprAs(sql.Raw("NULL::timestamptz"), "issued_at").
		AppendSelect(invoices.C("service_period_start")).
		AppendSelect(invoices.C("service_period_end")).
		AppendSelect(
			invoices.C("created_at"),
			invoices.C("updated_at"),
			invoices.C("deleted_at"),
		).
		AppendSelectExprAs(sql.Raw("NULL::timestamptz"), "draft_until").
		AppendSelectAs(invoices.C("next_collection_at"), "collection_at").
		AppendSelectExprAs(sql.Raw("NULL::jsonb"), "status_details_cache").
		AppendSelectExprAs(sql.Raw("NULL::text"), "invoicing_app_external_id").
		AppendSelectExprAs(sql.Raw("NULL::text"), "payment_app_external_id").
		AppendSelectExprAs(sql.Raw("NULL::text"), "tax_app_external_id").
		AppendSelect(invoices.C("schema_level"))
}

func (BillingInvoiceSearchV1) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}),
		field.String("namespace").
			Immutable(),
		field.String("customer_id").
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}),
		field.String("customer_name"),
		field.String("currency").
			GoType(currencyx.Code("")).
			SchemaType(map[string]string{
				dialect.Postgres: "varchar(3)",
			}),
		field.String("invoice_type").
			GoType(billing.InvoiceType("")),
		field.String("status").
			GoType(billing.StandardInvoiceStatus("")),
		field.Time("issued_at").
			Optional().
			Nillable(),
		field.Time("service_period_start").
			Optional().
			Nillable(),
		field.Time("service_period_end").
			Optional().
			Nillable(),
		field.Time("created_at"),
		field.Time("updated_at"),
		field.Time("deleted_at").
			Optional().
			Nillable(),
		field.Time("draft_until").
			Optional().
			Nillable(),
		field.Time("collection_at").
			Optional().
			Nillable(),
		field.JSON("status_details_cache", billing.StandardInvoiceStatusDetails{}).
			Optional(),
		field.String("invoicing_app_external_id").
			Optional().
			Nillable(),
		field.String("payment_app_external_id").
			Optional().
			Nillable(),
		field.String("tax_app_external_id").
			Optional().
			Nillable(),
		field.Int("schema_level"),
	}
}
