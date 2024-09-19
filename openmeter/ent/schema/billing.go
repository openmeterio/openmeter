package schema

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/invoice"
	"github.com/openmeterio/openmeter/openmeter/billing/provider"
	"github.com/openmeterio/openmeter/openmeter/billing/provider/openmetersandbox"
	"github.com/openmeterio/openmeter/openmeter/billing/provider/stripe"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/timezone"
)

type BillingProfile struct {
	ent.Schema
}

func (BillingProfile) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
	}
}

func (BillingProfile) Fields() []ent.Field {
	return []ent.Field{
		field.String("key").
			NotEmpty().
			Immutable(),
		field.String("provider_config").
			GoType(provider.Configuration{}).
			ValueScanner(ProviderConfigValueScanner).
			SchemaType(map[string]string{
				"postgres": "jsonb",
			}),
		field.String("workflow_config_id").
			NotEmpty(),
		field.String("timezone").
			GoType(timezone.Timezone("")),
		field.Bool("default").
			Default(false),
	}
}

func (BillingProfile) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("billing_invoices", BillingInvoice.Type),
		edge.From("billing_workflow_config", BillingWorkflowConfig.Type).
			Ref("billing_profile").
			Field("workflow_config_id").
			Unique().
			Required(),
	}
}

func (BillingProfile) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "key"),
		index.Fields("namespace", "id"),
		index.Fields("namespace", "default"),
	}
}

type providerConfigSerde[T any] struct {
	provider.Meta

	Config T `json:"config"`
}

var ProviderConfigValueScanner = field.ValueScannerFunc[provider.Configuration, *sql.NullString]{
	V: func(config provider.Configuration) (driver.Value, error) {
		switch config.Type {
		case provider.TypeOpenMeter:
			return json.Marshal(providerConfigSerde[openmetersandbox.Config]{
				Meta:   provider.Meta{Type: provider.TypeOpenMeter},
				Config: config.OpenMeterSandbox,
			})
		case provider.TypeStripe:
			return json.Marshal(providerConfigSerde[stripe.Config]{
				Meta:   provider.Meta{Type: provider.TypeStripe},
				Config: config.Stripe,
			})
		default:
			return nil, fmt.Errorf("unknown backend type: %s", config.Type)
		}
	},
	S: func(ns *sql.NullString) (provider.Configuration, error) {
		if !ns.Valid {
			return provider.Configuration{}, errors.New("backend config is null")
		}

		data := []byte(ns.String)

		var meta provider.Meta
		if err := json.Unmarshal(data, &meta); err != nil {
			return provider.Configuration{}, err
		}

		switch meta.Type {
		case provider.TypeOpenMeter:
			serde := providerConfigSerde[openmetersandbox.Config]{}

			if err := json.Unmarshal(data, &serde); err != nil {
				return provider.Configuration{}, err
			}

			return provider.Configuration{
				Meta:             serde.Meta,
				OpenMeterSandbox: serde.Config,
			}, nil
		case provider.TypeStripe:
			serde := providerConfigSerde[stripe.Config]{}

			if err := json.Unmarshal(data, &serde); err != nil {
				return provider.Configuration{}, err
			}

			return provider.Configuration{
				Meta:   serde.Meta,
				Stripe: serde.Config,
			}, nil
		default:
			return provider.Configuration{}, fmt.Errorf("unknown backend type: %s", meta.Type)
		}
	},
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
		field.Enum("collection_alignment").
			GoType(billing.AlignmentKind("")),

		// TODO: later we will add more alignment details here (e.g. monthly, yearly, etc.)

		field.Int64("collection_period_seconds"),

		field.Bool("invoice_auto_advance").
			Nillable(),

		field.Int64("invoice_draft_period_seconds"),

		field.Int64("invoice_due_after_days").
			Optional().
			Comment("Optional as if we have auto collection we don't need this"),

		field.Enum("invoice_collection_method").
			GoType(billing.CollectionMethod("")),

		field.Enum("invoice_item_resolution").
			GoType(billing.GranualityResolution("")),

		field.Bool("invoice_item_per_subject").
			Default(false),
	}
}

func (BillingWorkflowConfig) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "id"),
	}
}

func (BillingWorkflowConfig) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("billing_invoices", BillingInvoice.Type),
		edge.To("billing_profile", BillingProfile.Type),
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
			GoType(currencyx.Code("")).
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
		// TODO: resource mixin
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
		entutils.MetadataAnnotationsMixin{},

		entutils.CustomerAddressMixin{
			FieldPrefix: "billing",
		},
	}
}

func (BillingInvoice) Fields() []ent.Field {
	return []ent.Field{
		// TODO: this is a mess!
		field.String("key").
			NotEmpty().
			Immutable(),
		field.String("key_series").
			NotEmpty().
			Immutable(),
		field.String("key_number"). // TODO:?!?! int maybe?
						NotEmpty().
						Immutable(),

		field.Enum("type").
			GoType(invoice.InvoiceType("")).
			Immutable(),

		// Customer related fields
		field.String("customer_id").
			NotEmpty().
			SchemaType(map[string]string{
				"postgres": "char(26)",
			}).
			Immutable(),
		field.Bool("customer_snapshot_taken").
			Default(false),
		field.String("customer_name").
			Optional().Nillable(),
		field.String("customer_primary_email").Optional().Nillable(),
		field.String("billing_profile_id").
			NotEmpty().
			Immutable().
			SchemaType(map[string]string{
				"postgres": "char(26)",
			}),

		// Lifecycle
		field.Strings("preceding_invoice_ids").
			Optional().
			SchemaType(map[string]string{
				"postgres": "char(26)[]",
			}),
		field.Time("issued_at").
			Optional().
			Nillable().
			Comment("issued_at specifies the date when the invoice was issued. Not set if the invoice is not yet issued (e.g. in draft state)."),
		field.Time("voided_at").
			Optional().
			Nillable(),
		field.String("currency").
			GoType(currencyx.Code("")).
			NotEmpty().
			Immutable().
			SchemaType(map[string]string{
				"postgres": "varchar(3)",
			}),
		field.String("timezone").
			GoType(timezone.Timezone("")),

		field.Time("due_date").
			Optional().
			Nillable().
			Comment(`due_date specifies the date when the invoice is due to be paid. Not set if the invoice is paid immediately.

			Stored as a timestamp, so that if the billing is done in a different timezone than the customer's timezone we can still calculate this
			in a consistent manner.

			For invoicing only the date part is relevant, always returned in the timezone set in the invoice.
			`),
		field.Enum("status").
			GoType(invoice.InvoiceStatus("")),

		field.String("provider_config").
			GoType(provider.Configuration{}).
			ValueScanner(ProviderConfigValueScanner).
			SchemaType(map[string]string{
				"postgres": "jsonb",
			}),

		field.String("workflow_config_id").
			SchemaType(map[string]string{
				"postgres": "char(26)",
			}),
		field.String("provider_reference").
			GoType(provider.Reference{}).
			ValueScanner(ProviderReferenceValueScanner).
			SchemaType(map[string]string{
				"postgres": "jsonb",
			}),

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
			Required().
			Unique(),
		edge.To("billing_invoice_items", BillingInvoiceItem.Type),
		edge.From("customer", Customer.Type).
			Ref("billing_invoices").
			Field("customer_id").
			Required().
			Unique().
			Immutable(),
	}
}

type providerReferenceSerde[T any] struct {
	provider.Meta

	Reference T `json:"ref"`
}

var ProviderReferenceValueScanner = field.ValueScannerFunc[provider.Reference, *sql.NullString]{
	V: func(ref provider.Reference) (driver.Value, error) {
		switch ref.Type {
		case provider.TypeOpenMeter:
			return json.Marshal(providerReferenceSerde[provider.OpenMeterReference]{
				Meta:      provider.Meta{Type: provider.TypeOpenMeter},
				Reference: ref.OpenMeter,
			})
		case provider.TypeStripe:
			return json.Marshal(providerReferenceSerde[provider.StripeReference]{
				Meta:      provider.Meta{Type: provider.TypeStripe},
				Reference: ref.Stripe,
			})
		default:
			return nil, fmt.Errorf("unknown backend type: %s", ref.Type)
		}
	},
	S: func(ns *sql.NullString) (provider.Reference, error) {
		if !ns.Valid {
			return provider.Reference{}, errors.New("backend config is null")
		}

		data := []byte(ns.String)

		var meta provider.Meta
		if err := json.Unmarshal(data, &meta); err != nil {
			return provider.Reference{}, err
		}

		switch meta.Type {
		case provider.TypeOpenMeter:
			serde := providerReferenceSerde[provider.OpenMeterReference]{}

			if err := json.Unmarshal(data, &serde); err != nil {
				return provider.Reference{}, err
			}

			return provider.Reference{
				Meta:      serde.Meta,
				OpenMeter: serde.Reference,
			}, nil
		case provider.TypeStripe:
			serde := providerConfigSerde[provider.StripeReference]{}

			if err := json.Unmarshal(data, &serde); err != nil {
				return provider.Reference{}, err
			}

			return provider.Reference{
				Meta:   serde.Meta,
				Stripe: serde.Config,
			}, nil
		default:
			return provider.Reference{}, fmt.Errorf("unknown backend type: %s", meta.Type)
		}
	},
}
