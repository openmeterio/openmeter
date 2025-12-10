package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"entgo.io/ent/schema/mixin"
	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

var (
	BillingDiscountsValueScanner      = entutils.JSONStringValueScanner[*billing.Discounts]()
	BillingDiscountReasonValueScanner = entutils.JSONStringValueScanner[*billing.DiscountReason]()
)

type BillingProfile struct {
	ent.Schema
}

func (BillingProfile) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.ResourceMixin{},
		entutils.CustomerAddressMixin{
			FieldPrefix: "supplier",
		},
	}
}

func (BillingProfile) Fields() []ent.Field {
	return []ent.Field{
		field.String("tax_app_id").
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}),
		field.String("invoicing_app_id").
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}),
		field.String("payment_app_id").
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}),
		field.String("workflow_config_id").
			NotEmpty(),
		field.Bool("default").
			Default(false),
		field.String("supplier_name").
			NotEmpty(),
		field.String("supplier_tax_code").
			Optional().
			Nillable(),
	}
}

func (BillingProfile) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("billing_invoices", BillingInvoice.Type),
		edge.To("billing_customer_override", BillingCustomerOverride.Type),
		edge.From("workflow_config", BillingWorkflowConfig.Type).
			Ref("billing_profile").
			Field("workflow_config_id").
			Unique().
			Required(),
		edge.From("tax_app", App.Type).
			Ref("billing_profile_tax_app").
			Field("tax_app_id").
			Unique().
			Immutable().
			Required(),
		edge.From("invoicing_app", App.Type).
			Ref("billing_profile_invoicing_app").
			Field("invoicing_app_id").
			Unique().
			Immutable().
			Required(),
		edge.From("payment_app", App.Type).
			Ref("billing_profile_payment_app").
			Field("payment_app_id").
			Unique().
			Immutable().
			Required(),
	}
}

func (BillingProfile) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "default").
			Annotations(
				entsql.IndexWhere("\"default\" AND deleted_at IS NULL"),
			).
			Unique(),
	}
}

type BillingWorkflowConfig struct {
	ent.Schema
}

func (BillingWorkflowConfig) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
	}
}

func (BillingWorkflowConfig) Fields() []ent.Field {
	return []ent.Field{
		// TODO: later we will add more alignment details here (e.g. monthly, yearly, etc.)
		field.Enum("collection_alignment").
			GoType(billing.AlignmentKind("")),

		field.JSON("anchored_alignment_detail", &billing.AnchoredAlignmentDetail{}).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Optional(),

		field.String("line_collection_period").GoType(datetime.ISODurationString("")),

		field.Bool("invoice_auto_advance"),

		field.String("invoice_draft_period").GoType(datetime.ISODurationString("")),

		field.String("invoice_due_after").GoType(datetime.ISODurationString("")),

		field.Enum("invoice_collection_method").
			GoType(billing.CollectionMethod("")),

		field.Bool("invoice_progressive_billing"),

		field.JSON("invoice_default_tax_settings", productcatalog.TaxConfig{}).
			Optional(),

		// Enable automatic tax calculation when tax is supported by the app.
		field.Bool("tax_enabled").Default(true),

		// Enforce tax calculation when tax is supported by the app.
		field.Bool("tax_enforced").Default(false),
	}
}

func (BillingWorkflowConfig) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "id"),
	}
}

func (BillingWorkflowConfig) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("billing_invoices", BillingInvoice.Type).
			Unique(),
		edge.To("billing_profile", BillingProfile.Type).
			Unique(),
	}
}

type BillingCustomerOverride struct {
	ent.Schema
}

func (BillingCustomerOverride) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
	}
}

func (BillingCustomerOverride) Fields() []ent.Field {
	return []ent.Field{
		field.String("customer_id").
			Unique().
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}),

		// For now we are not allowing for provider type overrides (that should be a separate billing provider entity).
		//
		// When we have the provider configs ready, we will add the field overrides for those specific fields.
		field.String("billing_profile_id").
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}),

		// Workflow config overrides
		field.Enum("collection_alignment").
			GoType(billing.AlignmentKind("")).
			Optional().
			Nillable(),

		field.JSON("anchored_alignment_detail", &billing.AnchoredAlignmentDetail{}).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Optional(),

		field.String("line_collection_period").
			GoType(datetime.ISODurationString("")).
			Optional().
			Nillable(),

		field.Bool("invoice_auto_advance").
			Optional().
			Nillable(),

		field.String("invoice_draft_period").
			GoType(datetime.ISODurationString("")).
			Optional().
			Nillable(),

		field.String("invoice_due_after").
			GoType(datetime.ISODurationString("")).
			Optional().
			Nillable(),

		field.Enum("invoice_collection_method").
			GoType(billing.CollectionMethod("")).
			Optional().
			Nillable(),

		field.Bool("invoice_progressive_billing").
			Optional().
			Nillable(),

		field.JSON("invoice_default_tax_config", productcatalog.TaxConfig{}).
			Optional(),
	}
}

func (BillingCustomerOverride) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "id").Unique(),
		index.Fields("namespace", "customer_id").Unique(),
	}
}

func (BillingCustomerOverride) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("customer", Customer.Type).
			Ref("billing_customer_override").
			Field("customer_id").
			Unique().
			Required().
			Immutable(),
		edge.From("billing_profile", BillingProfile.Type).
			Ref("billing_customer_override").
			Field("billing_profile_id").
			Unique(),
	}
}

type TotalsMixin struct {
	mixin.Schema
}

func (m TotalsMixin) Fields() []ent.Field {
	return []ent.Field{
		field.Other("amount", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),
		field.Other("taxes_total", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),
		field.Other("taxes_inclusive_total", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),
		field.Other("taxes_exclusive_total", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),
		field.Other("charges_total", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),
		field.Other("discounts_total", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),
		field.Other("total", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),
	}
}

type InvoiceLineBaseMixin struct {
	mixin.Schema
}

func (InvoiceLineBaseMixin) Fields() []ent.Field {
	return []ent.Field{
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
	}
}

type BillingInvoiceLine struct {
	ent.Schema
}

func (BillingInvoiceLine) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.AnnotationsMixin{},
		entutils.ResourceMixin{},
		InvoiceLineBaseMixin{},
		TotalsMixin{},
	}
}

func (BillingInvoiceLine) Fields() []ent.Field {
	return []ent.Field{
		// TODO: when we are done migrating the lines, let's also rename this to be service_period_start and service_period_end
		field.Time("period_start"),
		field.Time("period_end"),

		field.String("invoice_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}),

		field.Enum("managed_by").
			GoType(billing.InvoiceLineManagedBy("")),

		field.String("parent_line_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).Optional().Nillable(),

		field.Time("invoice_at"),

		// TODO[dependency]: overrides (as soon as plan override entities are ready)

		field.Enum("type").
			GoType(billing.InvoiceLineType("")).
			Immutable(),

		field.Enum("status").
			GoType(billing.InvoiceLineStatus("")),

		// Quantity is optional as for usage-based billing we can only persist this value,
		// when the invoice is issued
		field.Other("quantity", alpacadecimal.Decimal{}).
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),

		field.String("ratecard_discounts").
			GoType(&billing.Discounts{}).
			ValueScanner(BillingDiscountsValueScanner).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Optional().
			Nillable(),

		// ID of the line in the external invoicing app
		// For example, Stripe invoice line item ID
		field.String("invoicing_app_external_id").
			Optional().
			Nillable(),

		// child_unique_reference_id is uniqe per parent line, can be used for upserting
		// and identifying lines created for the same reason (e.g. tiered price tier)
		// between different invoices.
		field.String("child_unique_reference_id").
			Optional().
			Nillable(),

		// Subscriptions metadata
		field.String("subscription_id").
			Optional().
			Nillable(),

		field.String("subscription_phase_id").
			Optional().
			Nillable(),

		field.String("subscription_item_id").
			Optional().
			Nillable(),

		field.Time("subscription_billing_period_from").
			Optional().
			Nillable(),
		field.Time("subscription_billing_period_to").
			Optional().
			Nillable(),

		// NOTE: This is only valid for ubp lines, but eventually this table will become the "ubp" table
		field.String("split_line_group_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Optional().
			Nillable(),

		// Deprecated fields
		field.String("line_ids").
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Deprecated("invoice discounts are deprecated, use line_discounts instead"),
	}
}

func (BillingInvoiceLine) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "invoice_id"),
		index.Fields("namespace", "parent_line_id"),
		index.Fields("namespace", "parent_line_id", "child_unique_reference_id").
			Annotations(
				entsql.IndexWhere("child_unique_reference_id IS NOT NULL AND deleted_at IS NULL"),
			).Unique(),
		index.Fields("namespace", "subscription_id", "subscription_phase_id", "subscription_item_id"),
	}
}

func (BillingInvoiceLine) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("billing_invoice", BillingInvoice.Type).
			Ref("billing_invoice_lines").
			Field("invoice_id").
			Unique().
			Required(),
		edge.From("split_line_group", BillingInvoiceSplitLineGroup.Type).
			Ref("billing_invoice_lines").
			Field("split_line_group_id").
			Unique(),
		edge.To("flat_fee_line", BillingInvoiceFlatFeeLineConfig.Type).
			StorageKey(edge.Column("fee_line_config_id")).
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("usage_based_line", BillingInvoiceUsageBasedLineConfig.Type).
			StorageKey(edge.Column("usage_based_line_config_id")).
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("detailed_lines", BillingInvoiceLine.Type).
			From("parent_line").
			Field("parent_line_id").
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("line_usage_discounts", BillingInvoiceLineUsageDiscount.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("line_amount_discounts", BillingInvoiceLineDiscount.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.From("subscription", Subscription.Type).
			Ref("billing_lines").
			Field("subscription_id").
			Unique(),
		edge.From("subscription_phase", SubscriptionPhase.Type).
			Ref("billing_lines").
			Field("subscription_phase_id").
			Unique(),
		edge.From("subscription_item", SubscriptionItem.Type).
			Ref("billing_lines").
			Field("subscription_item_id").
			Unique(),
	}
}

type BillingInvoiceFlatFeeLineConfig struct {
	ent.Schema
}

func (BillingInvoiceFlatFeeLineConfig) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.NamespaceMixin{},
		entutils.IDMixin{},
	}
}

func (BillingInvoiceFlatFeeLineConfig) Fields() []ent.Field {
	return []ent.Field{
		field.Other("per_unit_amount", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),
		field.Enum("category").
			GoType(billing.FlatFeeCategory("")).
			Default(string(billing.FlatFeeCategoryRegular)),
		field.Enum("payment_term").
			GoType(productcatalog.PaymentTermType("")).
			Default(string(productcatalog.InAdvancePaymentTerm)),
		// Note: this is only used for sorting the lines in the invoice, only valid for detailed lines
		field.Int("index").
			Optional().
			Nillable(),
	}
}

type BillingInvoiceUsageBasedLineConfig struct {
	ent.Schema
}

func (BillingInvoiceUsageBasedLineConfig) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.NamespaceMixin{},
		entutils.IDMixin{},
	}
}

func (BillingInvoiceUsageBasedLineConfig) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("price_type").
			GoType(productcatalog.PriceType("")),
		field.String("feature_key").
			Immutable().
			Optional().
			Nillable(),
		field.String("price").
			GoType(&productcatalog.Price{}).
			ValueScanner(PriceValueScanner).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}),
		field.Other("pre_line_period_quantity", alpacadecimal.Decimal{}).
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),
		field.Other("metered_pre_line_period_quantity", alpacadecimal.Decimal{}).
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),
		field.Other("metered_quantity", alpacadecimal.Decimal{}).
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),
	}
}

type BillingInvoiceLineDiscountBase struct {
	mixin.Schema
}

func (BillingInvoiceLineDiscountBase) Fields() []ent.Field {
	return []ent.Field{
		field.String("line_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}),

		field.String("child_unique_reference_id").
			Optional().
			Nillable(),

		field.String("description").
			Optional().
			Nillable(),

		field.Enum("reason").
			GoType(billing.DiscountReasonType("")),

		// ID of the line discount in the external invoicing app
		// For example, Stripe invoice line item ID
		field.String("invoicing_app_external_id").
			Optional().
			Nillable(),
	}
}

type BillingInvoiceSplitLineGroup struct {
	ent.Schema
}

func (BillingInvoiceSplitLineGroup) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.ResourceMixin{},
		InvoiceLineBaseMixin{},
	}
}

func (BillingInvoiceSplitLineGroup) Fields() []ent.Field {
	return []ent.Field{
		field.Time("service_period_start"),
		field.Time("service_period_end"),

		field.String("unique_reference_id").
			Optional().
			Nillable().
			Immutable(),

		field.String("ratecard_discounts").
			GoType(&billing.Discounts{}).
			ValueScanner(BillingDiscountsValueScanner).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Optional().
			Nillable(),

		field.String("feature_key").
			Optional().
			Nillable().
			Immutable(),

		field.String("price").
			GoType(&productcatalog.Price{}).
			ValueScanner(PriceValueScanner).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).Immutable(),

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

		field.Time("subscription_billing_period_from").
			Optional().
			Nillable(),
		field.Time("subscription_billing_period_to").
			Optional().
			Nillable(),
	}
}

func (BillingInvoiceSplitLineGroup) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "unique_reference_id").
			Annotations(
				entsql.IndexWhere("unique_reference_id IS NOT NULL AND deleted_at IS NULL"),
			).Unique(),
	}
}

func (BillingInvoiceSplitLineGroup) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("billing_invoice_lines", BillingInvoiceLine.Type),
		edge.From("subscription", Subscription.Type).
			Ref("billing_split_line_groups").
			Field("subscription_id").
			Unique().
			Immutable(),
		edge.From("subscription_phase", SubscriptionPhase.Type).
			Ref("billing_split_line_groups").
			Field("subscription_phase_id").
			Unique().
			Immutable(),
		edge.From("subscription_item", SubscriptionItem.Type).
			Ref("billing_split_line_groups").
			Field("subscription_item_id").
			Unique().
			Immutable(),
	}
}

// TODO[later]: Rename to BillingInvoiceLineUsageDiscount
type BillingInvoiceLineDiscount struct {
	ent.Schema
}

func (BillingInvoiceLineDiscount) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
		BillingInvoiceLineDiscountBase{},
	}
}

func (BillingInvoiceLineDiscount) Fields() []ent.Field {
	return []ent.Field{
		field.Other("amount", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),

		field.Other("rounding_amount", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}).
			Optional().
			Nillable(),

		// TODO: Ent has issues with custom value scanners from mixins, so this is a duplicate
		// TODO[later]: Rename to reason_details (same time as the DB table rename, as this is breaking either ways)
		field.String("source_discount").
			GoType(&billing.DiscountReason{}).
			ValueScanner(BillingDiscountReasonValueScanner).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Optional().
			Nillable(),

		// Deprecated fields
		field.String("type").
			Optional().
			Nillable().
			Deprecated("due to split of amount and usage discount tables"),

		field.Other("quantity", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}).
			Optional().
			Nillable().
			Deprecated("due to split of amount and usage discount tables"),

		field.Other("pre_line_period_quantity", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}).
			Optional().
			Nillable().
			Deprecated("due to split of amount and usage discount tables"),
	}
}

func (BillingInvoiceLineDiscount) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "line_id"),
		index.Fields("namespace", "line_id", "child_unique_reference_id").
			Annotations(
				entsql.IndexWhere("child_unique_reference_id IS NOT NULL AND deleted_at IS NULL"),
			).
			Unique(),
	}
}

func (BillingInvoiceLineDiscount) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("billing_invoice_line", BillingInvoiceLine.Type).
			Ref("line_amount_discounts").
			Field("line_id").
			Unique().
			Required(),
	}
}

type BillingInvoiceLineUsageDiscount struct {
	ent.Schema
}

func (BillingInvoiceLineUsageDiscount) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
		BillingInvoiceLineDiscountBase{},
	}
}

func (BillingInvoiceLineUsageDiscount) Fields() []ent.Field {
	return []ent.Field{
		field.Other("quantity", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),

		field.Other("pre_line_period_quantity", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}).
			Optional().
			Nillable(),

		// TODO: Ent has issues with custom value scanners from mixins, so this is a duplicate
		field.String("reason_details").
			GoType(&billing.DiscountReason{}).
			ValueScanner(BillingDiscountReasonValueScanner).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Optional().
			Nillable(),
	}
}

func (BillingInvoiceLineUsageDiscount) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "line_id"),
		index.Fields("namespace", "line_id", "child_unique_reference_id").
			Annotations(
				entsql.IndexWhere("child_unique_reference_id IS NOT NULL AND deleted_at IS NULL"),
			).
			Unique(),
	}
}

func (BillingInvoiceLineUsageDiscount) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("billing_invoice_line", BillingInvoiceLine.Type).
			Ref("line_usage_discounts").
			Field("line_id").
			Unique().
			Required(),
	}
}

type BillingInvoice struct {
	ent.Schema
}

func (BillingInvoice) Mixin() []ent.Mixin {
	return []ent.Mixin{
		// This cannot be a resource mixin as the invoice doesn't have a name field
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.MetadataMixin{},
		entutils.TimeMixin{},
		entutils.CustomerAddressMixin{
			FieldPrefix: "supplier",
		},
		entutils.CustomerAddressMixin{
			FieldPrefix: "customer",
		},
		TotalsMixin{},
	}
}

func (BillingInvoice) Fields() []ent.Field {
	return []ent.Field{
		// Customer/supplier
		field.String("supplier_name").
			NotEmpty(),

		field.String("supplier_tax_code").
			Optional().
			Nillable(),

		field.String("customer_key").
			Optional().
			Nillable(),

		field.String("customer_name").
			NotEmpty(),

		field.JSON("customer_usage_attribution", &billing.VersionedCustomerUsageAttribution{}).
			Optional(),

		// Invoice number
		field.String("number"),

		field.Enum("type").
			GoType(billing.InvoiceType("")),

		field.String("description").
			Optional().
			Nillable(),

		field.String("customer_id").
			NotEmpty().
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Immutable(),

		field.String("source_billing_profile_id").
			NotEmpty().
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}),
		field.Time("voided_at").
			Optional().
			Nillable(),

		// issued_at can be in the future in case of pre-issuing invoices
		field.Time("issued_at").
			Optional().
			Nillable(),

		field.Time("sent_to_customer_at").
			Optional().
			Nillable(),

		field.Time("draft_until").
			Optional().
			Nillable(),

		field.Time("quantity_snapshoted_at").
			Optional().
			Nillable(),

		field.String("currency").
			GoType(currencyx.Code("")).
			NotEmpty().
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "varchar(3)",
			}),

		field.Time("due_at").
			Optional().
			Nillable(),

		field.Enum("status").
			GoType(billing.InvoiceStatus("")),

		field.JSON("status_details_cache", billing.InvoiceStatusDetails{}).
			Optional(),

		// Cloned profile settings
		field.String("workflow_config_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}),

		field.String("tax_app_id").
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}),
		field.String("invoicing_app_id").
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}),
		field.String("payment_app_id").
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}),

		// ID of the line in the external invoicing app
		// For example, Stripe invoice ID
		field.String("invoicing_app_external_id").
			Optional().
			Nillable(),

		// ID of the payment in the external invoicing app
		// For example, Stripe payment intent ID
		field.String("payment_app_external_id").
			Optional().
			Nillable(),

		// ID of the tax in the external invoicing app
		// For example, Stripe tax calculation ID
		field.String("tax_app_external_id").
			Optional().
			Nillable(),

		// These fields are optional as they are calculated from the invoice lines, which might not
		// be present on an invoice.
		field.Time("period_start").
			Optional().
			Nillable(),

		field.Time("period_end").
			Optional().
			Nillable(),

		// The timestamp set in the collection_at field defines when new draft invoice is need to be created
		// from line-items available on the gathering invoice. It is defaulted to time.Now() on creation.
		field.Time("collection_at").
			Optional().
			Default(clock.Now),

		// This is the timestamp the invoice first entered the Payment Processing State (InvoiceStatusPaymentProcessingPending).
		// This is relevant as we later use this to determine stale-ness and guard against fraud.
		field.Time("payment_processing_entered_at").
			Optional().
			Nillable(),
	}
}

func (BillingInvoice) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "id"),
		index.Fields("namespace", "customer_id"),
		index.Fields("namespace", "status"),
		index.Fields("namespace", "period_start"),
		index.Fields("namespace", "created_at"),
		index.Fields("namespace", "updated_at"),
		index.Fields("namespace", "issued_at"),
		index.Fields("status_details_cache").
			Annotations(
				entsql.IndexTypes(map[string]string{
					dialect.Postgres: "GIN",
				}),
			),
	}
}

func (BillingInvoice) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("source_billing_profile", BillingProfile.Type).
			Ref("billing_invoices").
			Field("source_billing_profile_id").
			Required().
			Unique().
			Immutable(), // Billing profile changes are forbidden => invoice must be voided in this case
		edge.From("billing_workflow_config", BillingWorkflowConfig.Type).
			Ref("billing_invoices").
			Field("workflow_config_id").
			Unique().
			Required(),
		edge.To("billing_invoice_lines", BillingInvoiceLine.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("billing_invoice_validation_issues", BillingInvoiceValidationIssue.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.From("billing_invoice_customer", Customer.Type).
			Ref("billing_invoice").
			Field("customer_id").
			Unique().
			Required().
			Immutable(),
		edge.From("tax_app", App.Type).
			Ref("billing_invoice_tax_app").
			Field("tax_app_id").
			Unique().
			Immutable().
			Required(),
		edge.From("invoicing_app", App.Type).
			Ref("billing_invoice_invoicing_app").
			Field("invoicing_app_id").
			Unique().
			Immutable().
			Required(),
		edge.From("payment_app", App.Type).
			Ref("billing_invoice_payment_app").
			Field("payment_app_id").
			Unique().
			Immutable().
			Required(),
	}
}

type BillingInvoiceValidationIssue struct {
	ent.Schema
}

func (BillingInvoiceValidationIssue) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
	}
}

func (BillingInvoiceValidationIssue) Fields() []ent.Field {
	return []ent.Field{
		field.String("invoice_id").
			NotEmpty().
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}),

		field.Enum("severity").
			GoType(billing.ValidationIssueSeverity("")),

		field.String("code").
			Nillable().
			Optional(),

		field.String("message").
			NotEmpty(),

		// Note: field is conflicting with the ent builtin methods, so we use path instead
		field.String("path").
			Optional().
			Nillable(),

		field.String("component"),

		field.Bytes("dedupe_hash").
			MinLen(32).
			MaxLen(32),
	}
}

func (BillingInvoiceValidationIssue) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "invoice_id", "dedupe_hash").Unique(),
	}
}

func (BillingInvoiceValidationIssue) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("billing_invoice", BillingInvoice.Type).
			Ref("billing_invoice_validation_issues").
			Field("invoice_id").
			Unique().
			Required(),
	}
}

type BillingSequenceNumbers struct {
	ent.Schema
}

func (BillingSequenceNumbers) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.NamespaceMixin{},
	}
}

func (BillingSequenceNumbers) Fields() []ent.Field {
	return []ent.Field{
		field.String("scope"),
		field.Other("last", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}),
	}
}

func (BillingSequenceNumbers) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "scope").Unique(),
	}
}

type BillingCustomerLock struct {
	ent.Schema
}

func (BillingCustomerLock) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.NamespaceMixin{},
		entutils.IDMixin{},
	}
}

func (BillingCustomerLock) Fields() []ent.Field {
	return []ent.Field{
		field.String("customer_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}),
	}
}

func (BillingCustomerLock) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "customer_id").Unique(),
	}
}
