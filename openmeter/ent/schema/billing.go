package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/invoice"
	"github.com/openmeterio/openmeter/openmeter/billing/provider"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type BillingProfile struct {
	ent.Schema
}

func (BillingProfile) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.UniqueResourceMixin{},
		entutils.CustomerAddressMixin{
			FieldPrefix: "supplier",
		},
	}
}

func (BillingProfile) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("tax_provider").GoType(provider.TaxProvider("")),
		field.Enum("invoicing_provider").GoType(provider.InvoicingProvider("")),
		field.Enum("payment_provider").GoType(provider.PaymentProvider("")),
		field.String("workflow_config_id").
			NotEmpty(),
		field.Bool("default").
			Default(false),
		field.String("supplier_name").
			NotEmpty(),
	}
}

func (BillingProfile) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("billing_invoices", BillingInvoice.Type),
		edge.From("workflow_config", BillingWorkflowConfig.Type).
			Ref("billing_profile").
			Field("workflow_config_id").
			Unique().
			Required(),
	}
}

func (BillingProfile) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "default", "deleted_at").Unique(),
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

		field.Int64("item_collection_period_seconds"),

		field.Bool("invoice_auto_advance"),

		field.Int64("invoice_draft_period_seconds"),

		field.Int64("invoice_due_after_seconds"),

		field.Enum("invoice_collection_method").
			GoType(billing.CollectionMethod("")),

		field.Enum("invoice_item_resolution").
			GoType(billing.GranularityResolution("")),

		field.Bool("invoice_item_per_subject"),
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

type BillingInvoiceItem struct {
	ent.Schema
}

func (BillingInvoiceItem) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
		entutils.MetadataAnnotationsMixin{},
	}
}

func (BillingInvoiceItem) Fields() []ent.Field {
	return []ent.Field{
		field.String("invoice_id").
			Optional().
			SchemaType(map[string]string{
				"postgres": "char(26)",
			}),
		field.String("customer_id").
			NotEmpty().
			Immutable().
			SchemaType(map[string]string{
				"postgres": "char(26)",
			}),

		field.Time("period_start"),
		field.Time("period_end"),
		field.Time("invoice_at"),

		// TODO[dependency]: overrides (as soon as plan override entities are ready)

		field.Other("quantity", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				"postgres": "numeric",
			}),
		field.Other("unit_price", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				"postgres": "numeric",
			}),
		field.String("currency").
			NotEmpty().
			Immutable().
			SchemaType(map[string]string{
				"postgres": "varchar(3)",
			}),
		field.JSON("tax_code_override", invoice.TaxOverrides{}).
			SchemaType(map[string]string{
				"postgres": "jsonb",
			}),
	}
}

func (BillingInvoiceItem) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "id"),
		index.Fields("namespace", "invoice_id"),
		index.Fields("namespace", "customer_id"),
	}
}

func (BillingInvoiceItem) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("billing_invoice", BillingInvoice.Type).
			Ref("billing_invoice_items").
			Field("invoice_id").
			Unique(),
		// TODO[dependency]: Customer edge, as soon as customer entities are ready

	}
}

type BillingInvoice struct {
	ent.Schema
}

func (BillingInvoice) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
		entutils.MetadataAnnotationsMixin{},
	}
}

func (BillingInvoice) Fields() []ent.Field {
	return []ent.Field{
		field.String("key").
			NotEmpty().
			Immutable(),
		field.String("customer_id").
			NotEmpty().
			SchemaType(map[string]string{
				"postgres": "char(26)",
			}).
			Immutable(),
		field.String("billing_profile_id").
			NotEmpty().
			Immutable().
			SchemaType(map[string]string{
				"postgres": "char(26)",
			}),
		field.Time("voided_at").
			Optional(),
		field.String("currency").
			NotEmpty().
			Immutable().
			SchemaType(map[string]string{
				"postgres": "varchar(3)",
			}),
		field.Other("total_amount", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				"postgres": "numeric",
			}),
		field.Time("due_date"),
		field.Enum("status").
			GoType(invoice.InvoiceStatus("")),

		field.Enum("tax_provider").GoType(provider.TaxProvider("")).Optional().Nillable(),
		field.Enum("invoicing_provider").GoType(provider.InvoicingProvider("")).Optional().Nillable(),
		field.Enum("payment_provider").GoType(provider.PaymentProvider("")).Optional().Nillable(),

		field.String("workflow_config_id").
			SchemaType(map[string]string{
				"postgres": "char(26)",
			}),

		// TODO[later]: Add either provider annotations or typed provider status fields

		field.Time("period_start"),
		field.Time("period_end"),
	}
}

func (BillingInvoice) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "id"),
		index.Fields("namespace", "key"),
		index.Fields("namespace", "customer_id"),
		index.Fields("namespace", "due_date"),
		index.Fields("namespace", "status"),
	}
}

func (BillingInvoice) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("billing_profile", BillingProfile.Type).
			Ref("billing_invoices").
			Field("billing_profile_id").
			Required().
			Unique().
			Immutable(), // Billing profile changes are forbidden => invoice must be voided in this case
		edge.From("billing_workflow_config", BillingWorkflowConfig.Type).
			Ref("billing_invoices").
			Field("workflow_config_id").
			Unique().
			Required(),
		edge.To("billing_invoice_items", BillingInvoiceItem.Type),
	}
}
