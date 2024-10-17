package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datex"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/timezone"
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
		edge.From("workflow_invoicing_app", BillingWorkflowConfig.Type).
			Ref("workflow_invoicing_app").
			Field("invoicing_app_id").
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
		// Defaults to UTC if not specified, customer TZ overrides this
		field.String("timezone").GoType(timezone.Timezone("")).Optional().Nillable(),

		// TODO: later we will add more alignment details here (e.g. monthly, yearly, etc.)
		field.Enum("collection_alignment").
			GoType(billing.AlignmentKind("")),

		field.String("item_collection_period").GoType(datex.ISOString("")),

		field.Bool("invoice_auto_advance"),

		field.String("invoice_draft_period").GoType(datex.ISOString("")),

		field.String("invoice_due_after").GoType(datex.ISOString("")),

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
		// TODO: later we will add more alignment details here (e.g. monthly, yearly, etc.)
		field.Enum("collection_alignment").
			GoType(billing.AlignmentKind("")).
			Optional().
			Nillable(),

		field.String("item_collection_period").
			GoType(datex.ISOString("")).
			Optional().
			Nillable(),

		field.Bool("invoice_auto_advance").
			Optional().
			Nillable(),

		field.String("invoice_draft_period").
			GoType(datex.ISOString("")).
			Optional().
			Nillable(),

		field.String("invoice_due_after").
			GoType(datex.ISOString("")).
			Optional().
			Nillable(),

		field.Enum("invoice_collection_method").
			GoType(billing.CollectionMethod("")).
			Optional().
			Nillable(),

		field.Enum("invoice_item_resolution").
			GoType(billing.GranularityResolution("")).
			Optional().
			Nillable(),

		field.Bool("invoice_item_per_subject").
			Optional().
			Nillable(),
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
			Nillable().
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

		field.Enum("type").
			GoType(billing.InvoiceItemType("")),

		field.String("name").
			NotEmpty(),

		// Quantity is only required for static items
		field.Other("quantity", alpacadecimal.Decimal{}).
			Optional().
			Nillable().
			SchemaType(map[string]string{
				"postgres": "numeric",
			}),
		field.Other("unit_price", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				"postgres": "numeric",
			}),
		field.String("currency").
			GoType(currencyx.Code("")).
			NotEmpty().
			Immutable().
			SchemaType(map[string]string{
				"postgres": "varchar(3)",
			}),
		field.JSON("tax_code_override", billing.TaxOverrides{}).
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
		// Invoice number
		field.String("series").
			Optional().
			Nillable(),
		field.String("code").
			Optional().
			Nillable(),

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

		field.Time("due_date"),
		field.Enum("status").
			GoType(billing.InvoiceStatus("")),

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
		index.Fields("namespace", "customer_id"),
		index.Fields("namespace", "due_date"),
		index.Fields("namespace", "status"),
		// Some countries require per seller uniqueness, but that's something we can't enforce here
		index.Fields("namespace", "customer_id", "series", "code").Unique(),
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
